package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// AllocationConversion describes the converted amount/target for a single
// allocation row when an account's currency is changed.
type AllocationConversion struct {
	ID           string
	Amount       decimal.Decimal
	TargetAmount *decimal.Decimal
}

var ErrAccountNotFound = errors.New("account not found")

type AccountRepository interface {
	Create(account *model.Account) error
	ByID(id string) (*model.Account, error)
	BySpaceID(spaceID string) ([]*model.Account, error)
	Rename(id, name string) error
	Delete(id string) error
	// ChangeCurrency atomically updates an account's currency and balance and
	// rewrites each provided allocation's amount/target in the new currency.
	ChangeCurrency(accountID, newCurrency string, newBalance decimal.Decimal, allocationConversions []AllocationConversion) error
}

type accountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(account *model.Account) error {
	query := `INSERT INTO accounts (id, name, space_id, currency, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6);`
	_, err := r.db.Exec(query, account.ID, account.Name, account.SpaceID, account.Currency, account.CreatedAt, account.UpdatedAt)
	return err
}

func (r *accountRepository) ByID(id string) (*model.Account, error) {
	account := &model.Account{}
	query := `SELECT * FROM accounts WHERE id = $1;`
	err := r.db.Get(account, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrAccountNotFound
	}
	return account, err
}

func (r *accountRepository) BySpaceID(spaceID string) ([]*model.Account, error) {
	var accounts []*model.Account
	query := `SELECT * FROM accounts WHERE space_id = $1 ORDER BY created_at ASC;`
	err := r.db.Select(&accounts, query, spaceID)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *accountRepository) Rename(id, name string) error {
	query := `UPDATE accounts SET name = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;`
	_, err := r.db.Exec(query, name, id)
	return err
}

func (r *accountRepository) Delete(id string) error {
	query := `DELETE FROM accounts WHERE id = $1;`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *accountRepository) ChangeCurrency(accountID, newCurrency string, newBalance decimal.Decimal, allocationConversions []AllocationConversion) error {
	return WithTx(r.db, func(tx *sqlx.Tx) error {
		now := time.Now()
		if _, err := tx.Exec(
			`UPDATE accounts SET currency = $1, balance = $2, updated_at = $3 WHERE id = $4;`,
			newCurrency, newBalance, now, accountID,
		); err != nil {
			return err
		}
		for _, c := range allocationConversions {
			if _, err := tx.Exec(
				`UPDATE allocations SET amount = $1, target_amount = $2, updated_at = $3 WHERE id = $4;`,
				c.Amount, c.TargetAmount, now, c.ID,
			); err != nil {
				return err
			}
		}
		return nil
	})
}
