CREATE TABLE words (
    id bigserial PRIMARY KEY,
    word text NOT NULL,
    level text NOT NULL CHECK (level IN ('A0', 'A1', 'A2', 'B1', 'B2')),
    part_of_speech text NOT NULL DEFAULT '',
    frequency_per_million double precision NOT NULL DEFAULT 0,
    frequency_rank integer NOT NULL,
    source text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT words_source_entry_unique UNIQUE (source, word, part_of_speech)
);

CREATE INDEX words_level_random_idx ON words (level, id);
CREATE INDEX words_word_idx ON words (word);
