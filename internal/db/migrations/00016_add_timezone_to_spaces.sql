-- +goose Up
ALTER TABLE spaces ADD COLUMN timezone TEXT;

-- +goose Down
ALTER TABLE spaces DROP COLUMN IF EXISTS timezone;
