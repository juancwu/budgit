package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var (
	ErrReceiptNotFound = errors.New("receipt not found")
)

type ReceiptRepository interface {
	CreateWithSources(
		receipt *model.Receipt,
		sources []model.ReceiptFundingSource,
		balanceExpense *model.Expense,
		accountTransfers []*model.AccountTransfer,
	) error
	GetByID(id string) (*model.Receipt, error)
	GetByLoanIDPaginated(loanID string, limit, offset int) ([]*model.Receipt, error)
	CountByLoanID(loanID string) (int, error)
	GetBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.Receipt, error)
	CountBySpaceID(spaceID string) (int, error)
	GetFundingSourcesByReceiptID(receiptID string) ([]model.ReceiptFundingSource, error)
	GetFundingSourcesWithAccountsByReceiptIDs(receiptIDs []string) (map[string][]model.ReceiptFundingSourceWithAccount, error)
	DeleteWithReversal(receiptID string) error
	UpdateWithSources(
		receipt *model.Receipt,
		sources []model.ReceiptFundingSource,
		balanceExpense *model.Expense,
		accountTransfers []*model.AccountTransfer,
	) error
}

type receiptRepository struct {
	db *sqlx.DB
}

func NewReceiptRepository(db *sqlx.DB) ReceiptRepository {
	return &receiptRepository{db: db}
}

