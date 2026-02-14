package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrRecurringExpenseNotFound = errors.New("recurring expense not found")
)

type RecurringExpenseRepository interface {
	Create(re *model.RecurringExpense, tagIDs []string) error
	GetByID(id string) (*model.RecurringExpense, error)
	GetBySpaceID(spaceID string) ([]*model.RecurringExpense, error)
	GetTagsByRecurringExpenseIDs(ids []string) (map[string][]*model.Tag, error)
	GetPaymentMethodsByRecurringExpenseIDs(ids []string) (map[string]*model.PaymentMethod, error)
	Update(re *model.RecurringExpense, tagIDs []string) error
	Delete(id string) error
	SetActive(id string, active bool) error
	GetDueRecurrences(now time.Time) ([]*model.RecurringExpense, error)
	GetDueRecurrencesForSpace(spaceID string, now time.Time) ([]*model.RecurringExpense, error)
	UpdateNextOccurrence(id string, next time.Time) error
	Deactivate(id string) error
}

type recurringExpenseRepository struct {
	db *sqlx.DB
}

func NewRecurringExpenseRepository(db *sqlx.DB) RecurringExpenseRepository {
	return &recurringExpenseRepository{db: db}
}

func (r *recurringExpenseRepository) Create(re *model.RecurringExpense, tagIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO recurring_expenses (id, space_id, created_by, description, amount_cents, type, payment_method_id, frequency, start_date, end_date, next_occurrence, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);`
	_, err = tx.Exec(query, re.ID, re.SpaceID, re.CreatedBy, re.Description, re.AmountCents, re.Type, re.PaymentMethodID, re.Frequency, re.StartDate, re.EndDate, re.NextOccurrence, re.IsActive, re.CreatedAt, re.UpdatedAt)
	if err != nil {
		return err
	}

	if len(tagIDs) > 0 {
		tagQuery := `INSERT INTO recurring_expense_tags (recurring_expense_id, tag_id) VALUES ($1, $2);`
		for _, tagID := range tagIDs {
			if _, err := tx.Exec(tagQuery, re.ID, tagID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *recurringExpenseRepository) GetByID(id string) (*model.RecurringExpense, error) {
	re := &model.RecurringExpense{}
	query := `SELECT * FROM recurring_expenses WHERE id = $1;`
	err := r.db.Get(re, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrRecurringExpenseNotFound
	}
	return re, err
}

func (r *recurringExpenseRepository) GetBySpaceID(spaceID string) ([]*model.RecurringExpense, error) {
	var results []*model.RecurringExpense
	query := `SELECT * FROM recurring_expenses WHERE space_id = $1 ORDER BY is_active DESC, next_occurrence ASC;`
	err := r.db.Select(&results, query, spaceID)
	return results, err
}

func (r *recurringExpenseRepository) GetTagsByRecurringExpenseIDs(ids []string) (map[string][]*model.Tag, error) {
	if len(ids) == 0 {
		return make(map[string][]*model.Tag), nil
	}

	type row struct {
		RecurringExpenseID string  `db:"recurring_expense_id"`
		ID                 string  `db:"id"`
		SpaceID            string  `db:"space_id"`
		Name               string  `db:"name"`
		Color              *string `db:"color"`
	}

	query, args, err := sqlx.In(`
		SELECT ret.recurring_expense_id, t.id, t.space_id, t.name, t.color
		FROM recurring_expense_tags ret
		JOIN tags t ON ret.tag_id = t.id
		WHERE ret.recurring_expense_id IN (?)
		ORDER BY t.name;
	`, ids)
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
		result[rw.RecurringExpenseID] = append(result[rw.RecurringExpenseID], &model.Tag{
			ID:      rw.ID,
			SpaceID: rw.SpaceID,
			Name:    rw.Name,
			Color:   rw.Color,
		})
	}
	return result, nil
}

func (r *recurringExpenseRepository) GetPaymentMethodsByRecurringExpenseIDs(ids []string) (map[string]*model.PaymentMethod, error) {
	if len(ids) == 0 {
		return make(map[string]*model.PaymentMethod), nil
	}

	type row struct {
		RecurringExpenseID string                  `db:"recurring_expense_id"`
		ID                 string                  `db:"id"`
		SpaceID            string                  `db:"space_id"`
		Name               string                  `db:"name"`
		Type               model.PaymentMethodType `db:"type"`
		LastFour           *string                 `db:"last_four"`
	}

	query, args, err := sqlx.In(`
		SELECT re.id AS recurring_expense_id, pm.id, pm.space_id, pm.name, pm.type, pm.last_four
		FROM recurring_expenses re
		JOIN payment_methods pm ON re.payment_method_id = pm.id
		WHERE re.id IN (?) AND re.payment_method_id IS NOT NULL;
	`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var rows []row
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}

	result := make(map[string]*model.PaymentMethod)
	for _, rw := range rows {
		result[rw.RecurringExpenseID] = &model.PaymentMethod{
			ID:       rw.ID,
			SpaceID:  rw.SpaceID,
			Name:     rw.Name,
			Type:     rw.Type,
			LastFour: rw.LastFour,
		}
	}
	return result, nil
}

func (r *recurringExpenseRepository) Update(re *model.RecurringExpense, tagIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE recurring_expenses SET description = $1, amount_cents = $2, type = $3, payment_method_id = $4, frequency = $5, start_date = $6, end_date = $7, next_occurrence = $8, updated_at = $9 WHERE id = $10;`
	_, err = tx.Exec(query, re.Description, re.AmountCents, re.Type, re.PaymentMethodID, re.Frequency, re.StartDate, re.EndDate, re.NextOccurrence, re.UpdatedAt, re.ID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM recurring_expense_tags WHERE recurring_expense_id = $1;`, re.ID)
	if err != nil {
		return err
	}

	if len(tagIDs) > 0 {
		tagQuery := `INSERT INTO recurring_expense_tags (recurring_expense_id, tag_id) VALUES ($1, $2);`
		for _, tagID := range tagIDs {
			if _, err := tx.Exec(tagQuery, re.ID, tagID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *recurringExpenseRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM recurring_expenses WHERE id = $1;`, id)
	return err
}

func (r *recurringExpenseRepository) SetActive(id string, active bool) error {
	_, err := r.db.Exec(`UPDATE recurring_expenses SET is_active = $1, updated_at = $2 WHERE id = $3;`, active, time.Now(), id)
	return err
}

func (r *recurringExpenseRepository) GetDueRecurrences(now time.Time) ([]*model.RecurringExpense, error) {
	var results []*model.RecurringExpense
	query := `SELECT * FROM recurring_expenses WHERE is_active = true AND next_occurrence <= $1;`
	err := r.db.Select(&results, query, now)
	return results, err
}

func (r *recurringExpenseRepository) GetDueRecurrencesForSpace(spaceID string, now time.Time) ([]*model.RecurringExpense, error) {
	var results []*model.RecurringExpense
	query := `SELECT * FROM recurring_expenses WHERE is_active = true AND space_id = $1 AND next_occurrence <= $2;`
	err := r.db.Select(&results, query, spaceID, now)
	return results, err
}

func (r *recurringExpenseRepository) UpdateNextOccurrence(id string, next time.Time) error {
	_, err := r.db.Exec(`UPDATE recurring_expenses SET next_occurrence = $1, updated_at = $2 WHERE id = $3;`, next, time.Now(), id)
	return err
}

func (r *recurringExpenseRepository) Deactivate(id string) error {
	return r.SetActive(id, false)
}
