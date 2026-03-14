package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Loan struct {
	ID                  string          `db:"id"`
	SpaceID             string          `db:"space_id"`
	Name                string          `db:"name"`
	Description         string          `db:"description"`
	OriginalAmount      decimal.Decimal `db:"original_amount"`
	OriginalAmountCents int             `db:"original_amount_cents"` // deprecated: kept for SELECT * compatibility
	InterestRateBps     int             `db:"interest_rate_bps"`
	StartDate           time.Time       `db:"start_date"`
	EndDate             *time.Time      `db:"end_date"`
	IsPaidOff           bool            `db:"is_paid_off"`
	CreatedBy           string          `db:"created_by"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}

type LoanWithPaymentSummary struct {
	Loan
	TotalPaid    decimal.Decimal
	Remaining    decimal.Decimal
	ReceiptCount int
}
