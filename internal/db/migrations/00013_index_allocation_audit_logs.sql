-- +goose Up
-- +goose StatementBegin
-- Mirror of idx_space_audit_logs_account_id but for allocation.* actions, so
-- the account-scoped activity feed can OR-merge the two prefixes without a
-- sequential scan over space_audit_logs.

CREATE INDEX idx_space_audit_logs_allocation_account_id
    ON space_audit_logs ((metadata->>'account_id'), created_at DESC)
    WHERE action LIKE 'allocation.%';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_space_audit_logs_allocation_account_id;
-- +goose StatementEnd
