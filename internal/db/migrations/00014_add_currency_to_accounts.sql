-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts ADD COLUMN currency TEXT NOT NULL DEFAULT 'CAD';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE accounts DROP COLUMN currency;
-- +goose StatementEnd
