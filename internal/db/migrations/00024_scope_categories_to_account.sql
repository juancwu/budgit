-- +goose Up
-- +goose StatementBegin
-- Categories move from space scope to account scope. There is no clean mapping
-- from a space category to a single account, so drop existing categories and
-- their transaction links, then re-key the table to account_id.
DELETE FROM transaction_categories;
DELETE FROM categories;

DROP INDEX IF EXISTS idx_categories_space_name;
DROP INDEX IF EXISTS idx_categories_space_id;
ALTER TABLE categories DROP COLUMN space_id;

ALTER TABLE categories
    ADD COLUMN account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE;

CREATE INDEX idx_categories_account_id ON categories (account_id);
CREATE UNIQUE INDEX idx_categories_account_name ON categories (account_id, name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_categories_account_name;
DROP INDEX IF EXISTS idx_categories_account_id;
ALTER TABLE categories DROP COLUMN account_id;

ALTER TABLE categories
    ADD COLUMN space_id TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE;

CREATE INDEX idx_categories_space_id ON categories (space_id);
CREATE UNIQUE INDEX idx_categories_space_name ON categories (space_id, name);
-- +goose StatementEnd
