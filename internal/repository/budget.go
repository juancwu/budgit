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
	Create(budget *model.Budget, tagIDs []string) error
	GetByID(id string) (*model.Budget, error)
	GetBySpaceID(spaceID string) ([]*model.Budget, error)
	GetSpentForBudget(spaceID string, tagIDs []string, periodStart, periodEnd time.Time) (int, error)
	GetTagsByBudgetIDs(budgetIDs []string) (map[string][]*model.Tag, error)
	Update(budget *model.Budget, tagIDs []string) error
	Delete(id string) error
}

type budgetRepository struct {
	db *sqlx.DB
}

func NewBudgetRepository(db *sqlx.DB) BudgetRepository {
	return &budgetRepository{db: db}
}

func (r *budgetRepository) Create(budget *model.Budget, tagIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO budgets (id, space_id, amount_cents, period, start_date, end_date, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`
	_, err = tx.Exec(query, budget.ID, budget.SpaceID, budget.AmountCents, budget.Period, budget.StartDate, budget.EndDate, budget.IsActive, budget.CreatedBy, budget.CreatedAt, budget.UpdatedAt)
	if err != nil {
		return err
	}

	if len(tagIDs) > 0 {
		tagQuery := `INSERT INTO budget_tags (budget_id, tag_id) VALUES ($1, $2);`
		for _, tagID := range tagIDs {
			if _, err := tx.Exec(tagQuery, budget.ID, tagID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
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

func (r *budgetRepository) GetSpentForBudget(spaceID string, tagIDs []string, periodStart, periodEnd time.Time) (int, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}

	query, args, err := sqlx.In(`
		SELECT COALESCE(SUM(e.amount_cents), 0)
		FROM expenses e
		WHERE e.space_id = ? AND e.type = 'expense' AND e.date >= ? AND e.date <= ?
		  AND EXISTS (SELECT 1 FROM expense_tags et WHERE et.expense_id = e.id AND et.tag_id IN (?))
	`, spaceID, periodStart, periodEnd, tagIDs)
	if err != nil {
		return 0, err
	}
	query = r.db.Rebind(query)

	var spent int
	err = r.db.Get(&spent, query, args...)
	return spent, err
}

func (r *budgetRepository) GetTagsByBudgetIDs(budgetIDs []string) (map[string][]*model.Tag, error) {
	if len(budgetIDs) == 0 {
		return make(map[string][]*model.Tag), nil
	}

	type row struct {
		BudgetID string  `db:"budget_id"`
		ID       string  `db:"id"`
		SpaceID  string  `db:"space_id"`
		Name     string  `db:"name"`
		Color    *string `db:"color"`
	}

	query, args, err := sqlx.In(`
		SELECT bt.budget_id, t.id, t.space_id, t.name, t.color
		FROM budget_tags bt
		JOIN tags t ON bt.tag_id = t.id
		WHERE bt.budget_id IN (?)
		ORDER BY t.name;
	`, budgetIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var rows []row
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}

	result := make(map[string][]*model.Tag)
	for _, rw := range rows {
		result[rw.BudgetID] = append(result[rw.BudgetID], &model.Tag{
			ID:      rw.ID,
			SpaceID: rw.SpaceID,
			Name:    rw.Name,
			Color:   rw.Color,
		})
	}
	return result, nil
}

func (r *budgetRepository) Update(budget *model.Budget, tagIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE budgets SET amount_cents = $1, period = $2, start_date = $3, end_date = $4, is_active = $5, updated_at = $6 WHERE id = $7;`
	_, err = tx.Exec(query, budget.AmountCents, budget.Period, budget.StartDate, budget.EndDate, budget.IsActive, budget.UpdatedAt, budget.ID)
	if err != nil {
		return err
	}

	// Replace tags: delete old, insert new
	if _, err := tx.Exec(`DELETE FROM budget_tags WHERE budget_id = $1;`, budget.ID); err != nil {
		return err
	}

	if len(tagIDs) > 0 {
		tagQuery := `INSERT INTO budget_tags (budget_id, tag_id) VALUES ($1, $2);`
		for _, tagID := range tagIDs {
			if _, err := tx.Exec(tagQuery, budget.ID, tagID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *budgetRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM budgets WHERE id = $1;`, id)
	return err
}
