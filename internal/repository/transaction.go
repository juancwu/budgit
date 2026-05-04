package repository

import (
	"database/sql"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type TransactionRepository interface {
	CreateBillAtomic(t *model.Transaction, newBalance decimal.Decimal, categoryID *string) error
	CreateDepositAtomic(t *model.Transaction, newBalance decimal.Decimal) error
	UpdateBillAtomic(t *model.Transaction, newBalance decimal.Decimal, categoryID *string) error
	UpdateDepositAtomic(t *model.Transaction, newBalance decimal.Decimal) error
	TransferAtomic(withdrawal, deposit *model.Transaction, sourceNewBalance, destNewBalance decimal.Decimal) error
	GetByID(id string) (*model.Transaction, error)
	GetCategoryID(transactionID string) (*string, error)
	GetRelatedID(transactionID string) (*string, error)
	TransferIDsIn(ids []string) (map[string]bool, error)
	ListByAccount(accountID string, limit, offset int) ([]*model.Transaction, error)
	CountByAccount(accountID string) (int, error)
}

type transactionRepository struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) CreateBillAtomic(t *model.Transaction, newBalance decimal.Decimal, categoryID *string) error {
	return WithTx(r.db, func(tx *sqlx.Tx) error {
		insertTxn := `
			INSERT INTO transactions
				(id, value, type, account_id, title, description, occurred_at, created_at, updated_at)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9);
		`
		if _, err := tx.Exec(
			insertTxn,
			t.ID, t.Value, t.Type, t.AccountID, t.Title, t.Description,
			t.OccurredAt, t.CreatedAt, t.UpdatedAt,
		); err != nil {
			return err
		}

		updateBalance := `UPDATE accounts SET balance = $1, updated_at = $2 WHERE id = $3;`
		if _, err := tx.Exec(updateBalance, newBalance, time.Now(), t.AccountID); err != nil {
			return err
		}

		if categoryID != nil && *categoryID != "" {
			linkCategory := `INSERT INTO transaction_categories (category_id, transaction_id) VALUES ($1, $2);`
			if _, err := tx.Exec(linkCategory, *categoryID, t.ID); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *transactionRepository) CreateDepositAtomic(t *model.Transaction, newBalance decimal.Decimal) error {
	return WithTx(r.db, func(tx *sqlx.Tx) error {
		insertTxn := `
			INSERT INTO transactions
				(id, value, type, account_id, title, description, occurred_at, created_at, updated_at)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9);
		`
		if _, err := tx.Exec(
			insertTxn,
			t.ID, t.Value, t.Type, t.AccountID, t.Title, t.Description,
			t.OccurredAt, t.CreatedAt, t.UpdatedAt,
		); err != nil {
			return err
		}

		updateBalance := `UPDATE accounts SET balance = $1, updated_at = $2 WHERE id = $3;`
		if _, err := tx.Exec(updateBalance, newBalance, time.Now(), t.AccountID); err != nil {
			return err
		}
		return nil
	})
}

func (r *transactionRepository) UpdateBillAtomic(t *model.Transaction, newBalance decimal.Decimal, categoryID *string) error {
	return WithTx(r.db, func(tx *sqlx.Tx) error {
		updateTxn := `
			UPDATE transactions
			SET value = $1, title = $2, description = $3, occurred_at = $4, updated_at = $5
			WHERE id = $6;
		`
		if _, err := tx.Exec(
			updateTxn,
			t.Value, t.Title, t.Description, t.OccurredAt, t.UpdatedAt, t.ID,
		); err != nil {
			return err
		}

		updateBalance := `UPDATE accounts SET balance = $1, updated_at = $2 WHERE id = $3;`
		if _, err := tx.Exec(updateBalance, newBalance, time.Now(), t.AccountID); err != nil {
			return err
		}

		if _, err := tx.Exec(`DELETE FROM transaction_categories WHERE transaction_id = $1;`, t.ID); err != nil {
			return err
		}
		if categoryID != nil && *categoryID != "" {
			linkCategory := `INSERT INTO transaction_categories (category_id, transaction_id) VALUES ($1, $2);`
			if _, err := tx.Exec(linkCategory, *categoryID, t.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *transactionRepository) UpdateDepositAtomic(t *model.Transaction, newBalance decimal.Decimal) error {
	return WithTx(r.db, func(tx *sqlx.Tx) error {
		updateTxn := `
			UPDATE transactions
			SET value = $1, title = $2, description = $3, occurred_at = $4, updated_at = $5
			WHERE id = $6;
		`
		if _, err := tx.Exec(
			updateTxn,
			t.Value, t.Title, t.Description, t.OccurredAt, t.UpdatedAt, t.ID,
		); err != nil {
			return err
		}

		updateBalance := `UPDATE accounts SET balance = $1, updated_at = $2 WHERE id = $3;`
		if _, err := tx.Exec(updateBalance, newBalance, time.Now(), t.AccountID); err != nil {
			return err
		}
		return nil
	})
}

// TransferAtomic creates the withdrawal + deposit transaction pair, updates both
// account balances, and links the two via related_transactions in a single SQL
// transaction. Negative balances are allowed — overdraft enforcement is a product
// decision left to the service layer.
func (r *transactionRepository) TransferAtomic(withdrawal, deposit *model.Transaction, sourceNewBalance, destNewBalance decimal.Decimal) error {
	return WithTx(r.db, func(tx *sqlx.Tx) error {
		insertTxn := `
			INSERT INTO transactions
				(id, value, type, account_id, title, description, occurred_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
		`
		if _, err := tx.Exec(insertTxn,
			withdrawal.ID, withdrawal.Value, withdrawal.Type, withdrawal.AccountID, withdrawal.Title,
			withdrawal.Description, withdrawal.OccurredAt, withdrawal.CreatedAt, withdrawal.UpdatedAt,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(insertTxn,
			deposit.ID, deposit.Value, deposit.Type, deposit.AccountID, deposit.Title,
			deposit.Description, deposit.OccurredAt, deposit.CreatedAt, deposit.UpdatedAt,
		); err != nil {
			return err
		}

		updateBalance := `UPDATE accounts SET balance = $1, updated_at = $2 WHERE id = $3;`
		now := time.Now()
		if _, err := tx.Exec(updateBalance, sourceNewBalance, now, withdrawal.AccountID); err != nil {
			return err
		}
		if _, err := tx.Exec(updateBalance, destNewBalance, now, deposit.AccountID); err != nil {
			return err
		}

		// related_transactions has CHECK (transaction_one_id < transaction_two_id);
		// order the IDs to satisfy it.
		one, two := withdrawal.ID, deposit.ID
		if one > two {
			one, two = two, one
		}
		if _, err := tx.Exec(
			`INSERT INTO related_transactions (transaction_one_id, transaction_two_id) VALUES ($1, $2);`,
			one, two,
		); err != nil {
			return err
		}
		return nil
	})
}

func (r *transactionRepository) GetByID(id string) (*model.Transaction, error) {
	query := `
		SELECT id, value, type, account_id, title, description, occurred_at, created_at, updated_at
		FROM transactions
		WHERE id = $1;
	`
	t := &model.Transaction{}
	if err := r.db.Get(t, query, id); err != nil {
		return nil, err
	}
	return t, nil
}

// GetRelatedID returns the other half of a transfer pair if `transactionID` is
// part of one. Returns (nil, nil) when the transaction is standalone.
func (r *transactionRepository) GetRelatedID(transactionID string) (*string, error) {
	var other string
	err := r.db.Get(&other, `
		SELECT CASE
			WHEN transaction_one_id = $1 THEN transaction_two_id
			ELSE transaction_one_id
		END
		FROM related_transactions
		WHERE transaction_one_id = $1 OR transaction_two_id = $1
		LIMIT 1;
	`, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &other, nil
}

// TransferIDsIn returns the subset of `ids` that appear in related_transactions
// (either side). Used by list pages to decide which rows are non-editable so we
// don't N+1 a per-row check.
func (r *transactionRepository) TransferIDsIn(ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	query, args, err := sqlx.In(`
		SELECT transaction_one_id AS id FROM related_transactions WHERE transaction_one_id IN (?)
		UNION
		SELECT transaction_two_id AS id FROM related_transactions WHERE transaction_two_id IN (?)
	`, ids, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var hits []string
	if err := r.db.Select(&hits, query, args...); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(hits))
	for _, id := range hits {
		out[id] = true
	}
	return out, nil
}

func (r *transactionRepository) GetCategoryID(transactionID string) (*string, error) {
	var id string
	err := r.db.Get(&id, `SELECT category_id FROM transaction_categories WHERE transaction_id = $1 LIMIT 1;`, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

func (r *transactionRepository) ListByAccount(accountID string, limit, offset int) ([]*model.Transaction, error) {
	query := `
		SELECT id, value, type, account_id, title, description, occurred_at, created_at, updated_at
		FROM transactions
		WHERE account_id = $1
		ORDER BY occurred_at DESC, created_at DESC
		LIMIT $2 OFFSET $3;
	`
	txns := []*model.Transaction{}
	if err := r.db.Select(&txns, query, accountID, limit, offset); err != nil {
		return nil, err
	}
	return txns, nil
}

func (r *transactionRepository) CountByAccount(accountID string) (int, error) {
	var count int
	if err := r.db.Get(&count, `SELECT COUNT(*) FROM transactions WHERE account_id = $1;`, accountID); err != nil {
		return 0, err
	}
	return count, nil
}
