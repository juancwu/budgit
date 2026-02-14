-- +goose Up
CREATE TABLE recurring_expenses (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    created_by TEXT NOT NULL,
    description TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('expense', 'topup')),
    payment_method_id TEXT,
    frequency TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly', 'biweekly', 'monthly', 'yearly')),
    start_date DATE NOT NULL,
    end_date DATE,
    next_occurrence DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE SET NULL
);

CREATE TABLE recurring_expense_tags (
    recurring_expense_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (recurring_expense_id, tag_id),
    FOREIGN KEY (recurring_expense_id) REFERENCES recurring_expenses(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

ALTER TABLE expenses ADD COLUMN recurring_expense_id TEXT REFERENCES recurring_expenses(id) ON DELETE SET NULL;

CREATE INDEX idx_recurring_expenses_space_id ON recurring_expenses(space_id);
CREATE INDEX idx_recurring_expenses_next_occurrence ON recurring_expenses(next_occurrence);
CREATE INDEX idx_recurring_expenses_active ON recurring_expenses(is_active);
CREATE INDEX idx_expenses_recurring_expense_id ON expenses(recurring_expense_id);

-- +goose Down
DROP INDEX IF EXISTS idx_expenses_recurring_expense_id;
DROP INDEX IF EXISTS idx_recurring_expenses_active;
DROP INDEX IF EXISTS idx_recurring_expenses_next_occurrence;
DROP INDEX IF EXISTS idx_recurring_expenses_space_id;
ALTER TABLE expenses DROP COLUMN IF EXISTS recurring_expense_id;
DROP TABLE IF EXISTS recurring_expense_tags;
DROP TABLE IF EXISTS recurring_expenses;
