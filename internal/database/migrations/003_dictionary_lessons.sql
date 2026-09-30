CREATE TABLE dictionary_lessons (
    word text PRIMARY KEY,
    lesson jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
