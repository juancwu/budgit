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
	GetByID(id string) (*model.Transaction, error)
	GetCategoryID(transactionID string) (*string, error)
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
