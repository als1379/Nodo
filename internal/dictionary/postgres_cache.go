package dictionary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresLessonCache struct{ db *pgxpool.Pool }

func NewPostgresLessonCache(db *pgxpool.Pool) *PostgresLessonCache {
	return &PostgresLessonCache{db: db}
}

func (c *PostgresLessonCache) Get(ctx context.Context, word string) (*LearningGuide, bool, error) {
	var data []byte
	err := c.db.QueryRow(ctx, `SELECT lesson FROM dictionary_lessons WHERE word=$1`, word).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read cached lesson: %w", err)
	}
	var lesson LearningGuide
	if err := json.Unmarshal(data, &lesson); err != nil {
		return nil, false, fmt.Errorf("decode cached lesson: %w", err)
	}
	if err := lesson.Validate(); err != nil {
		return nil, false, fmt.Errorf("validate cached lesson: %w", err)
	}
	return &lesson, true, nil
}

func (c *PostgresLessonCache) Set(ctx context.Context, word string, lesson *LearningGuide) error {
	data, err := json.Marshal(lesson)
	if err != nil {
		return fmt.Errorf("encode cached lesson: %w", err)
	}
	_, err = c.db.Exec(ctx, `
		INSERT INTO dictionary_lessons(word, lesson) VALUES($1, $2)
		ON CONFLICT(word) DO UPDATE SET lesson=EXCLUDED.lesson, updated_at=now()
	`, word, data)
	if err != nil {
		return fmt.Errorf("store cached lesson: %w", err)
	}
	return nil
}
