-- +goose Up
ALTER TABLE profiles ADD COLUMN timezone TEXT;

-- +goose Down
ALTER TABLE profiles DROP COLUMN IF EXISTS timezone;
