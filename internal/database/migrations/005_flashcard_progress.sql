CREATE TABLE user_word_progress (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    word_id bigint NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    due timestamptz NOT NULL,
    stability double precision NOT NULL DEFAULT 0,
    difficulty double precision NOT NULL DEFAULT 0,
    scheduled_days bigint NOT NULL DEFAULT 0,
    reps bigint NOT NULL DEFAULT 0,
    lapses bigint NOT NULL DEFAULT 0,
    state smallint NOT NULL DEFAULT 0 CHECK (state BETWEEN 0 AND 3),
    last_review timestamptz,
    remaining_steps integer NOT NULL DEFAULT 0,
    known_count integer NOT NULL DEFAULT 0,
    unknown_count integer NOT NULL DEFAULT 0,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, word_id)
);

CREATE INDEX user_word_due_idx ON user_word_progress (user_id, due, unknown_count DESC);

CREATE TABLE flashcard_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level text NOT NULL CHECK (level IN ('A0', 'A1', 'A2', 'B1', 'B2')),
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);

CREATE TABLE flashcard_session_cards (
    session_id uuid NOT NULL REFERENCES flashcard_sessions(id) ON DELETE CASCADE,
    word_id bigint NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    position integer NOT NULL,
    was_review boolean NOT NULL,
    answered_at timestamptz,
    knew_answer boolean,
    PRIMARY KEY (session_id, word_id),
    UNIQUE (session_id, position)
);

CREATE TABLE flashcard_review_logs (
    id bigserial PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    word_id bigint NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    session_id uuid NOT NULL REFERENCES flashcard_sessions(id) ON DELETE CASCADE,
    rating smallint NOT NULL CHECK (rating IN (1, 3)),
    reviewed_at timestamptz NOT NULL,
    previous_state smallint NOT NULL,
    previous_due timestamptz NOT NULL,
    scheduled_days bigint NOT NULL,
    stability double precision NOT NULL,
    difficulty double precision NOT NULL
);

CREATE INDEX flashcard_review_logs_user_word_idx ON flashcard_review_logs (user_id, word_id, reviewed_at DESC);
