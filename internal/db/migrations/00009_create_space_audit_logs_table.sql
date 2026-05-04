-- +goose Up
-- +goose StatementBegin
CREATE TABLE space_audit_logs (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    actor_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    target_email TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_space_audit_logs_space_id_created_at
    ON space_audit_logs (space_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE space_audit_logs;
-- +goose StatementEnd
