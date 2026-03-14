-- +goose Up

CREATE TABLE loans (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    original_amount_cents INTEGER NOT NULL,
    interest_rate_bps INTEGER NOT NULL DEFAULT 0,
    start_date DATE NOT NULL,
    end_date DATE,
    is_paid_off BOOLEAN NOT NULL DEFAULT FALSE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_loans_space_id ON loans(space_id);

CREATE TABLE recurring_receipts (
    id TEXT PRIMARY KEY NOT NULL,
    loan_id TEXT NOT NULL,
    space_id TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    total_amount_cents INTEGER NOT NULL,
    frequency TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly', 'biweekly', 'monthly', 'yearly')),
    start_date DATE NOT NULL,
    end_date DATE,
    next_occurrence DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (loan_id) REFERENCES loans(id) ON DELETE CASCADE,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_recurring_receipts_space_id ON recurring_receipts(space_id);
CREATE INDEX idx_recurring_receipts_loan_id ON recurring_receipts(loan_id);
CREATE INDEX idx_recurring_receipts_next_occurrence ON recurring_receipts(next_occurrence);
CREATE INDEX idx_recurring_receipts_active ON recurring_receipts(is_active);

CREATE TABLE recurring_receipt_sources (
    id TEXT PRIMARY KEY NOT NULL,
    recurring_receipt_id TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('balance', 'account')),
    account_id TEXT,
    amount_cents INTEGER NOT NULL,
    FOREIGN KEY (recurring_receipt_id) REFERENCES recurring_receipts(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES money_accounts(id) ON DELETE SET NULL
);

CREATE INDEX idx_recurring_receipt_sources_recurring_receipt_id ON recurring_receipt_sources(recurring_receipt_id);

CREATE TABLE receipts (
    id TEXT PRIMARY KEY NOT NULL,
    loan_id TEXT NOT NULL,
    space_id TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    total_amount_cents INTEGER NOT NULL,
    date DATE NOT NULL,
    recurring_receipt_id TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (loan_id) REFERENCES loans(id) ON DELETE CASCADE,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (recurring_receipt_id) REFERENCES recurring_receipts(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_receipts_loan_id ON receipts(loan_id);
CREATE INDEX idx_receipts_space_id ON receipts(space_id);
CREATE INDEX idx_receipts_recurring_receipt_id ON receipts(recurring_receipt_id);

CREATE TABLE receipt_funding_sources (
    id TEXT PRIMARY KEY NOT NULL,
    receipt_id TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('balance', 'account')),
    account_id TEXT,
    amount_cents INTEGER NOT NULL,
    linked_expense_id TEXT,
    linked_transfer_id TEXT,
    FOREIGN KEY (receipt_id) REFERENCES receipts(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES money_accounts(id) ON DELETE SET NULL,
    FOREIGN KEY (linked_expense_id) REFERENCES expenses(id) ON DELETE SET NULL,
    FOREIGN KEY (linked_transfer_id) REFERENCES account_transfers(id) ON DELETE SET NULL
);

CREATE INDEX idx_receipt_funding_sources_receipt_id ON receipt_funding_sources(receipt_id);

-- +goose Down
DROP INDEX IF EXISTS idx_receipt_funding_sources_receipt_id;
DROP INDEX IF EXISTS idx_receipts_recurring_receipt_id;
DROP INDEX IF EXISTS idx_receipts_space_id;
DROP INDEX IF EXISTS idx_receipts_loan_id;
DROP INDEX IF EXISTS idx_recurring_receipt_sources_recurring_receipt_id;
DROP INDEX IF EXISTS idx_recurring_receipts_active;
DROP INDEX IF EXISTS idx_recurring_receipts_next_occurrence;
DROP INDEX IF EXISTS idx_recurring_receipts_loan_id;
DROP INDEX IF EXISTS idx_recurring_receipts_space_id;
DROP INDEX IF EXISTS idx_loans_space_id;
DROP TABLE IF EXISTS receipt_funding_sources;
DROP TABLE IF EXISTS receipts;
DROP TABLE IF EXISTS recurring_receipt_sources;
DROP TABLE IF EXISTS recurring_receipts;
DROP TABLE IF EXISTS loans;
