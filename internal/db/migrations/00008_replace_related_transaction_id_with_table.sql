-- +goose Up
-- +goose StatementBegin
CREATE TABLE related_transactions (
    transaction_one_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    transaction_two_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (transaction_one_id, transaction_two_id),
    UNIQUE (transaction_one_id),
    UNIQUE (transaction_two_id),
    CHECK (transaction_one_id < transaction_two_id)
);

ALTER TABLE transactions DROP COLUMN related_transaction_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN related_transaction_id TEXT REFERENCES transactions(id) ON DELETE SET NULL;

DROP TABLE related_transactions;
-- +goose StatementEnd
