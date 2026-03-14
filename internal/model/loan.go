package model

import "time"

type Loan struct {
	ID                  string     `db:"id"`
	SpaceID             string     `db:"space_id"`
	Name                string     `db:"name"`
	Description         string     `db:"description"`
	OriginalAmountCents int        `db:"original_amount_cents"`
	InterestRateBps     int        `db:"interest_rate_bps"`
	StartDate           time.Time  `db:"start_date"`
	EndDate             *time.Time `db:"end_date"`
	IsPaidOff           bool       `db:"is_paid_off"`
	CreatedBy           string     `db:"created_by"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

type LoanWithPaymentSummary struct {
	Loan
	TotalPaidCents int
	RemainingCents int
	ReceiptCount   int
}
