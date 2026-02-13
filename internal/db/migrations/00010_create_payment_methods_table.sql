-- +goose Up
CREATE TABLE payment_methods (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('credit', 'debit')),
    last_four TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(space_id, name),
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);
ALTER TABLE expenses ADD COLUMN payment_method_id TEXT REFERENCES payment_methods(id) ON DELETE SET NULL;
CREATE INDEX idx_payment_method_space_id ON payment_methods(space_id);
CREATE INDEX idx_expenses_payment_method_id ON expenses(payment_method_id);

-- +goose Down
DROP INDEX IF EXISTS idx_expenses_payment_method_id;
DROP INDEX IF EXISTS idx_payment_method_space_id;
ALTER TABLE expenses DROP COLUMN IF EXISTS payment_method_id;
DROP TABLE IF EXISTS payment_methods;
