-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS expenses (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    created_by TEXT NOT NULL,
    description TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('expense', 'topup')),
    date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS expense_tags (
    expense_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (expense_id, tag_id),
    FOREIGN KEY (expense_id) REFERENCES expenses(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS expense_items (
    expense_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    PRIMARY KEY (expense_id, item_id),
    FOREIGN KEY (expense_id) REFERENCES expenses(id) ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES list_items(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_expenses_space_id ON expenses(space_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_expenses_space_id;
DROP TABLE IF EXISTS expense_items;
DROP TABLE IF EXISTS expense_tags;
DROP TABLE IF EXISTS expenses;
-- +goose StatementEnd
