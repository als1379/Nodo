package flashcard

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"nodo/internal/database"
)

func TestStudyLifecycle(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL enables isolated PostgreSQL integration tests")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	schema := "study_test_" + newID()[:8]
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = root.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := root.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	user := newID()
	other := newID()
	for _, id := range []string{user, other} {
		if _, err = db.Exec(ctx, `INSERT INTO users(id,email,password_hash) VALUES($1,$2,'unused')`, id, id+"@test.invalid"); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(ctx, `INSERT INTO words(word,level,part_of_speech,curriculum_rank,source,difficulty_score,frequency_zipf)
        SELECT 'word'||i,'B2',CASE WHEN i%2=0 THEN 'n' ELSE 'v' END,i,'test',i,4 FROM generate_series(1,80) i`)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	sessions := make([]Session, 6)
	errs := make([]error, 6)
	var workers sync.WaitGroup
	for i := range sessions {
		workers.Go(func() { sessions[i], errs[i] = repo.CreateSession(ctx, user, ModeLearn) })
	}
	workers.Wait()
	for i, s := range sessions {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if s.ID != sessions[0].ID || s.Total != 20 || s.Answered != 0 {
			t.Fatalf("duplicate or invalid session: %+v", s)
		}
	}
	session := sessions[0]
	for _, card := range session.Cards {
		if card.Review || card.Difficulty == nil {
			t.Fatalf("invalid new card: %+v", card)
		}
	}
	// The source is deliberately B2; selection must use the 15-point target.
	if *session.Cards[0].Difficulty > 18 {
		t.Fatalf("selection ignored difficulty: %+v", session.Cards[0])
	}
	stats, err := repo.Overview(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Learning != 0 || stats.Due != 0 {
		t.Fatalf("reserved words counted as learned: %+v", stats)
	}
	if _, err = repo.CreateSession(ctx, user, ModeReview); !errors.Is(err, ErrNoReviewWords) {
		t.Fatalf("review included unseen cards: %v", err)
	}
	if _, err = repo.Answer(ctx, other, session.ID, session.Cards[0].ID, true); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-user answer: %v", err)
	}
	first, err := repo.Answer(ctx, user, session.ID, session.Cards[0].ID, false)
	if err != nil {
		t.Fatal(err)
	}
	again, err := repo.Answer(ctx, user, session.ID, session.Cards[0].ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.UnknownCount != 1 || again.UnknownCount != 1 {
		t.Fatal("retry double counted")
	}
	if _, err = repo.Answer(ctx, user, session.ID, session.Cards[0].ID, true); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("changed answer accepted: %v", err)
	}
	resumed, err := repo.CreateSession(ctx, user, ModeLearn)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Answered != 1 || resumed.Total != 20 || len(resumed.Cards) != 19 {
		t.Fatalf("resume lost counts: %+v", resumed)
	}
	review, err := repo.CreateSession(ctx, user, ModeReview)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Cards) != 1 || !review.Cards[0].Review || review.ID == session.ID {
		t.Fatalf("invalid review session: %+v", review)
	}
	library, err := repo.ListProgress(ctx, user, false)
	if err != nil || len(library) != 1 {
		t.Fatalf("invalid library: %d %v", len(library), err)
	}
	for _, card := range session.Cards[1:] {
		if _, err = repo.Answer(ctx, user, session.ID, card.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	stats, err = repo.Overview(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DifficultyTarget != 17 || stats.Practiced != 20 || stats.Learned != 0 {
		t.Fatalf("incorrect completion: %+v", stats)
	}
	// A mastered word still needs its due reviews, even after source retirement.
	_, err = db.Exec(ctx, `UPDATE user_word_progress SET state=2,known_count=3,scheduled_days=8,due=now()-interval '1 day' WHERE user_id=$1 AND word_id=$2`, user, session.Cards[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, `UPDATE words SET active=false WHERE id=$1`, session.Cards[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	stats, err = repo.Overview(ctx, user)
	if err != nil || stats.Learned != 1 || stats.Due < 1 {
		t.Fatalf("mastery/due mismatch: %+v %v", stats, err)
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	due, err := selectReviews(ctx, tx, user, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, card := range due {
		if card.ID == session.Cards[1].ID {
			found = true
		}
	}
	if !found {
		t.Fatal("due mastered word was omitted")
	}
}
