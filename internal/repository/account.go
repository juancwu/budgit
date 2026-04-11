package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrAccountNotFound = errors.New("account not found")

type AccountRepository interface {
	Create(account *model.Account) error
	ByID(id string) (*model.Account, error)
	BySpaceID(spaceID string) ([]*model.Account, error)
	Delete(id string) error
}

type accountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(account *model.Account) error {
	query := `INSERT INTO accounts (id, name, space_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5);`
	_, err := r.db.Exec(query, account.ID, account.Name, account.SpaceID, account.CreatedAt, account.UpdatedAt)
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

func (r *accountRepository) Delete(id string) error {
	query := `DELETE FROM accounts WHERE id = $1;`
	_, err := r.db.Exec(query, id)
	return err
}
