package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrHoldingNotFound = errors.New("investment holding not found")

type InvestmentHoldingRepository interface {
	Create(h *model.InvestmentHolding) error
	ByID(id string) (*model.InvestmentHolding, error)
	ByAccountID(accountID string) ([]*model.InvestmentHolding, error)
	Update(id, symbol, displayName string) error
	Delete(id string) error
}

type investmentHoldingRepository struct {
	db *sqlx.DB
}

func NewInvestmentHoldingRepository(db *sqlx.DB) InvestmentHoldingRepository {
	return &investmentHoldingRepository{db: db}
}

func (r *investmentHoldingRepository) Create(h *model.InvestmentHolding) error {
	query := `INSERT INTO investment_holdings (id, account_id, symbol, display_name, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6);`
	_, err := r.db.Exec(query, h.ID, h.AccountID, h.Symbol, h.DisplayName, h.CreatedAt, h.UpdatedAt)
	return err
}

func (r *investmentHoldingRepository) ByID(id string) (*model.InvestmentHolding, error) {
	h := &model.InvestmentHolding{}
	query := `SELECT * FROM investment_holdings WHERE id = $1;`
	err := r.db.Get(h, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrHoldingNotFound
	}
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (r *investmentHoldingRepository) ByAccountID(accountID string) ([]*model.InvestmentHolding, error) {
	var holdings []*model.InvestmentHolding
	query := `SELECT * FROM investment_holdings WHERE account_id = $1 ORDER BY symbol ASC;`
	if err := r.db.Select(&holdings, query, accountID); err != nil {
		return nil, err
	}
	return holdings, nil
}

func (r *investmentHoldingRepository) Update(id, symbol, displayName string) error {
	query := `UPDATE investment_holdings
	          SET symbol = $1, display_name = $2, updated_at = CURRENT_TIMESTAMP
	          WHERE id = $3;`
	res, err := r.db.Exec(query, symbol, displayName, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrHoldingNotFound
	}
	return nil
}

func (r *investmentHoldingRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM investment_holdings WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrHoldingNotFound
	}
	return nil
}
