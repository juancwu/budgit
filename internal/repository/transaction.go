package repository

import (
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type TransactionRepository interface {
	CreateBillAtomic(t *model.Transaction, newBalance decimal.Decimal, categoryID *string) error
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
