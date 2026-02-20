package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrRecurringDepositNotFound = errors.New("recurring deposit not found")
)

type RecurringDepositRepository interface {
	Create(rd *model.RecurringDeposit) error
	GetByID(id string) (*model.RecurringDeposit, error)
	GetBySpaceID(spaceID string) ([]*model.RecurringDeposit, error)
	Update(rd *model.RecurringDeposit) error
	Delete(id string) error
	SetActive(id string, active bool) error
	GetDueRecurrences(now time.Time) ([]*model.RecurringDeposit, error)
	GetDueRecurrencesForSpace(spaceID string, now time.Time) ([]*model.RecurringDeposit, error)
	UpdateNextOccurrence(id string, next time.Time) error
	Deactivate(id string) error
}

type recurringDepositRepository struct {
	db *sqlx.DB
}

func NewRecurringDepositRepository(db *sqlx.DB) RecurringDepositRepository {
	return &recurringDepositRepository{db: db}
}

func (r *recurringDepositRepository) Create(rd *model.RecurringDeposit) error {
	query := `INSERT INTO recurring_deposits (id, space_id, account_id, amount_cents, frequency, start_date, end_date, next_occurrence, is_active, title, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);`
	_, err := r.db.Exec(query, rd.ID, rd.SpaceID, rd.AccountID, rd.AmountCents, rd.Frequency, rd.StartDate, rd.EndDate, rd.NextOccurrence, rd.IsActive, rd.Title, rd.CreatedBy, rd.CreatedAt, rd.UpdatedAt)
	return err
}

func (r *recurringDepositRepository) GetByID(id string) (*model.RecurringDeposit, error) {
	rd := &model.RecurringDeposit{}
	query := `SELECT * FROM recurring_deposits WHERE id = $1;`
	err := r.db.Get(rd, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrRecurringDepositNotFound
	}
	return rd, err
}

func (r *recurringDepositRepository) GetBySpaceID(spaceID string) ([]*model.RecurringDeposit, error) {
	var results []*model.RecurringDeposit
	query := `SELECT * FROM recurring_deposits WHERE space_id = $1 ORDER BY is_active DESC, next_occurrence ASC;`
	err := r.db.Select(&results, query, spaceID)
	return results, err
}

func (r *recurringDepositRepository) Update(rd *model.RecurringDeposit) error {
	query := `UPDATE recurring_deposits SET account_id = $1, amount_cents = $2, frequency = $3, start_date = $4, end_date = $5, next_occurrence = $6, title = $7, updated_at = $8 WHERE id = $9;`
	result, err := r.db.Exec(query, rd.AccountID, rd.AmountCents, rd.Frequency, rd.StartDate, rd.EndDate, rd.NextOccurrence, rd.Title, rd.UpdatedAt, rd.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrRecurringDepositNotFound
	}
	return err
}

func (r *recurringDepositRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM recurring_deposits WHERE id = $1;`, id)
	return err
}

func (r *recurringDepositRepository) SetActive(id string, active bool) error {
	_, err := r.db.Exec(`UPDATE recurring_deposits SET is_active = $1, updated_at = $2 WHERE id = $3;`, active, time.Now(), id)
	return err
}

func (r *recurringDepositRepository) GetDueRecurrences(now time.Time) ([]*model.RecurringDeposit, error) {
	var results []*model.RecurringDeposit
	query := `SELECT * FROM recurring_deposits WHERE is_active = true AND next_occurrence <= $1;`
	err := r.db.Select(&results, query, now)
	return results, err
}

func (r *recurringDepositRepository) GetDueRecurrencesForSpace(spaceID string, now time.Time) ([]*model.RecurringDeposit, error) {
	var results []*model.RecurringDeposit
	query := `SELECT * FROM recurring_deposits WHERE is_active = true AND space_id = $1 AND next_occurrence <= $2;`
	err := r.db.Select(&results, query, spaceID, now)
	return results, err
}

func (r *recurringDepositRepository) UpdateNextOccurrence(id string, next time.Time) error {
	_, err := r.db.Exec(`UPDATE recurring_deposits SET next_occurrence = $1, updated_at = $2 WHERE id = $3;`, next, time.Now(), id)
	return err
}

func (r *recurringDepositRepository) Deactivate(id string) error {
	return r.SetActive(id, false)
}
