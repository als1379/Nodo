ALTER TABLE words
    ADD COLUMN difficulty_score numeric(5,2),
    ADD COLUMN difficulty_source text,
    ADD COLUMN difficulty_confidence numeric(4,3),
    ADD COLUMN age_of_acquisition numeric(4,2),
    ADD COLUMN frequency_zipf numeric(4,2),
    ADD COLUMN difficulty_model_version text,
    ADD COLUMN difficulty_scored_at timestamptz;

ALTER TABLE words
    ADD CONSTRAINT words_difficulty_score_range
        CHECK (difficulty_score IS NULL OR difficulty_score BETWEEN 0 AND 100),
    ADD CONSTRAINT words_difficulty_confidence_range
        CHECK (difficulty_confidence IS NULL OR difficulty_confidence BETWEEN 0 AND 1);

CREATE INDEX words_active_difficulty_idx
    ON words(difficulty_score, curriculum_rank)
    WHERE active;
