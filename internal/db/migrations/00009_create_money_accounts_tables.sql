-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS money_accounts (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(space_id, name),
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS account_transfers (
    id TEXT PRIMARY KEY NOT NULL,
    account_id TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('deposit', 'withdrawal')),
    note TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES money_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_money_accounts_space_id ON money_accounts(space_id);
CREATE INDEX IF NOT EXISTS idx_account_transfers_account_id ON account_transfers(account_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_account_transfers_account_id;
DROP INDEX IF EXISTS idx_money_accounts_space_id;
DROP TABLE IF EXISTS account_transfers;
DROP TABLE IF EXISTS money_accounts;
-- +goose StatementEnd