func (r *receiptRepository) CreateWithSources(
	receipt *model.Receipt,
	sources []model.ReceiptFundingSource,
	balanceExpense *model.Expense,
	accountTransfers []*model.AccountTransfer,
) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert receipt
	_, err = tx.Exec(
		`INSERT INTO receipts (id, loan_id, space_id, description, total_amount, date, recurring_receipt_id, created_by, created_at, updated_at, total_amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0);`,
		receipt.ID, receipt.LoanID, receipt.SpaceID, receipt.Description, receipt.TotalAmount, receipt.Date, receipt.RecurringReceiptID, receipt.CreatedBy, receipt.CreatedAt, receipt.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Insert balance expense if present
	if balanceExpense != nil {
		_, err = tx.Exec(
			`INSERT INTO expenses (id, space_id, created_by, description, amount, type, date, payment_method_id, recurring_expense_id, created_at, updated_at, amount_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 0);`,
			balanceExpense.ID, balanceExpense.SpaceID, balanceExpense.CreatedBy, balanceExpense.Description, balanceExpense.Amount, balanceExpense.Type, balanceExpense.Date, balanceExpense.PaymentMethodID, balanceExpense.RecurringExpenseID, balanceExpense.CreatedAt, balanceExpense.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	// Insert account transfers
	for _, transfer := range accountTransfers {
		_, err = tx.Exec(
			`INSERT INTO account_transfers (id, account_id, amount, direction, note, recurring_deposit_id, created_by, created_at, amount_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0);`,
			transfer.ID, transfer.AccountID, transfer.Amount, transfer.Direction, transfer.Note, transfer.RecurringDepositID, transfer.CreatedBy, transfer.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	// Insert funding sources
	for _, src := range sources {
		_, err = tx.Exec(
			`INSERT INTO receipt_funding_sources (id, receipt_id, source_type, account_id, amount, linked_expense_id, linked_transfer_id, amount_cents)
			Values ($1, $2, $3, $4, $5, $6, $7, 0);`,
			src.ID, src.ReceiptID, src.SourceType, src.AccountID, src.Amount, src.LinkedExpenseID, src.LinkedTransferID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *receiptRepository) GetByID(id string) (*model.Receipt, error) {
	receipt := &model.Receipt{}
	query := `SELECT * FROM receipts WHERE id = $1;`
	err := r.db.Get(receipt, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrReceiptNotFound
	}
	return receipt, err
}

func (r *receiptRepository) GetByLoanIDPaginated(loanID string, limit, offset int) ([]*model.Receipt, error) {
	var receipts []*model.Receipt
	query := `SELECT * FROM receipts WHERE loan_id = $1 ORDER BY date DESC, created_at DESC LIMIT $2 OFFSET $3;`
	err := r.db.Select(&receipts, query, loanID, limit, offset)
	return receipts, err
}

func (r *receiptRepository) CountByLoanID(loanID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM receipts WHERE loan_id = $1;`, loanID)
	return count, err
}

func (r *receiptRepository) GetBySpaceIDPaginated(spaceID string, limit, offset int) ([]*model.Receipt, error) {
	var receipts []*model.Receipt
	query := `SELECT * FROM receipts WHERE space_id = $1 ORDER BY date DESC, created_at DESC LIMIT $2 OFFSET $3;`
	err := r.db.Select(&receipts, query, spaceID, limit, offset)
	return receipts, err
}

func (r *receiptRepository) CountBySpaceID(spaceID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM receipts WHERE space_id = $1;`, spaceID)
	return count, err
}

func (r *receiptRepository) GetFundingSourcesByReceiptID(receiptID string) ([]model.ReceiptFundingSource, error) {
	var sources []model.ReceiptFundingSource
	query := `SELECT * FROM receipt_funding_sources WHERE receipt_id = $1;`
	err := r.db.Select(&sources, query, receiptID)
	return sources, err
}

func (r *receiptRepository) GetFundingSourcesWithAccountsByReceiptIDs(receiptIDs []string) (map[string][]model.ReceiptFundingSourceWithAccount, error) {
	if len(receiptIDs) == 0 {
		return make(map[string][]model.ReceiptFundingSourceWithAccount), nil
	}

	type row struct {
		ID               string                  `db:"id"`
		ReceiptID        string                  `db:"receipt_id"`
		SourceType       model.FundingSourceType `db:"source_type"`
		AccountID        *string                 `db:"account_id"`
		Amount           decimal.Decimal         `db:"amount"`
		LinkedExpenseID  *string                 `db:"linked_expense_id"`
		LinkedTransferID *string                 `db:"linked_transfer_id"`
		AccountName      *string                 `db:"account_name"`
	}

	query, args, err := sqlx.In(`
		SELECT rfs.id, rfs.receipt_id, rfs.source_type, rfs.account_id, rfs.amount,
			rfs.linked_expense_id, rfs.linked_transfer_id,
			ma.name AS account_name
		FROM receipt_funding_sources rfs
		LEFT JOIN money_accounts ma ON rfs.account_id = ma.id
		WHERE rfs.receipt_id IN (?)
		ORDER BY rfs.source_type ASC;
	`, receiptIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var rows []row
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}

	result := make(map[string][]model.ReceiptFundingSourceWithAccount)
	for _, rw := range rows {
		accountName := ""
		if rw.AccountName != nil {
			accountName = *rw.AccountName
		}
		result[rw.ReceiptID] = append(result[rw.ReceiptID], model.ReceiptFundingSourceWithAccount{
			ReceiptFundingSource: model.ReceiptFundingSource{
				ID:               rw.ID,
				ReceiptID:        rw.ReceiptID,
				SourceType:       rw.SourceType,
				AccountID:        rw.AccountID,
				Amount:           rw.Amount,
				LinkedExpenseID:  rw.LinkedExpenseID,
				LinkedTransferID: rw.LinkedTransferID,
			},
			AccountName: accountName,
		})
	}
	return result, nil
}

func (r *receiptRepository) DeleteWithReversal(receiptID string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get all funding sources for this receipt
	var sources []model.ReceiptFundingSource
	if err := tx.Select(&sources, `SELECT * FROM receipt_funding_sources WHERE receipt_id = $1;`, receiptID); err != nil {
		return err
	}

	// Delete linked expenses and transfers
	for _, src := range sources {
		if src.LinkedExpenseID != nil {
			if _, err := tx.Exec(`DELETE FROM expenses WHERE id = $1;`, *src.LinkedExpenseID); err != nil {
				return err
			}
		}
		if src.LinkedTransferID != nil {
			if _, err := tx.Exec(`DELETE FROM account_transfers WHERE id = $1;`, *src.LinkedTransferID); err != nil {
				return err
			}
		}
	}

	// Delete funding sources (cascade would handle this, but be explicit)
	if _, err := tx.Exec(`DELETE FROM receipt_funding_sources WHERE receipt_id = $1;`, receiptID); err != nil {
		return err
	}

	// Delete the receipt
	if _, err := tx.Exec(`DELETE FROM receipts WHERE id = $1;`, receiptID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *receiptRepository) UpdateWithSources(
	receipt *model.Receipt,
	sources []model.ReceiptFundingSource,
	balanceExpense *model.Expense,
	accountTransfers []*model.AccountTransfer,
) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete old linked records
	var oldSources []model.ReceiptFundingSource
	if err := tx.Select(&oldSources, `SELECT * FROM receipt_funding_sources WHERE receipt_id = $1;`, receipt.ID); err != nil {
		return err
	}
	for _, src := range oldSources {
		if src.LinkedExpenseID != nil {
			if _, err := tx.Exec(`DELETE FROM expenses WHERE id = $1;`, *src.LinkedExpenseID); err != nil {
				return err
			}
		}
		if src.LinkedTransferID != nil {
			if _, err := tx.Exec(`DELETE FROM account_transfers WHERE id = $1;`, *src.LinkedTransferID); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(`DELETE FROM receipt_funding_sources WHERE receipt_id = $1;`, receipt.ID); err != nil {
		return err
	}

	// Update receipt
	_, err = tx.Exec(
		`UPDATE receipts SET description = $1, total_amount = $2, date = $3, updated_at = $4 WHERE id = $5;`,
		receipt.Description, receipt.TotalAmount, receipt.Date, receipt.UpdatedAt, receipt.ID,
	)
	if err != nil {
		return err
	}

	// Insert new balance expense
	if balanceExpense != nil {
		_, err = tx.Exec(
			`INSERT INTO expenses (id, space_id, created_by, description, amount, type, date, payment_method_id, recurring_expense_id, created_at, updated_at, amount_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 0);`,
			balanceExpense.ID, balanceExpense.SpaceID, balanceExpense.CreatedBy, balanceExpense.Description, balanceExpense.Amount, balanceExpense.Type, balanceExpense.Date, balanceExpense.PaymentMethodID, balanceExpense.RecurringExpenseID, balanceExpense.CreatedAt, balanceExpense.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	// Insert new account transfers
	for _, transfer := range accountTransfers {
		_, err = tx.Exec(
			`INSERT INTO account_transfers (id, account_id, amount, direction, note, recurring_deposit_id, created_by, created_at, amount_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0);`,
			transfer.ID, transfer.AccountID, transfer.Amount, transfer.Direction, transfer.Note, transfer.RecurringDepositID, transfer.CreatedBy, transfer.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	// Insert new funding sources
	for _, src := range sources {
		_, err = tx.Exec(
			`INSERT INTO receipt_funding_sources (id, receipt_id, source_type, account_id, amount, linked_expense_id, linked_transfer_id, amount_cents)
			Values ($1, $2, $3, $4, $5, $6, $7, 0);`,
			src.ID, src.ReceiptID, src.SourceType, src.AccountID, src.Amount, src.LinkedExpenseID, src.LinkedTransferID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
