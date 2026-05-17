-- +goose Up
-- +goose StatementBegin
-- Flag on users for fast middleware lookups: once set, every request from the
-- user is funneled to the "Account Pending Deletion" page until the background
-- worker finishes wiping their data.
ALTER TABLE users ADD COLUMN pending_deletion_at TIMESTAMP NULL;
CREATE INDEX idx_users_pending_deletion_at ON users (pending_deletion_at) WHERE pending_deletion_at IS NOT NULL;

-- Single table that acts as both the work queue AND the permanent audit
-- record for account deletion requests. Rows are not foreign-keyed to users
-- because the related user row is hard-deleted on completion; the snapshot
-- columns preserve who/when/from-where for audit purposes after the data is
-- gone. Operational columns (status, attempts, last_error) let a background
-- worker pick the row up, retry on failure, and resume across restarts.
CREATE TABLE account_deletion_requests (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    name TEXT NULL,
    reason TEXT NULL,
    ip_address TEXT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending | processing | completed | failed
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    spaces_deleted INTEGER NULL,
    requested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL
);

CREATE INDEX idx_account_deletion_requests_pending
    ON account_deletion_requests (requested_at)
    WHERE status IN ('pending', 'processing');
CREATE INDEX idx_account_deletion_requests_user_id
    ON account_deletion_requests (user_id);
CREATE INDEX idx_account_deletion_requests_requested_at
    ON account_deletion_requests (requested_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE account_deletion_requests;
DROP INDEX IF EXISTS idx_users_pending_deletion_at;
ALTER TABLE users DROP COLUMN pending_deletion_at;
-- +goose StatementEnd
