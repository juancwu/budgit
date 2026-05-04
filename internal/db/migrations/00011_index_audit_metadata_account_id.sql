-- +goose Up
-- +goose StatementBegin
-- The account-scoped activity feeds filter audit rows by metadata->>'account_id'.
-- A partial expression index is the right shape for this access pattern in
-- PostgreSQL 17: it is small (only the rows where the field exists), uses a
-- standard B-tree (cheap equality + ORDER BY created_at), and avoids the bloat
-- of a full GIN over the metadata document.

CREATE INDEX idx_space_audit_logs_account_id
    ON space_audit_logs ((metadata->>'account_id'), created_at DESC)
    WHERE action LIKE 'account.%';

CREATE INDEX idx_transaction_audit_logs_account_id
    ON transaction_audit_logs ((metadata->>'account_id'), created_at DESC)
    WHERE metadata ? 'account_id';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_space_audit_logs_account_id;
DROP INDEX IF EXISTS idx_transaction_audit_logs_account_id;
-- +goose StatementEnd
