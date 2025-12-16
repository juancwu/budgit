-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    owner_type TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    type TEXT NOT NULL,
    filename TEXT NOT NULL,
    original_name TEXT,
    mime_type TEXT,
    size INTEGER,
    storage_path TEXT NOT NULL,
    public BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files(owner_type, owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_owner_type ON files(owner_type, owner_id, type) WHERE type IN ('avatar');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_files_owner_type;
DROP INDEX IF EXISTS idx_files_owner;
DROP INDEX IF EXISTS idx_files_user_id;
DROP TABLE IF EXISTS files;
-- +goose StatementEnd
