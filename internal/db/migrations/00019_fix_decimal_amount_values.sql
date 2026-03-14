-- +goose Up
-- Fix amount column values that were incorrectly converted from amount_cents
-- in migration 00018. The SUBSTR(..., -2, 2) approach works in SQLite but
-- fails in PostgreSQL where negative offsets mean "before string start".

-- expenses
UPDATE expenses SET amount =
    CASE WHEN amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(amount_cents) % 100 AS TEXT)
    END;

-- account_transfers
UPDATE account_transfers SET amount =
    CASE WHEN amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(amount_cents) % 100 AS TEXT)
    END;

-- recurring_expenses
UPDATE recurring_expenses SET amount =
    CASE WHEN amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(amount_cents) % 100 AS TEXT)
    END;

-- budgets
UPDATE budgets SET amount =
    CASE WHEN amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(amount_cents) % 100 AS TEXT)
    END;

-- recurring_deposits
UPDATE recurring_deposits SET amount =
    CASE WHEN amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(amount_cents) % 100 AS TEXT)
    END;

-- loans
UPDATE loans SET original_amount =
    CASE WHEN original_amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(original_amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(original_amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(original_amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(original_amount_cents) % 100 AS TEXT)
    END;

-- recurring_receipts
UPDATE recurring_receipts SET total_amount =
    CASE WHEN total_amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(total_amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(total_amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(total_amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(total_amount_cents) % 100 AS TEXT)
    END;

-- recurring_receipt_sources
UPDATE recurring_receipt_sources SET amount =
    CASE WHEN amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(amount_cents) % 100 AS TEXT)
    END;

-- receipts
UPDATE receipts SET total_amount =
    CASE WHEN total_amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(total_amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(total_amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(total_amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(total_amount_cents) % 100 AS TEXT)
    END;

-- receipt_funding_sources
UPDATE receipt_funding_sources SET amount =
    CASE WHEN amount_cents < 0 THEN '-' ELSE '' END ||
    CAST(ABS(amount_cents) / 100 AS TEXT) || '.' ||
    CASE WHEN ABS(amount_cents) % 100 < 10
         THEN '0' || CAST(ABS(amount_cents) % 100 AS TEXT)
         ELSE CAST(ABS(amount_cents) % 100 AS TEXT)
    END;

-- +goose Down
-- No-op: this migration fixes corrupted data from migration 00018.
