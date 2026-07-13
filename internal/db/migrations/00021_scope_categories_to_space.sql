-- +goose Up
-- +goose StatementBegin
-- Categories become user-created and owned by a space. Drop the predefined
-- (global, seedless-owner) categories and any references to them first, then
-- add the space_id owner column.
UPDATE budget_plan_lines SET category_id = NULL;
DELETE FROM transaction_categories;
DELETE FROM categories;

ALTER TABLE categories
    ADD COLUMN space_id TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE;

CREATE INDEX idx_categories_space_id ON categories (space_id);
-- Category names are unique within a space (case-sensitive at the DB level;
-- the service layer normalizes/validates before insert).
CREATE UNIQUE INDEX idx_categories_space_name ON categories (space_id, name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_categories_space_name;
DROP INDEX IF EXISTS idx_categories_space_id;
ALTER TABLE categories DROP COLUMN space_id;

-- Restore the predefined categories that existed before this migration.
INSERT INTO categories (id, name, description) VALUES
    ('housing',      'Housing',        'rent/mortgage, utilities, maintenance'),
    ('food',         'Food',           'groceries and dining out'),
    ('transport',    'Transport',      'fuel, transit, car payments, parking'),
    ('health',       'Health',         'medical, pharmacy, gym'),
    ('lifestyle',    'Lifestyle',      'entertainment, hobbies, subscriptions'),
    ('shopping',     'Shopping',       'clothing, electronics, household goods'),
    ('personal',     'Personal',       'haircuts, gifts, donations'),
    ('savings_debt', 'Savings & Debt', 'loan payments, savings contributions');
-- +goose StatementEnd
