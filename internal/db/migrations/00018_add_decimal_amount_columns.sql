-- +goose Up

-- expenses
ALTER TABLE expenses ADD COLUMN amount TEXT NOT NULL DEFAULT '0';
UPDATE expenses SET amount = CAST(amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(amount_cents) % 100 AS TEXT), -2, 2);

-- account_transfers
ALTER TABLE account_transfers ADD COLUMN amount TEXT NOT NULL DEFAULT '0';
UPDATE account_transfers SET amount = CAST(amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(amount_cents) % 100 AS TEXT), -2, 2);

-- recurring_expenses
ALTER TABLE recurring_expenses ADD COLUMN amount TEXT NOT NULL DEFAULT '0';
UPDATE recurring_expenses SET amount = CAST(amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(amount_cents) % 100 AS TEXT), -2, 2);

-- budgets
ALTER TABLE budgets ADD COLUMN amount TEXT NOT NULL DEFAULT '0';
UPDATE budgets SET amount = CAST(amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(amount_cents) % 100 AS TEXT), -2, 2);

-- recurring_deposits
ALTER TABLE recurring_deposits ADD COLUMN amount TEXT NOT NULL DEFAULT '0';
UPDATE recurring_deposits SET amount = CAST(amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(amount_cents) % 100 AS TEXT), -2, 2);

-- loans
ALTER TABLE loans ADD COLUMN original_amount TEXT NOT NULL DEFAULT '0';
UPDATE loans SET original_amount = CAST(original_amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(original_amount_cents) % 100 AS TEXT), -2, 2);

-- recurring_receipts
ALTER TABLE recurring_receipts ADD COLUMN total_amount TEXT NOT NULL DEFAULT '0';
UPDATE recurring_receipts SET total_amount = CAST(total_amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(total_amount_cents) % 100 AS TEXT), -2, 2);

-- recurring_receipt_sources
ALTER TABLE recurring_receipt_sources ADD COLUMN amount TEXT NOT NULL DEFAULT '0';
UPDATE recurring_receipt_sources SET amount = CAST(amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(amount_cents) % 100 AS TEXT), -2, 2);

-- receipts
ALTER TABLE receipts ADD COLUMN total_amount TEXT NOT NULL DEFAULT '0';
UPDATE receipts SET total_amount = CAST(total_amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(total_amount_cents) % 100 AS TEXT), -2, 2);

-- receipt_funding_sources
ALTER TABLE receipt_funding_sources ADD COLUMN amount TEXT NOT NULL DEFAULT '0';
UPDATE receipt_funding_sources SET amount = CAST(amount_cents / 100 AS TEXT) || '.' || SUBSTR('00' || CAST(ABS(amount_cents) % 100 AS TEXT), -2, 2);

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions, but modernc.org/sqlite supports it.
ALTER TABLE expenses DROP COLUMN amount;
ALTER TABLE account_transfers DROP COLUMN amount;
ALTER TABLE recurring_expenses DROP COLUMN amount;
ALTER TABLE budgets DROP COLUMN amount;
ALTER TABLE recurring_deposits DROP COLUMN amount;
ALTER TABLE loans DROP COLUMN original_amount;
ALTER TABLE recurring_receipts DROP COLUMN total_amount;
ALTER TABLE recurring_receipt_sources DROP COLUMN amount;
ALTER TABLE receipts DROP COLUMN total_amount;
ALTER TABLE receipt_funding_sources DROP COLUMN amount;
