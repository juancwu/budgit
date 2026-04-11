-- +goose Up
-- +goose StatementBegin
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

-- +goose Down
-- +goose StatementBegin
DELETE FROM categories WHERE id IN (
    'housing', 'food', 'transport', 'health',
    'lifestyle', 'shopping', 'personal', 'savings_debt'
);
-- +goose StatementEnd
