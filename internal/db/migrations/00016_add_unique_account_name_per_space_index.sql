-- +goose Up
CREATE UNIQUE INDEX unique_account_name_per_space_index ON accounts(name, space_id);

-- +goose Down
DROP INDEX unique_account_name_per_space_index;
