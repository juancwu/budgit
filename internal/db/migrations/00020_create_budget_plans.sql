-- +goose Up
-- +goose StatementBegin
CREATE TABLE budget_plans (
    id TEXT NOT NULL PRIMARY KEY,
    space_id TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    note TEXT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_budget_plans_space_id ON budget_plans (space_id);

CREATE TABLE budget_plan_lines (
    id TEXT NOT NULL PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES budget_plans(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    category_id TEXT NULL REFERENCES categories(id),
    label TEXT NOT NULL,
    amount TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_budget_plan_lines_plan_id ON budget_plan_lines (plan_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE budget_plan_lines;
DROP TABLE budget_plans;
-- +goose StatementEnd
