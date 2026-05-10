-- +goose Up
-- +goose StatementBegin
ALTER TABLE recurring_events
    ADD COLUMN business_days_only BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recurring_events
    DROP COLUMN business_days_only;
-- +goose StatementEnd
