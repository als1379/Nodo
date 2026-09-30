ALTER TABLE words RENAME COLUMN frequency_rank TO curriculum_rank;
ALTER TABLE words ADD COLUMN active boolean NOT NULL DEFAULT true;

CREATE INDEX words_active_curriculum_idx
    ON words(level, part_of_speech, curriculum_rank, word)
    WHERE active;
