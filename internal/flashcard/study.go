package flashcard

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-spaced-repetition/go-fsrs/v4"
)

const sessionSize = 20

var (
	ErrSessionNotFound = errors.New("study session not found")
	ErrAlreadyAnswered = errors.New("card has already been answered")
	ErrNoReviewWords   = errors.New("No words to review yet. Start a learning session first.")
	ErrNoWords         = errors.New("No new words or due reviews are available. Your next review will appear when it is ready.")
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// All session mutations lock the user row. Double clicks and multiple tabs cannot
// allocate competing sessions or race the progress update.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&id)
}

func (r *Repository) CreateSession(ctx context.Context, userID string, mode SessionMode) (Session, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockUser(ctx, tx, userID); err != nil {
		return Session{}, err
	}
	if existing, ok, err := unfinishedSession(ctx, tx, userID, mode); err != nil {
		return Session{}, err
	} else if ok {
		return existing, nil
	}
	if _, err = tx.Exec(ctx, `INSERT INTO study_settings(user_id) VALUES($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return Session{}, err
	}
	var target float64
	if err = tx.QueryRow(ctx, `SELECT difficulty_target FROM study_settings WHERE user_id=$1`, userID).Scan(&target); err != nil {
		return Session{}, err
	}
	var cards []Card
	if mode == ModeReview {
		cards, err = selectReviews(ctx, tx, userID, sessionSize, true)
		if err != nil {
			return Session{}, err
		}
		if len(cards) == 0 {
			return Session{}, ErrNoReviewWords
		}
	} else {
		reviews, err := selectReviews(ctx, tx, userID, sessionSize, false)
		if err != nil {
			return Session{}, err
		}
		reviewCount := min(len(reviews), sessionSize/2)
		newCards, err := selectNewCards(ctx, tx, userID, target, sessionSize-reviewCount)
		if err != nil {
			return Session{}, err
		}
		reviewCount = min(len(reviews), sessionSize-len(newCards))
		cards = interleaveCards(reviews[:reviewCount], newCards)
	}
	if len(cards) == 0 {
		return Session{}, ErrNoWords
	}
	session := Session{ID: newID(), Mode: mode, Target: target, Total: len(cards), Cards: cards}
	if _, err = tx.Exec(ctx, `INSERT INTO flashcard_sessions(id,user_id,mode,difficulty_target) VALUES($1,$2,$3,$4)`, session.ID, userID, mode, target); err != nil {
		return Session{}, err
	}
	for position, card := range cards {
		if _, err = tx.Exec(ctx, `INSERT INTO flashcard_session_cards(session_id,word_id,position,was_review) VALUES($1,$2,$3,$4)`, session.ID, card.ID, position, card.Review); err != nil {
			return Session{}, err
		}
	}
	// Merely reserving a card must not create learning progress.
	if err = tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return session, nil
}

func unfinishedSession(ctx context.Context, tx pgx.Tx, userID string, mode SessionMode) (Session, bool, error) {
	var session Session
	err := tx.QueryRow(ctx, `SELECT id,mode,COALESCE(difficulty_target,15) FROM flashcard_sessions s
        WHERE user_id=$1 AND mode=$2 AND completed_at IS NULL
        AND EXISTS(SELECT 1 FROM flashcard_session_cards c WHERE c.session_id=s.id AND c.answered_at IS NULL)
        ORDER BY created_at DESC LIMIT 1`, userID, mode).Scan(&session.ID, &session.Mode, &session.Target)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	err = tx.QueryRow(ctx, `SELECT count(*),count(answered_at),count(*) FILTER(WHERE knew_answer) FROM flashcard_session_cards WHERE session_id=$1`, session.ID).Scan(&session.Total, &session.Answered, &session.Recalled)
	if err != nil {
		return Session{}, false, err
	}
	rows, err := tx.Query(ctx, `SELECT w.id,w.word,w.difficulty_score,CASE w.part_of_speech WHEN 'n' THEN 'noun' ELSE 'verb' END,
        EXISTS(SELECT 1 FROM user_word_progress p WHERE p.user_id=$2 AND p.word_id=w.id AND p.reps>0)
        FROM flashcard_session_cards c JOIN words w ON w.id=c.word_id
        WHERE c.session_id=$1 AND c.answered_at IS NULL ORDER BY c.position`, session.ID, userID)
	if err != nil {
		return Session{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var card Card
		if err = rows.Scan(&card.ID, &card.Word, &card.Difficulty, &card.PartOfSpeech, &card.Review); err != nil {
			return Session{}, false, err
		}
		session.Cards = append(session.Cards, card)
	}
	return session, true, rows.Err()
}

func selectReviews(ctx context.Context, tx pgx.Tx, userID string, limit int, includeLearning bool) ([]Card, error) {
	rows, err := tx.Query(ctx, `SELECT w.id,w.word,w.difficulty_score,CASE w.part_of_speech WHEN 'n' THEN 'noun' ELSE 'verb' END
        FROM user_word_progress p JOIN words w ON w.id=p.word_id
        WHERE p.user_id=$1 AND p.reps>0 AND w.part_of_speech IN ('n','v')
        AND (p.due<=now() OR ($3 AND NOT (`+learnedSQL("p")+`)))
        ORDER BY (p.due<=now()) DESC,p.unknown_count DESC,p.due,w.id LIMIT $2`, userID, limit, includeLearning)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []Card
	for rows.Next() {
		var card Card
		if err = rows.Scan(&card.ID, &card.Word, &card.Difficulty, &card.PartOfSpeech); err != nil {
			return nil, err
		}
		card.Review = true
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func selectNewCards(ctx context.Context, tx pgx.Tx, userID string, target float64, limit int) ([]Card, error) {
	// Score bands are an app curriculum heuristic. Prefer the comfortable band,
	// then easier gaps, then the nearest harder words. Frequency breaks ties.
	rows, err := tx.Query(ctx, `WITH candidates AS (
        SELECT w.*, row_number() OVER(PARTITION BY w.part_of_speech ORDER BY
            CASE WHEN w.difficulty_score BETWEEN GREATEST(0,$2::float8-8) AND $2+3 THEN 0
                 WHEN w.difficulty_score<$2-8 THEN 1 ELSE 2 END,
            abs(w.difficulty_score-($2-3)),w.frequency_zipf DESC NULLS LAST,w.id) AS position
        FROM words w WHERE w.active AND w.part_of_speech IN ('n','v') AND w.difficulty_score IS NOT NULL
        AND NOT EXISTS(SELECT 1 FROM user_word_progress p WHERE p.user_id=$1 AND p.word_id=w.id AND p.reps>0)
        AND NOT EXISTS(SELECT 1 FROM flashcard_session_cards c JOIN flashcard_sessions s ON s.id=c.session_id
            WHERE s.user_id=$1 AND s.completed_at IS NULL AND c.answered_at IS NULL AND c.word_id=w.id)
    ) SELECT id,word,difficulty_score,CASE part_of_speech WHEN 'n' THEN 'noun' ELSE 'verb' END
        FROM candidates ORDER BY position,part_of_speech,id LIMIT $3`, userID, target, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []Card
	for rows.Next() {
		var card Card
		if err = rows.Scan(&card.ID, &card.Word, &card.Difficulty, &card.PartOfSpeech); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func interleaveCards(reviews, newCards []Card) []Card {
	cards := make([]Card, 0, len(reviews)+len(newCards))
	for i := 0; i < len(reviews) || i < len(newCards); i++ {
		if i < len(reviews) {
			cards = append(cards, reviews[i])
		}
		if i < len(newCards) {
			cards = append(cards, newCards[i])
		}
	}
	return cards
}

func (r *Repository) Answer(ctx context.Context, userID, sessionID string, wordID int64, known bool) (WordProgress, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return WordProgress{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockUser(ctx, tx, userID); err != nil {
		return WordProgress{}, err
	}
	var answeredAt sql.NullTime
	var previousKnown sql.NullBool
	err = tx.QueryRow(ctx, `SELECT c.answered_at,c.knew_answer FROM flashcard_session_cards c JOIN flashcard_sessions s ON s.id=c.session_id WHERE c.session_id=$1 AND c.word_id=$2 AND s.user_id=$3 FOR UPDATE`, sessionID, wordID, userID).Scan(&answeredAt, &previousKnown)
	if errors.Is(err, pgx.ErrNoRows) {
		return WordProgress{}, ErrSessionNotFound
	}
	if err != nil {
		return WordProgress{}, err
	}
	if answeredAt.Valid {
		if previousKnown.Valid && previousKnown.Bool == known {
			existing, progress, err := loadFSRSCard(ctx, tx, userID, wordID)
			progress.State = existing.State.String()
			progress.Due = existing.Due.Format(time.RFC3339)
			return progress, err
		}
		return WordProgress{}, ErrAlreadyAnswered
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_word_progress(user_id,word_id,due) VALUES($1,$2,now()) ON CONFLICT DO NOTHING`, userID, wordID); err != nil {
		return WordProgress{}, err
	}

	card, progress, err := loadFSRSCard(ctx, tx, userID, wordID)
	if err != nil {
		return WordProgress{}, err
	}
	now := time.Now().UTC()
	rating := fsrs.Again
	if known {
		rating = fsrs.Good
	}
	result, err := fsrs.NewFSRS(fsrs.DefaultParam()).Next(card, now, rating)
	if err != nil {
		return WordProgress{}, fmt.Errorf("schedule review: %w", err)
	}
	next := result.Card
	_, err = tx.Exec(ctx, `UPDATE user_word_progress SET due=$3,stability=$4,difficulty=$5,scheduled_days=$6,reps=$7,lapses=$8,state=$9,last_review=$10,remaining_steps=$11,known_count=known_count+$12,unknown_count=unknown_count+$13,updated_at=$10 WHERE user_id=$1 AND word_id=$2`,
		userID, wordID, next.Due, next.Stability, next.Difficulty, next.ScheduledDays, next.Reps, next.Lapses, next.State, next.LastReview, next.RemainingSteps, boolInt(known), boolInt(!known))
	if err != nil {
		return WordProgress{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE flashcard_session_cards SET answered_at=$4,knew_answer=$5 WHERE session_id=$1 AND word_id=$2 AND EXISTS(SELECT 1 FROM flashcard_sessions WHERE id=$1 AND user_id=$3)`, sessionID, wordID, userID, now, known); err != nil {
		return WordProgress{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO flashcard_review_logs(user_id,word_id,session_id,rating,reviewed_at,previous_state,previous_due,scheduled_days,stability,difficulty) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, userID, wordID, sessionID, rating, now, card.State, card.Due, next.ScheduledDays, next.Stability, next.Difficulty); err != nil {
		return WordProgress{}, err
	}
	tag, err := tx.Exec(ctx, `UPDATE flashcard_sessions SET completed_at=$2 WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM flashcard_session_cards WHERE session_id=$1 AND answered_at IS NULL)`, sessionID, now)
	if err != nil {
		return WordProgress{}, err
	}
	if tag.RowsAffected() > 0 {
		var mode SessionMode
		var answered, recalled int
		err = tx.QueryRow(ctx, `SELECT s.mode,count(*),count(*) FILTER(WHERE c.knew_answer) FROM flashcard_sessions s
            JOIN flashcard_session_cards c ON c.session_id=s.id WHERE s.id=$1 GROUP BY s.mode`, sessionID).Scan(&mode, &answered, &recalled)
		if err != nil {
			return WordProgress{}, err
		}
		if mode == ModeLearn {
			if _, err = tx.Exec(ctx, `INSERT INTO study_settings(user_id) VALUES($1) ON CONFLICT DO NOTHING`, userID); err != nil {
				return WordProgress{}, err
			}
			var target float64
			if err = tx.QueryRow(ctx, `SELECT difficulty_target FROM study_settings WHERE user_id=$1`, userID).Scan(&target); err != nil {
				return WordProgress{}, err
			}
			if _, err = tx.Exec(ctx, `UPDATE study_settings SET difficulty_target=$2,updated_at=now() WHERE user_id=$1`, userID, TargetAfterSession(target, answered, recalled)); err != nil {
				return WordProgress{}, err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return WordProgress{}, err
	}
	progress.State = next.State.String()
	progress.Due = next.Due.Format(time.RFC3339)
	if known {
		progress.KnownCount++
	} else {
		progress.UnknownCount++
	}
	return progress, nil
}

func loadFSRSCard(ctx context.Context, tx pgx.Tx, userID string, wordID int64) (fsrs.Card, WordProgress, error) {
	var card fsrs.Card
	var lastReview sql.NullTime
	var progress WordProgress
	err := tx.QueryRow(ctx, `SELECT p.due,p.stability,p.difficulty,p.scheduled_days,p.reps,p.lapses,p.state,p.last_review,p.remaining_steps,p.known_count,p.unknown_count,w.id,w.word,w.difficulty_score,CASE w.part_of_speech WHEN 'n' THEN 'noun' ELSE 'verb' END FROM user_word_progress p JOIN words w ON w.id=p.word_id WHERE p.user_id=$1 AND p.word_id=$2 FOR UPDATE`, userID, wordID).Scan(&card.Due, &card.Stability, &card.Difficulty, &card.ScheduledDays, &card.Reps, &card.Lapses, &card.State, &lastReview, &card.RemainingSteps, &progress.KnownCount, &progress.UnknownCount, &progress.ID, &progress.Word, &progress.Difficulty, &progress.PartOfSpeech)
	if err != nil {
		return card, progress, err
	}
	if lastReview.Valid {
		card.LastReview = lastReview.Time
	}
	return card, progress, nil
}

func (r *Repository) ListProgress(ctx context.Context, userID string, learned bool) ([]WordProgress, error) {
	query := `SELECT w.id,w.word,w.difficulty_score,CASE w.part_of_speech WHEN 'n' THEN 'noun' ELSE 'verb' END,
        p.state,p.known_count,p.unknown_count,p.due FROM user_word_progress p JOIN words w ON w.id=p.word_id
        WHERE p.user_id=$1 AND p.reps>0 AND (` + learnedSQL("p") + `)=$2 ORDER BY p.due,p.unknown_count DESC,w.id`
	rows, err := r.db.Query(ctx, query, userID, learned)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []WordProgress{}
	for rows.Next() {
		var item WordProgress
		var state fsrs.State
		var due time.Time
		if err := rows.Scan(&item.ID, &item.Word, &item.Difficulty, &item.PartOfSpeech, &state, &item.KnownCount, &item.UnknownCount, &due); err != nil {
			return nil, err
		}
		item.Review = true
		item.State = state.String()
		item.Due = due.Format(time.RFC3339)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) Overview(ctx context.Context, userID string) (Overview, error) {
	var out Overview
	err := r.db.QueryRow(ctx, `SELECT COALESCE((SELECT difficulty_target FROM study_settings WHERE user_id=$1),15),
        count(*) FILTER(WHERE NOT (`+learnedSQL("p")+`)),count(*) FILTER(WHERE `+learnedSQL("p")+`),
        count(*) FILTER(WHERE p.due<=now()),count(*) FILTER(WHERE p.due<=now() OR NOT (`+learnedSQL("p")+`)),
        min(p.due) FILTER(WHERE p.due>now()),
        (SELECT count(DISTINCT word_id) FROM flashcard_review_logs WHERE user_id=$1 AND reviewed_at>now()-interval '24 hours')
        FROM user_word_progress p WHERE p.user_id=$1 AND p.reps>0`, userID).Scan(&out.DifficultyTarget, &out.Learning, &out.Learned, &out.Due, &out.ReviewAvailable, &out.NextReview, &out.Practiced)
	return out, err
}

func (r *Repository) SaveSettings(ctx context.Context, userID string, settings Settings) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockUser(ctx, tx, userID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO study_settings(user_id,difficulty_target) VALUES($1,$2)
        ON CONFLICT(user_id) DO UPDATE SET difficulty_target=EXCLUDED.difficulty_target,updated_at=now()`, userID, settings.DifficultyTarget)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b)
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
