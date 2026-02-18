-- +goose Up
-- Drop index before rename (SQLite keeps index names after table rename)
DROP INDEX IF EXISTS idx_budgets_space_id;

-- Rename old budgets table so we can recreate it without tag_id
ALTER TABLE budgets RENAME TO budgets_old;

-- Recreate budgets table without tag_id (SQLite can't DROP COLUMN with constraints)
CREATE TABLE budgets (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    period TEXT NOT NULL CHECK (period IN ('weekly', 'monthly', 'yearly')),
    start_date DATE NOT NULL,
    end_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO budgets (id, space_id, amount_cents, period, start_date, end_date, is_active, created_by, created_at, updated_at)
SELECT id, space_id, amount_cents, period, start_date, end_date, is_active, created_by, created_at, updated_at FROM budgets_old;

CREATE INDEX idx_budgets_space_id ON budgets(space_id);

-- Create budget_tags join table for many-to-many relationship
CREATE TABLE budget_tags (
    budget_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (budget_id, tag_id),
    FOREIGN KEY (budget_id) REFERENCES budgets(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX idx_budget_tags_tag_id ON budget_tags(tag_id);

-- Migrate existing tag_id data to the join table
INSERT INTO budget_tags (budget_id, tag_id)
SELECT id, tag_id FROM budgets_old WHERE tag_id IS NOT NULL;

-- Drop the old table (nothing references it now)
DROP TABLE budgets_old;

-- +goose Down
-- Drop budget_tags first (it references budgets)
DROP INDEX IF EXISTS idx_budget_tags_tag_id;

-- Save tag mappings before dropping budget_tags
CREATE TEMP TABLE budget_tag_mappings AS
SELECT budget_id, tag_id FROM budget_tags;

DROP TABLE budget_tags;

-- Drop index before rename (SQLite keeps index names after table rename)
DROP INDEX IF EXISTS idx_budgets_space_id;

-- Rename current budgets out of the way
ALTER TABLE budgets RENAME TO budgets_new;

-- Recreate budgets table with tag_id column
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

-- Copy data back, restoring first tag from saved mappings
INSERT INTO budgets (id, space_id, tag_id, amount_cents, period, start_date, end_date, is_active, created_by, created_at, updated_at)
SELECT b.id, b.space_id,
    COALESCE((SELECT m.tag_id FROM budget_tag_mappings m WHERE m.budget_id = b.id LIMIT 1), ''),
    b.amount_cents, b.period, b.start_date, b.end_date, b.is_active, b.created_by, b.created_at, b.updated_at
FROM budgets_new b;

CREATE INDEX idx_budgets_space_id ON budgets(space_id);

DROP TABLE budgets_new;
DROP TABLE budget_tag_mappings;
