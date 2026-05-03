-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_audit_logs (
    id TEXT PRIMARY KEY NOT NULL,
    transaction_id TEXT NOT NULL,
    actor_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transaction_audit_logs_transaction_id_created_at
    ON transaction_audit_logs (transaction_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE transaction_audit_logs;
-- +goose StatementEnd
