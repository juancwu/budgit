package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrRecurringReceiptNotFound = errors.New("recurring receipt not found")
)

type RecurringReceiptRepository interface {
	Create(rr *model.RecurringReceipt, sources []model.RecurringReceiptSource) error
	GetByID(id string) (*model.RecurringReceipt, error)
	GetByLoanID(loanID string) ([]*model.RecurringReceipt, error)
	GetBySpaceID(spaceID string) ([]*model.RecurringReceipt, error)
	GetSourcesByRecurringReceiptID(id string) ([]model.RecurringReceiptSource, error)
	Update(rr *model.RecurringReceipt, sources []model.RecurringReceiptSource) error
	Delete(id string) error
	SetActive(id string, active bool) error
	Deactivate(id string) error
	GetDueRecurrences(now time.Time) ([]*model.RecurringReceipt, error)
	GetDueRecurrencesForSpace(spaceID string, now time.Time) ([]*model.RecurringReceipt, error)
	UpdateNextOccurrence(id string, next time.Time) error
}

type recurringReceiptRepository struct {
	db *sqlx.DB
}

func NewRecurringReceiptRepository(db *sqlx.DB) RecurringReceiptRepository {
	return &recurringReceiptRepository{db: db}
}

func (r *recurringReceiptRepository) Create(rr *model.RecurringReceipt, sources []model.RecurringReceiptSource) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO recurring_receipts (id, loan_id, space_id, description, total_amount_cents, frequency, start_date, end_date, next_occurrence, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);`,
		rr.ID, rr.LoanID, rr.SpaceID, rr.Description, rr.TotalAmountCents, rr.Frequency, rr.StartDate, rr.EndDate, rr.NextOccurrence, rr.IsActive, rr.CreatedBy, rr.CreatedAt, rr.UpdatedAt,
	)
	if err != nil {
		return err
	}

	for _, src := range sources {
		_, err = tx.Exec(
			`INSERT INTO recurring_receipt_sources (id, recurring_receipt_id, source_type, account_id, amount_cents)
			VALUES ($1, $2, $3, $4, $5);`,
			src.ID, src.RecurringReceiptID, src.SourceType, src.AccountID, src.AmountCents,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *recurringReceiptRepository) GetByID(id string) (*model.RecurringReceipt, error) {
	rr := &model.RecurringReceipt{}
	query := `SELECT * FROM recurring_receipts WHERE id = $1;`
	err := r.db.Get(rr, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrRecurringReceiptNotFound
	}
	return rr, err
}

func (r *recurringReceiptRepository) GetByLoanID(loanID string) ([]*model.RecurringReceipt, error) {
	var results []*model.RecurringReceipt
	query := `SELECT * FROM recurring_receipts WHERE loan_id = $1 ORDER BY is_active DESC, next_occurrence ASC;`
	err := r.db.Select(&results, query, loanID)
	return results, err
}

func (r *recurringReceiptRepository) GetBySpaceID(spaceID string) ([]*model.RecurringReceipt, error) {
	var results []*model.RecurringReceipt
	query := `SELECT * FROM recurring_receipts WHERE space_id = $1 ORDER BY is_active DESC, next_occurrence ASC;`
	err := r.db.Select(&results, query, spaceID)
	return results, err
}

func (r *recurringReceiptRepository) GetSourcesByRecurringReceiptID(id string) ([]model.RecurringReceiptSource, error) {
	var sources []model.RecurringReceiptSource
	query := `SELECT * FROM recurring_receipt_sources WHERE recurring_receipt_id = $1;`
	err := r.db.Select(&sources, query, id)
	return sources, err
}

func (r *recurringReceiptRepository) Update(rr *model.RecurringReceipt, sources []model.RecurringReceiptSource) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE recurring_receipts SET description = $1, total_amount_cents = $2, frequency = $3, start_date = $4, end_date = $5, next_occurrence = $6, updated_at = $7 WHERE id = $8;`,
		rr.Description, rr.TotalAmountCents, rr.Frequency, rr.StartDate, rr.EndDate, rr.NextOccurrence, rr.UpdatedAt, rr.ID,
	)
	if err != nil {
		return err
	}

	// Replace sources
	if _, err := tx.Exec(`DELETE FROM recurring_receipt_sources WHERE recurring_receipt_id = $1;`, rr.ID); err != nil {
		return err
	}

	for _, src := range sources {
		_, err = tx.Exec(
			`INSERT INTO recurring_receipt_sources (id, recurring_receipt_id, source_type, account_id, amount_cents)
			VALUES ($1, $2, $3, $4, $5);`,
			src.ID, src.RecurringReceiptID, src.SourceType, src.AccountID, src.AmountCents,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *recurringReceiptRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM recurring_receipts WHERE id = $1;`, id)
	return err
}

func (r *recurringReceiptRepository) SetActive(id string, active bool) error {
	_, err := r.db.Exec(`UPDATE recurring_receipts SET is_active = $1, updated_at = $2 WHERE id = $3;`, active, time.Now(), id)
	return err
}

func (r *recurringReceiptRepository) Deactivate(id string) error {
	return r.SetActive(id, false)
}

func (r *recurringReceiptRepository) GetDueRecurrences(now time.Time) ([]*model.RecurringReceipt, error) {
	var results []*model.RecurringReceipt
	query := `SELECT * FROM recurring_receipts WHERE is_active = true AND next_occurrence <= $1;`
	err := r.db.Select(&results, query, now)
	return results, err
}

func (r *recurringReceiptRepository) GetDueRecurrencesForSpace(spaceID string, now time.Time) ([]*model.RecurringReceipt, error) {
	var results []*model.RecurringReceipt
	query := `SELECT * FROM recurring_receipts WHERE is_active = true AND space_id = $1 AND next_occurrence <= $2;`
	err := r.db.Select(&results, query, spaceID, now)
	return results, err
}

func (r *recurringReceiptRepository) UpdateNextOccurrence(id string, next time.Time) error {
	_, err := r.db.Exec(`UPDATE recurring_receipts SET next_occurrence = $1, updated_at = $2 WHERE id = $3;`, next, time.Now(), id)
	return err
}
