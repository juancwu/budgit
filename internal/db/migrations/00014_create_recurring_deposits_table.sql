-- +goose Up
CREATE TABLE recurring_deposits (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    frequency TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly', 'biweekly', 'monthly', 'yearly')),
    start_date DATE NOT NULL,
    end_date DATE,
    next_occurrence DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    title TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES money_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_recurring_deposits_space_id ON recurring_deposits(space_id);
CREATE INDEX idx_recurring_deposits_account_id ON recurring_deposits(account_id);
CREATE INDEX idx_recurring_deposits_next_occurrence ON recurring_deposits(next_occurrence);
CREATE INDEX idx_recurring_deposits_active ON recurring_deposits(is_active);

ALTER TABLE account_transfers ADD COLUMN recurring_deposit_id TEXT
    REFERENCES recurring_deposits(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE account_transfers DROP COLUMN IF EXISTS recurring_deposit_id;
DROP TABLE IF EXISTS recurring_deposits;
