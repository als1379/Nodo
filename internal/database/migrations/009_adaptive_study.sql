CREATE TABLE study_settings (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    difficulty_target double precision NOT NULL DEFAULT 15 CHECK (difficulty_target BETWEEN 0 AND 100),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Retain historical CEFR metadata; new sessions span the whole scored vocabulary.
ALTER TABLE flashcard_sessions ALTER COLUMN level DROP NOT NULL;
ALTER TABLE flashcard_sessions ADD COLUMN difficulty_target double precision;
CREATE INDEX flashcard_sessions_resume_idx ON flashcard_sessions(user_id,mode,created_at DESC)
    WHERE completed_at IS NULL;
