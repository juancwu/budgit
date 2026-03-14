package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type FundingSourceType string

const (
	FundingSourceBalance FundingSourceType = "balance"
	FundingSourceAccount FundingSourceType = "account"
)

type Receipt struct {
	ID                 string          `db:"id"`
	LoanID             string          `db:"loan_id"`
	SpaceID            string          `db:"space_id"`
	Description        string          `db:"description"`
	TotalAmount        decimal.Decimal `db:"total_amount"`
	TotalAmountCents   int             `db:"total_amount_cents"` // deprecated: kept for SELECT * compatibility
	Date               time.Time       `db:"date"`
	RecurringReceiptID *string         `db:"recurring_receipt_id"`
	CreatedBy          string          `db:"created_by"`
	CreatedAt          time.Time       `db:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at"`
}

type ReceiptFundingSource struct {
	ID               string            `db:"id"`
	ReceiptID        string            `db:"receipt_id"`
	SourceType       FundingSourceType `db:"source_type"`
	AccountID        *string           `db:"account_id"`
	Amount           decimal.Decimal   `db:"amount"`
	AmountCents      int               `db:"amount_cents"` // deprecated: kept for SELECT * compatibility
	LinkedExpenseID  *string           `db:"linked_expense_id"`
	LinkedTransferID *string           `db:"linked_transfer_id"`
}

type ReceiptWithSources struct {
	Receipt
	Sources []ReceiptFundingSource
}

type ReceiptFundingSourceWithAccount struct {
	ReceiptFundingSource
	AccountName string
}

type ReceiptWithSourcesAndAccounts struct {
	Receipt
	Sources []ReceiptFundingSourceWithAccount
}
