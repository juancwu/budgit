package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID        string          `db:"id"`
	Name      string          `db:"name"`
	SpaceID   string          `db:"space_id"`
	Balance   decimal.Decimal `db:"balance"`
	Currency  string          `db:"currency"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
)

type Transaction struct {
	ID          string          `db:"id"`
	Value       decimal.Decimal `db:"value"`
	Type        TransactionType `db:"type"`
	AccountID   string          `db:"account_id"`
	Title       string          `db:"title"`
	Description *string         `db:"description"`
	OccurredAt  time.Time       `db:"occurred_at"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

type Tag struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	SpaceID   string    `db:"space_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Allocation struct {
	ID           string           `db:"id"`
	AccountID    string           `db:"account_id"`
	Name         string           `db:"name"`
	Amount       decimal.Decimal  `db:"amount"`
	TargetAmount *decimal.Decimal `db:"target_amount"`
	SortOrder    int              `db:"sort_order"`
	CreatedAt    time.Time        `db:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at"`
}

type RecurringEventKind string

const (
	RecurringEventKindBill RecurringEventKind = "bill"
	RecurringEventKindFund RecurringEventKind = "fund"
)

type RecurringFrequency string

const (
	RecurringFrequencyDaily   RecurringFrequency = "daily"
	RecurringFrequencyWeekly  RecurringFrequency = "weekly"
	RecurringFrequencyMonthly RecurringFrequency = "monthly"
	RecurringFrequencyYearly  RecurringFrequency = "yearly"
)

type RecurringEvent struct {
	ID              string             `db:"id"`
	SpaceID         string             `db:"space_id"`
	Kind            RecurringEventKind `db:"kind"`
	SourceAccountID string             `db:"source_account_id"`
	Title           string             `db:"title"`
	Amount          decimal.Decimal    `db:"amount"`
	Description     *string            `db:"description"`

	Frequency     RecurringFrequency `db:"frequency"`
	IntervalCount int                `db:"interval_count"`
	DayOfWeek     *int               `db:"day_of_week"`
	DayOfMonth    *int               `db:"day_of_month"`
	MonthOfYear   *int               `db:"month_of_year"`
	FireHour      int                `db:"fire_hour"`
	FireMinute    int                `db:"fire_minute"`
	Timezone      string             `db:"timezone"`

	NextRunAt time.Time  `db:"next_run_at"`
	LastRunAt *time.Time `db:"last_run_at"`
	Paused    bool       `db:"paused"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Category struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
