package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrBudgetNotFound = errors.New("budget not found")
)

type BudgetRepository interface {
	Create(budget *model.Budget) error
	GetByID(id string) (*model.Budget, error)
	GetBySpaceID(spaceID string) ([]*model.Budget, error)
	GetSpentForBudget(spaceID, tagID string, periodStart, periodEnd time.Time) (int, error)
	Update(budget *model.Budget) error
	Delete(id string) error
}

type budgetRepository struct {
	db *sqlx.DB
}

func NewBudgetRepository(db *sqlx.DB) BudgetRepository {
	return &budgetRepository{db: db}
}

func (r *budgetRepository) Create(budget *model.Budget) error {
	query := `INSERT INTO budgets (id, space_id, tag_id, amount_cents, period, start_date, end_date, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);`
	_, err := r.db.Exec(query, budget.ID, budget.SpaceID, budget.TagID, budget.AmountCents, budget.Period, budget.StartDate, budget.EndDate, budget.IsActive, budget.CreatedBy, budget.CreatedAt, budget.UpdatedAt)
	return err
}

func (r *budgetRepository) GetByID(id string) (*model.Budget, error) {
	budget := &model.Budget{}
	err := r.db.Get(budget, `SELECT * FROM budgets WHERE id = $1;`, id)
	if err == sql.ErrNoRows {
		return nil, ErrBudgetNotFound
	}
	return budget, err
}

func (r *budgetRepository) GetBySpaceID(spaceID string) ([]*model.Budget, error) {
	var budgets []*model.Budget
	err := r.db.Select(&budgets, `SELECT * FROM budgets WHERE space_id = $1 ORDER BY created_at DESC;`, spaceID)
	return budgets, err
}

func (r *budgetRepository) GetSpentForBudget(spaceID, tagID string, periodStart, periodEnd time.Time) (int, error) {
	var spent int
	query := `
		SELECT COALESCE(SUM(e.amount_cents), 0)
		FROM expenses e
		JOIN expense_tags et ON e.id = et.expense_id
		WHERE e.space_id = $1 AND et.tag_id = $2 AND e.type = 'expense'
		  AND e.date >= $3 AND e.date <= $4;
	`
	err := r.db.Get(&spent, query, spaceID, tagID, periodStart, periodEnd)
	return spent, err
}

func (r *budgetRepository) Update(budget *model.Budget) error {
	query := `UPDATE budgets SET tag_id = $1, amount_cents = $2, period = $3, start_date = $4, end_date = $5, is_active = $6, updated_at = $7 WHERE id = $8;`
	_, err := r.db.Exec(query, budget.TagID, budget.AmountCents, budget.Period, budget.StartDate, budget.EndDate, budget.IsActive, budget.UpdatedAt, budget.ID)
	return err
}

func (r *budgetRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM budgets WHERE id = $1;`, id)
	return err
}
