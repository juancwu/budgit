-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tokens (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tokens_token ON tokens(token);
CREATE INDEX IF NOT EXISTS idx_tokens_expires_at ON tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_tokens_user_id ON tokens(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tokens_user_id;
DROP INDEX IF EXISTS idx_tokens_expires_at;
DROP INDEX IF EXISTS idx_tokens_token;
DROP TABLE IF EXISTS tokens;
-- +goose StatementEnd
