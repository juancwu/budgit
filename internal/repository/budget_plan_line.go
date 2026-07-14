package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var ErrBudgetPlanLineNotFound = errors.New("budget plan line not found")

type BudgetPlanLineRepository interface {
	Create(l *model.BudgetPlanLine) error
	ByID(id string) (*model.BudgetPlanLine, error)
	ByPlanID(planID string) ([]*model.BudgetPlanLine, error)
	Update(id, label string, amount decimal.Decimal) error
	Delete(id string) error
}

type budgetPlanLineRepository struct {
	db *sqlx.DB
}

func NewBudgetPlanLineRepository(db *sqlx.DB) BudgetPlanLineRepository {
	return &budgetPlanLineRepository{db: db}
}

func (r *budgetPlanLineRepository) Create(l *model.BudgetPlanLine) error {
	query := `INSERT INTO budget_plan_lines (
	    id, plan_id, kind, label, amount, sort_order, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`
	_, err := r.db.Exec(query,
		l.ID, l.PlanID, l.Kind, l.Label, l.Amount, l.SortOrder, l.CreatedAt, l.UpdatedAt,
	)
	return err
}

func (r *budgetPlanLineRepository) ByID(id string) (*model.BudgetPlanLine, error) {
	l := &model.BudgetPlanLine{}
	err := r.db.Get(l, `SELECT * FROM budget_plan_lines WHERE id = $1;`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBudgetPlanLineNotFound
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (r *budgetPlanLineRepository) ByPlanID(planID string) ([]*model.BudgetPlanLine, error) {
	var lines []*model.BudgetPlanLine
	err := r.db.Select(&lines,
		`SELECT * FROM budget_plan_lines WHERE plan_id = $1 ORDER BY sort_order ASC, created_at ASC;`,
		planID,
	)
	return lines, err
}

func (r *budgetPlanLineRepository) Update(id, label string, amount decimal.Decimal) error {
	res, err := r.db.Exec(
		`UPDATE budget_plan_lines
		 SET label = $1, amount = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3;`,
		label, amount, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrBudgetPlanLineNotFound
	}
	return nil
}

func (r *budgetPlanLineRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM budget_plan_lines WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrBudgetPlanLineNotFound
	}
	return nil
}
