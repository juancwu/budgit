-- +goose Up
-- +goose StatementBegin
CREATE TABLE recurring_events (
    id TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('bill', 'fund')),
    source_account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    amount TEXT NOT NULL,
    description TEXT,

    frequency TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly', 'yearly')),
    interval_count INTEGER NOT NULL DEFAULT 1 CHECK (interval_count >= 1),
    day_of_week INTEGER CHECK (day_of_week IS NULL OR (day_of_week >= 0 AND day_of_week <= 6)),
    day_of_month INTEGER CHECK (day_of_month IS NULL OR (day_of_month >= 1 AND day_of_month <= 31)),
    month_of_year INTEGER CHECK (month_of_year IS NULL OR (month_of_year >= 1 AND month_of_year <= 12)),
    fire_hour INTEGER NOT NULL CHECK (fire_hour >= 0 AND fire_hour <= 23),
    fire_minute INTEGER NOT NULL CHECK (fire_minute >= 0 AND fire_minute <= 59),
    timezone TEXT NOT NULL,

    next_run_at TIMESTAMP NOT NULL,
    last_run_at TIMESTAMP,
    paused BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_recurring_events_space_id
    ON recurring_events (space_id, created_at DESC);

CREATE INDEX idx_recurring_events_due
    ON recurring_events (next_run_at)
    WHERE paused = FALSE;

CREATE INDEX idx_recurring_events_source_account
    ON recurring_events (source_account_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE recurring_events;
-- +goose StatementEnd
