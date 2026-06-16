package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrBudgetPlanNotFound = errors.New("budget plan not found")

type BudgetPlanRepository interface {
	Create(p *model.BudgetPlan) error
	ByID(id string) (*model.BudgetPlan, error)
	BySpaceID(spaceID string) ([]*model.BudgetPlan, error)
	Rename(id, name string) error
	Delete(id string) error
}

type budgetPlanRepository struct {
	db *sqlx.DB
}

func NewBudgetPlanRepository(db *sqlx.DB) BudgetPlanRepository {
	return &budgetPlanRepository{db: db}
}

func (r *budgetPlanRepository) Create(p *model.BudgetPlan) error {
	query := `INSERT INTO budget_plans (id, space_id, name, note, currency, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7);`
	_, err := r.db.Exec(query, p.ID, p.SpaceID, p.Name, p.Note, p.Currency, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *budgetPlanRepository) ByID(id string) (*model.BudgetPlan, error) {
	p := &model.BudgetPlan{}
	err := r.db.Get(p, `SELECT * FROM budget_plans WHERE id = $1;`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBudgetPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *budgetPlanRepository) BySpaceID(spaceID string) ([]*model.BudgetPlan, error) {
	var plans []*model.BudgetPlan
	err := r.db.Select(&plans, `SELECT * FROM budget_plans WHERE space_id = $1 ORDER BY created_at DESC;`, spaceID)
	return plans, err
}

func (r *budgetPlanRepository) Rename(id, name string) error {
	res, err := r.db.Exec(
		`UPDATE budget_plans SET name = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;`,
		name, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrBudgetPlanNotFound
	}
	return nil
}

func (r *budgetPlanRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM budget_plans WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrBudgetPlanNotFound
	}
	return nil
}
