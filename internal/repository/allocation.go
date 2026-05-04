package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var ErrAllocationNotFound = errors.New("allocation not found")

type AllocationRepository interface {
	Create(allocation *model.Allocation) error
	ByID(id string) (*model.Allocation, error)
	ByAccountID(accountID string) ([]*model.Allocation, error)
	SumByAccountID(accountID string) (decimal.Decimal, error)
	Update(id, name string, amount decimal.Decimal, target *decimal.Decimal) error
	Delete(id string) error
}

type allocationRepository struct {
	db *sqlx.DB
}

func NewAllocationRepository(db *sqlx.DB) AllocationRepository {
	return &allocationRepository{db: db}
}

func (r *allocationRepository) Create(a *model.Allocation) error {
	query := `INSERT INTO allocations (id, account_id, name, amount, target_amount, sort_order, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`
	_, err := r.db.Exec(query, a.ID, a.AccountID, a.Name, a.Amount, a.TargetAmount, a.SortOrder, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *allocationRepository) ByID(id string) (*model.Allocation, error) {
	a := &model.Allocation{}
	query := `SELECT * FROM allocations WHERE id = $1;`
	err := r.db.Get(a, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrAllocationNotFound
	}
	return a, err
}

func (r *allocationRepository) ByAccountID(accountID string) ([]*model.Allocation, error) {
	var out []*model.Allocation
	query := `SELECT * FROM allocations WHERE account_id = $1 ORDER BY sort_order ASC, created_at ASC;`
	err := r.db.Select(&out, query, accountID)
	return out, err
}

func (r *allocationRepository) SumByAccountID(accountID string) (decimal.Decimal, error) {
	var sum decimal.Decimal
	query := `SELECT COALESCE(SUM(amount::numeric), 0)::text FROM allocations WHERE account_id = $1;`
	if err := r.db.Get(&sum, query, accountID); err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

func (r *allocationRepository) Update(id, name string, amount decimal.Decimal, target *decimal.Decimal) error {
	query := `UPDATE allocations
	          SET name = $1, amount = $2, target_amount = $3, updated_at = CURRENT_TIMESTAMP
	          WHERE id = $4;`
	res, err := r.db.Exec(query, name, amount, target, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrAllocationNotFound
	}
	return nil
}

func (r *allocationRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM allocations WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrAllocationNotFound
	}
	return nil
}
