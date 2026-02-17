-- +goose Up
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
SELECT id, tag_id FROM budgets WHERE tag_id IS NOT NULL;

-- Drop the unique constraint and FK that reference tag_id, then drop the column
ALTER TABLE budgets DROP CONSTRAINT budgets_space_id_tag_id_period_key;
ALTER TABLE budgets DROP CONSTRAINT budgets_tag_id_fkey;
ALTER TABLE budgets DROP COLUMN tag_id;

-- +goose Down
-- Add tag_id column back
ALTER TABLE budgets ADD COLUMN tag_id TEXT;

-- Copy first tag back from budget_tags
UPDATE budgets SET tag_id = (
    SELECT tag_id FROM budget_tags WHERE budget_tags.budget_id = budgets.id LIMIT 1
);

-- Re-add FK and unique constraint
ALTER TABLE budgets ADD CONSTRAINT budgets_tag_id_fkey
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;
ALTER TABLE budgets ADD CONSTRAINT budgets_space_id_tag_id_period_key
    UNIQUE (space_id, tag_id, period);

DROP INDEX IF EXISTS idx_budget_tags_tag_id;
DROP TABLE IF EXISTS budget_tags;
