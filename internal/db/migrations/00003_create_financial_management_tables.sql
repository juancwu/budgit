-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    space_id TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions (
    id TEXT NOT NULL PRIMARY KEY,
    value TEXT NOT NULL,
    type TEXT NOT NULL,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    description TEXT,
    related_transaction_id TEXT REFERENCES transactions(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tags (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    space_id TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transaction_tags (
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tag_id, transaction_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE transaction_tags;
DROP TABLE tags;
DROP TABLE transactions;
DROP TABLE accounts;
-- +goose StatementEnd
