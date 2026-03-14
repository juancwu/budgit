package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var (
	ErrMoneyAccountNotFound = errors.New("money account not found")
	ErrTransferNotFound     = errors.New("account transfer not found")
)

type MoneyAccountRepository interface {
	Create(account *model.MoneyAccount) error
	GetByID(id string) (*model.MoneyAccount, error)
	GetBySpaceID(spaceID string) ([]*model.MoneyAccount, error)
	Update(account *model.MoneyAccount) error
	Delete(id string) error

	CreateTransfer(transfer *model.AccountTransfer) error
	GetTransfersByAccountID(accountID string) ([]*model.AccountTransfer, error)
	DeleteTransfer(id string) error

	GetAccountBalance(accountID string) (decimal.Decimal, error)
	GetTotalAllocatedForSpace(spaceID string) (decimal.Decimal, error)

	GetTransfersBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.AccountTransferWithAccount, error)
	CountTransfersBySpaceID(spaceID string) (int, error)
}

type moneyAccountRepository struct {
	db *sqlx.DB
}

func NewMoneyAccountRepository(db *sqlx.DB) MoneyAccountRepository {
	return &moneyAccountRepository{db: db}
}

func (r *moneyAccountRepository) Create(account *model.MoneyAccount) error {
	query := `INSERT INTO money_accounts (id, space_id, name, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6);`
	_, err := r.db.Exec(query, account.ID, account.SpaceID, account.Name, account.CreatedBy, account.CreatedAt, account.UpdatedAt)
	return err
}

func (r *moneyAccountRepository) GetByID(id string) (*model.MoneyAccount, error) {
	account := &model.MoneyAccount{}
	query := `SELECT * FROM money_accounts WHERE id = $1;`
	err := r.db.Get(account, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrMoneyAccountNotFound
	}
	return account, err
}

func (r *moneyAccountRepository) GetBySpaceID(spaceID string) ([]*model.MoneyAccount, error) {
	var accounts []*model.MoneyAccount
	query := `SELECT * FROM money_accounts WHERE space_id = $1 ORDER BY created_at DESC;`
	err := r.db.Select(&accounts, query, spaceID)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *moneyAccountRepository) Update(account *model.MoneyAccount) error {
	account.UpdatedAt = time.Now()
	query := `UPDATE money_accounts SET name = $1, updated_at = $2 WHERE id = $3;`
	result, err := r.db.Exec(query, account.Name, account.UpdatedAt, account.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrMoneyAccountNotFound
	}
	return err
}

func (r *moneyAccountRepository) Delete(id string) error {
	query := `DELETE FROM money_accounts WHERE id = $1;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrMoneyAccountNotFound
	}
	return err
}

func (r *moneyAccountRepository) CreateTransfer(transfer *model.AccountTransfer) error {
	query := `INSERT INTO account_transfers (id, account_id, amount, direction, note, recurring_deposit_id, created_by, created_at, amount_cents) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0);`
	_, err := r.db.Exec(query, transfer.ID, transfer.AccountID, transfer.Amount, transfer.Direction, transfer.Note, transfer.RecurringDepositID, transfer.CreatedBy, transfer.CreatedAt)
	return err
}

func (r *moneyAccountRepository) GetTransfersByAccountID(accountID string) ([]*model.AccountTransfer, error) {
	var transfers []*model.AccountTransfer
	query := `SELECT * FROM account_transfers WHERE account_id = $1 ORDER BY created_at DESC;`
	err := r.db.Select(&transfers, query, accountID)
	if err != nil {
		return nil, err
	}
	return transfers, nil
}

func (r *moneyAccountRepository) DeleteTransfer(id string) error {
	query := `DELETE FROM account_transfers WHERE id = $1;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrTransferNotFound
	}
	return err
}

func (r *moneyAccountRepository) GetAccountBalance(accountID string) (decimal.Decimal, error) {
	var balance decimal.Decimal
	query := `SELECT COALESCE(SUM(CASE WHEN direction = 'deposit' THEN CAST(amount AS DECIMAL) ELSE -CAST(amount AS DECIMAL) END), 0) FROM account_transfers WHERE account_id = $1;`
	err := r.db.Get(&balance, query, accountID)
	return balance, err
}

func (r *moneyAccountRepository) GetTotalAllocatedForSpace(spaceID string) (decimal.Decimal, error) {
	var total decimal.Decimal
	query := `SELECT COALESCE(SUM(CASE WHEN t.direction = 'deposit' THEN CAST(t.amount AS DECIMAL) ELSE -CAST(t.amount AS DECIMAL) END), 0)
		FROM account_transfers t
		JOIN money_accounts a ON t.account_id = a.id
		WHERE a.space_id = $1;`
	err := r.db.Get(&total, query, spaceID)
	return total, err
}

func (r *moneyAccountRepository) GetTransfersBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.AccountTransferWithAccount, error) {
	var transfers []*model.AccountTransferWithAccount
	query := `SELECT t.id, t.account_id, t.amount, t.direction, t.note,
		t.recurring_deposit_id, t.created_by, t.created_at, a.name AS account_name
		FROM account_transfers t
		JOIN money_accounts a ON t.account_id = a.id
		WHERE a.space_id = $1
		ORDER BY t.created_at DESC
		LIMIT $2 OFFSET $3;`
	err := r.db.Select(&transfers, query, spaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return transfers, nil
}

func (r *moneyAccountRepository) CountTransfersBySpaceID(spaceID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM account_transfers t
		JOIN money_accounts a ON t.account_id = a.id
		WHERE a.space_id = $1;`
	err := r.db.Get(&count, query, spaceID)
	return count, err
}
