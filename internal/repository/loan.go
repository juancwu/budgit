package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var (
	ErrLoanNotFound = errors.New("loan not found")
)

type LoanRepository interface {
	Create(loan *model.Loan) error
	GetByID(id string) (*model.Loan, error)
	GetBySpaceID(spaceID string) ([]*model.Loan, error)
	GetBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.Loan, error)
	CountBySpaceID(spaceID string) (int, error)
	Update(loan *model.Loan) error
	Delete(id string) error
	SetPaidOff(id string, paidOff bool) error
	GetTotalPaidForLoan(loanID string) (decimal.Decimal, error)
	GetReceiptCountForLoan(loanID string) (int, error)
}

type loanRepository struct {
	db *sqlx.DB
}

func NewLoanRepository(db *sqlx.DB) LoanRepository {
	return &loanRepository{db: db}
}

func (r *loanRepository) Create(loan *model.Loan) error {
	query := `INSERT INTO loans (id, space_id, name, description, original_amount, interest_rate_bps, start_date, end_date, is_paid_off, created_by, created_at, updated_at, original_amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 0);`
	_, err := r.db.Exec(query, loan.ID, loan.SpaceID, loan.Name, loan.Description, loan.OriginalAmount, loan.InterestRateBps, loan.StartDate, loan.EndDate, loan.IsPaidOff, loan.CreatedBy, loan.CreatedAt, loan.UpdatedAt)
	return err
}

func (r *loanRepository) GetByID(id string) (*model.Loan, error) {
	loan := &model.Loan{}
	query := `SELECT * FROM loans WHERE id = $1;`
	err := r.db.Get(loan, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrLoanNotFound
	}
	return loan, err
}

func (r *loanRepository) GetBySpaceID(spaceID string) ([]*model.Loan, error) {
	var loans []*model.Loan
	query := `SELECT * FROM loans WHERE space_id = $1 ORDER BY is_paid_off ASC, created_at DESC;`
	err := r.db.Select(&loans, query, spaceID)
	return loans, err
}

func (r *loanRepository) GetBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.Loan, error) {
	var loans []*model.Loan
	query := `SELECT * FROM loans WHERE space_id = $1 ORDER BY is_paid_off ASC, created_at DESC LIMIT $2 OFFSET $3;`
	err := r.db.Select(&loans, query, spaceID, limit, offset)
	return loans, err
}

func (r *loanRepository) CountBySpaceID(spaceID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM loans WHERE space_id = $1;`, spaceID)
	return count, err
}

func (r *loanRepository) Update(loan *model.Loan) error {
	query := `UPDATE loans SET name = $1, description = $2, original_amount = $3, interest_rate_bps = $4, start_date = $5, end_date = $6, updated_at = $7 WHERE id = $8;`
	result, err := r.db.Exec(query, loan.Name, loan.Description, loan.OriginalAmount, loan.InterestRateBps, loan.StartDate, loan.EndDate, loan.UpdatedAt, loan.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrLoanNotFound
	}
	return err
}

func (r *loanRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM loans WHERE id = $1;`, id)
	return err
}

func (r *loanRepository) SetPaidOff(id string, paidOff bool) error {
	_, err := r.db.Exec(`UPDATE loans SET is_paid_off = $1, updated_at = $2 WHERE id = $3;`, paidOff, time.Now(), id)
	return err
}

func (r *loanRepository) GetTotalPaidForLoan(loanID string) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.Get(&total, `SELECT COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0) FROM receipts WHERE loan_id = $1;`, loanID)
	return total, err
}

func (r *loanRepository) GetReceiptCountForLoan(loanID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM receipts WHERE loan_id = $1;`, loanID)
	return count, err
}
