-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NULL, -- Allow null for passwordless login
    pending_email TEXT NULL, -- Store new email when changing email
    email_verified_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_passwordless ON users(id) WHERE password_hash IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_passwordless;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
