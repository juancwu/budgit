-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts ADD COLUMN balance TEXT NOT NULL DEFAULT '0.00';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE accounts DROP COLUMN balance;
-- +goose StatementEnd
