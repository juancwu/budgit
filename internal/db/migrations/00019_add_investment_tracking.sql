-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts
    ADD COLUMN is_investment BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN investment_subtype TEXT NULL;

CREATE TABLE investment_contribution_rooms (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    year INTEGER NOT NULL,
    room_amount TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, year)
);

CREATE TABLE investment_holdings (
    id TEXT NOT NULL PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    display_name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (account_id, symbol)
);

CREATE TABLE investment_trades (
    id TEXT NOT NULL PRIMARY KEY,
    holding_id TEXT NOT NULL REFERENCES investment_holdings(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    quantity TEXT NOT NULL,
    price_per_unit TEXT NOT NULL,
    fees TEXT NULL,
    occurred_at TIMESTAMP NOT NULL,
    notes TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_investment_trades_holding_id_occurred_at
    ON investment_trades (holding_id, occurred_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE investment_trades;
DROP TABLE investment_holdings;
DROP TABLE investment_contribution_rooms;
ALTER TABLE accounts
    DROP COLUMN investment_subtype,
    DROP COLUMN is_investment;
-- +goose StatementEnd
