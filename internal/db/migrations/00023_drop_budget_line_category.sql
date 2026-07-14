-- +goose Up
-- +goose StatementBegin
-- Budget plans are decoupled from categories: plan lines are now label-only.
ALTER TABLE budget_plan_lines DROP COLUMN category_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE budget_plan_lines
    ADD COLUMN category_id TEXT NULL REFERENCES categories(id) ON DELETE SET NULL;
-- +goose StatementEnd
