ALTER TABLE flashcard_sessions
    ADD COLUMN mode text NOT NULL DEFAULT 'learn'
    CHECK (mode IN ('learn', 'review'));
