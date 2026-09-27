ALTER TABLE users ADD COLUMN username text;

CREATE UNIQUE INDEX users_username_unique
    ON users (lower(username))
    WHERE username IS NOT NULL;
