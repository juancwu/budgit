package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type RecurringReceipt struct {
	ID               string          `db:"id"`
	LoanID           string          `db:"loan_id"`
	SpaceID          string          `db:"space_id"`
	Description      string          `db:"description"`
	TotalAmount      decimal.Decimal `db:"total_amount"`
	TotalAmountCents int             `db:"total_amount_cents"` // deprecated: kept for SELECT * compatibility
	Frequency        Frequency       `db:"frequency"`
	StartDate        time.Time       `db:"start_date"`
	EndDate          *time.Time      `db:"end_date"`
	NextOccurrence   time.Time       `db:"next_occurrence"`
	IsActive         bool            `db:"is_active"`
	CreatedBy        string          `db:"created_by"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

type RecurringReceiptSource struct {
	ID                 string            `db:"id"`
	RecurringReceiptID string            `db:"recurring_receipt_id"`
	SourceType         FundingSourceType `db:"source_type"`
	AccountID          *string           `db:"account_id"`
	Amount             decimal.Decimal   `db:"amount"`
	AmountCents        int               `db:"amount_cents"` // deprecated: kept for SELECT * compatibility
}

type RecurringReceiptWithSources struct {
	RecurringReceipt
	Sources []RecurringReceiptSource
}

type RecurringReceiptWithLoan struct {
	RecurringReceipt
	LoanName string
	Sources  []RecurringReceiptSource
}
