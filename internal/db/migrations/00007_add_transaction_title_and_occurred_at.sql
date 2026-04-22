-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE transactions ADD COLUMN occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions DROP COLUMN occurred_at;
ALTER TABLE transactions DROP COLUMN title;
-- +goose StatementEnd
