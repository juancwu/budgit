package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrExpenseNotFound = errors.New("expense not found")
)

type ExpenseRepository interface {
	Create(expense *model.Expense, tagIDs []string, itemIDs []string) error
	GetByID(id string) (*model.Expense, error)
	GetBySpaceID(spaceID string) ([]*model.Expense, error)
	GetBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.Expense, error)
	CountBySpaceID(spaceID string) (int, error)
	GetExpensesByTag(spaceID string, fromDate, toDate time.Time) ([]*model.TagExpenseSummary, error)
	GetTagsByExpenseIDs(expenseIDs []string) (map[string][]*model.Tag, error)
	Update(expense *model.Expense, tagIDs []string) error
	Delete(id string) error
}

type expenseRepository struct {
	db *sqlx.DB
}

func NewExpenseRepository(db *sqlx.DB) ExpenseRepository {
	return &expenseRepository{db: db}
}

func (r *expenseRepository) Create(expense *model.Expense, tagIDs []string, itemIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert Expense
	queryExpense := `INSERT INTO expenses (id, space_id, created_by, description, amount_cents, type, date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`
	_, err = tx.Exec(queryExpense, expense.ID, expense.SpaceID, expense.CreatedBy, expense.Description, expense.AmountCents, expense.Type, expense.Date, expense.CreatedAt, expense.UpdatedAt)
	if err != nil {
		return err
	}

	// Insert Tags
	if len(tagIDs) > 0 {
		queryTags := `INSERT INTO expense_tags (expense_id, tag_id) VALUES ($1, $2);`
		for _, tagID := range tagIDs {
			_, err := tx.Exec(queryTags, expense.ID, tagID)
			if err != nil {
				return err
			}
		}
	}

	// Insert Items
	if len(itemIDs) > 0 {
		queryItems := `INSERT INTO expense_items (expense_id, item_id) VALUES ($1, $2);`
		for _, itemID := range itemIDs {
			_, err := tx.Exec(queryItems, expense.ID, itemID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *expenseRepository) GetByID(id string) (*model.Expense, error) {
	expense := &model.Expense{}
	query := `SELECT * FROM expenses WHERE id = $1;`
	err := r.db.Get(expense, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrExpenseNotFound
	}
	return expense, err
}

func (r *expenseRepository) GetBySpaceID(spaceID string) ([]*model.Expense, error) {
	var expenses []*model.Expense
	query := `SELECT * FROM expenses WHERE space_id = $1 ORDER BY date DESC, created_at DESC;`
	err := r.db.Select(&expenses, query, spaceID)
	if err != nil {
		return nil, err
	}
	return expenses, nil
}

func (r *expenseRepository) GetBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.Expense, error) {
	var expenses []*model.Expense
	query := `SELECT * FROM expenses WHERE space_id = $1 ORDER BY date DESC, created_at DESC LIMIT $2 OFFSET $3;`
	err := r.db.Select(&expenses, query, spaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return expenses, nil
}

func (r *expenseRepository) CountBySpaceID(spaceID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM expenses WHERE space_id = $1;`, spaceID)
	return count, err
}

func (r *expenseRepository) GetExpensesByTag(spaceID string, fromDate, toDate time.Time) ([]*model.TagExpenseSummary, error) {
	var summaries []*model.TagExpenseSummary
	query := `
		SELECT
			t.id as tag_id,
			t.name as tag_name,
			t.color as tag_color,
			SUM(e.amount_cents) as total_amount
		FROM expenses e
		JOIN expense_tags et ON e.id = et.expense_id
		JOIN tags t ON et.tag_id = t.id
		WHERE e.space_id = $1 AND e.type = 'expense' AND e.date >= $2 AND e.date <= $3
		GROUP BY t.id, t.name, t.color
		ORDER BY total_amount DESC;
	`
	err := r.db.Select(&summaries, query, spaceID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

func (r *expenseRepository) GetTagsByExpenseIDs(expenseIDs []string) (map[string][]*model.Tag, error) {
	if len(expenseIDs) == 0 {
		return make(map[string][]*model.Tag), nil
	}

	type row struct {
		ExpenseID string  `db:"expense_id"`
		ID        string  `db:"id"`
		SpaceID   string  `db:"space_id"`
		Name      string  `db:"name"`
		Color     *string `db:"color"`
	}

	query, args, err := sqlx.In(`
		SELECT et.expense_id, t.id, t.space_id, t.name, t.color
		FROM expense_tags et
		JOIN tags t ON et.tag_id = t.id
		WHERE et.expense_id IN (?)
		ORDER BY t.name;
	`, expenseIDs)
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
		result[rw.ExpenseID] = append(result[rw.ExpenseID], &model.Tag{
			ID:      rw.ID,
			SpaceID: rw.SpaceID,
			Name:    rw.Name,
			Color:   rw.Color,
		})
	}
	return result, nil
}

func (r *expenseRepository) Update(expense *model.Expense, tagIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE expenses SET description = $1, amount_cents = $2, type = $3, date = $4, updated_at = $5 WHERE id = $6;`
	_, err = tx.Exec(query, expense.Description, expense.AmountCents, expense.Type, expense.Date, expense.UpdatedAt, expense.ID)
	if err != nil {
		return err
	}

	// Replace tags: delete all existing, re-insert
	_, err = tx.Exec(`DELETE FROM expense_tags WHERE expense_id = $1;`, expense.ID)
	if err != nil {
		return err
	}

	if len(tagIDs) > 0 {
		insertTag := `INSERT INTO expense_tags (expense_id, tag_id) VALUES ($1, $2);`
		for _, tagID := range tagIDs {
			if _, err := tx.Exec(insertTag, expense.ID, tagID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *expenseRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM expenses WHERE id = $1;`, id)
	return err
}
