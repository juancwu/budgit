-- +goose Up
CREATE TABLE budgets (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    period TEXT NOT NULL CHECK (period IN ('weekly', 'monthly', 'yearly')),
    start_date DATE NOT NULL,
    end_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(space_id, tag_id, period),
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_budgets_space_id ON budgets(space_id);

-- +goose Down
DROP INDEX IF EXISTS idx_budgets_space_id;
DROP TABLE IF EXISTS budgets;
