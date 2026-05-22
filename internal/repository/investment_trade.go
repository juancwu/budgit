package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var ErrTradeNotFound = errors.New("investment trade not found")

type InvestmentTradeRepository interface {
	Create(t *model.InvestmentTrade) error
	ByID(id string) (*model.InvestmentTrade, error)
	ByHoldingID(holdingID string) ([]*model.InvestmentTrade, error)
	Update(id string, quantity, pricePerUnit decimal.Decimal, fees *decimal.Decimal, occurredAt time.Time, notes *string) error
	Delete(id string) error
}

type investmentTradeRepository struct {
	db *sqlx.DB
}

func NewInvestmentTradeRepository(db *sqlx.DB) InvestmentTradeRepository {
	return &investmentTradeRepository{db: db}
}

func (r *investmentTradeRepository) Create(t *model.InvestmentTrade) error {
	query := `INSERT INTO investment_trades (id, holding_id, type, quantity, price_per_unit, fees, occurred_at, notes, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`
	_, err := r.db.Exec(query, t.ID, t.HoldingID, t.Type, t.Quantity, t.PricePerUnit, t.Fees, t.OccurredAt, t.Notes, t.CreatedAt)
	return err
}

func (r *investmentTradeRepository) ByID(id string) (*model.InvestmentTrade, error) {
	t := &model.InvestmentTrade{}
	query := `SELECT * FROM investment_trades WHERE id = $1;`
	err := r.db.Get(t, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrTradeNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *investmentTradeRepository) ByHoldingID(holdingID string) ([]*model.InvestmentTrade, error) {
	var trades []*model.InvestmentTrade
	query := `SELECT * FROM investment_trades WHERE holding_id = $1 ORDER BY occurred_at ASC, created_at ASC;`
	if err := r.db.Select(&trades, query, holdingID); err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *investmentTradeRepository) Update(id string, quantity, pricePerUnit decimal.Decimal, fees *decimal.Decimal, occurredAt time.Time, notes *string) error {
	query := `UPDATE investment_trades
	          SET quantity = $1, price_per_unit = $2, fees = $3, occurred_at = $4, notes = $5
	          WHERE id = $6;`
	res, err := r.db.Exec(query, quantity, pricePerUnit, fees, occurredAt, notes, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTradeNotFound
	}
	return nil
}

func (r *investmentTradeRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM investment_trades WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTradeNotFound
	}
	return nil
}
