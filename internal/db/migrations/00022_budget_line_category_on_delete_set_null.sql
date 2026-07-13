-- +goose Up
-- +goose StatementBegin
-- Deleting a user-created category should leave any budget plan lines that
-- referenced it "Uncategorized" rather than blocking the delete. (Transaction
-- category links already cascade.)
ALTER TABLE budget_plan_lines DROP CONSTRAINT budget_plan_lines_category_id_fkey;
ALTER TABLE budget_plan_lines
    ADD CONSTRAINT budget_plan_lines_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE budget_plan_lines DROP CONSTRAINT budget_plan_lines_category_id_fkey;
ALTER TABLE budget_plan_lines
    ADD CONSTRAINT budget_plan_lines_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories(id);
-- +goose StatementEnd
