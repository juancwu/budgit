package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type RecurringDeposit struct {
	ID             string          `db:"id"`
	SpaceID        string          `db:"space_id"`
	AccountID      string          `db:"account_id"`
	Amount         decimal.Decimal `db:"amount"`
	AmountCents    int             `db:"amount_cents"` // deprecated: kept for SELECT * compatibility
	Frequency      Frequency       `db:"frequency"`
	StartDate      time.Time       `db:"start_date"`
	EndDate        *time.Time      `db:"end_date"`
	NextOccurrence time.Time       `db:"next_occurrence"`
	IsActive       bool            `db:"is_active"`
	Title          string          `db:"title"`
	CreatedBy      string          `db:"created_by"`
	CreatedAt      time.Time       `db:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"`
}

type RecurringDepositWithAccount struct {
	RecurringDeposit
	AccountName string
}
