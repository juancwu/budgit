package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID                string          `db:"id"`
	Name              string          `db:"name"`
	SpaceID           string          `db:"space_id"`
	Balance           decimal.Decimal `db:"balance"`
	Currency          string          `db:"currency"`
	IsInvestment      bool            `db:"is_investment"`
	InvestmentSubtype *string         `db:"investment_subtype"`
	CreatedAt         time.Time       `db:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"`
}

type InvestmentSubtype string

const (
	InvestmentSubtypeTFSA     InvestmentSubtype = "tfsa"
	InvestmentSubtypeRRSP     InvestmentSubtype = "rrsp"
	InvestmentSubtypeFHSA     InvestmentSubtype = "fhsa"
	InvestmentSubtypePersonal InvestmentSubtype = "personal"
	InvestmentSubtypeOther    InvestmentSubtype = "other"
)

func IsValidInvestmentSubtype(s string) bool {
	switch InvestmentSubtype(s) {
	case InvestmentSubtypeTFSA, InvestmentSubtypeRRSP, InvestmentSubtypeFHSA,
		InvestmentSubtypePersonal, InvestmentSubtypeOther:
		return true
	}
	return false
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

// TransactionFilter describes optional criteria for narrowing a transaction
// listing. All fields are optional and combine with AND — an empty filter
// matches everything. Amounts are compared against the stored magnitude
// (value is unsigned; direction is derived from Type).
type TransactionFilter struct {
	// Title matches transactions whose title contains this text (case-insensitive).
	Title string
	// DateFrom / DateTo bound occurred_at (inclusive).
	DateFrom *time.Time
	DateTo   *time.Time
	// AmountMin / AmountMax bound value (inclusive). For an exact amount set both
	// to the same value.
	AmountMin *decimal.Decimal
	AmountMax *decimal.Decimal
}

// IsZero reports whether the filter has no active criteria.
func (f TransactionFilter) IsZero() bool {
	return f.Title == "" &&
		f.DateFrom == nil &&
		f.DateTo == nil &&
		f.AmountMin == nil &&
		f.AmountMax == nil
}

// CategoryTimeSeries is a bucketed breakdown of transaction totals by category
// over a time axis, suitable for a stacked time-series chart. Series are ordered
// largest-total first.
type CategoryTimeSeries struct {
	// Buckets are the x-axis time buckets, ascending, one per label. Each is the
	// start of its period (day/month/year).
	Buckets []time.Time
	// Series is one entry per category that has data in the range, plus an
	// "Uncategorized" entry when requested and non-empty.
	Series []CategorySeriesData
	// Total is the grand total across every bucket and series.
	Total decimal.Decimal
}

// CategorySeriesData is a single category's values aligned to
// CategoryTimeSeries.Buckets (same length, zero-filled for empty buckets).
type CategorySeriesData struct {
	CategoryID   string // "" for the uncategorized series
	CategoryName string
	Values       []decimal.Decimal
	Total        decimal.Decimal
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

	BusinessDaysOnly bool `db:"business_days_only"`

	NextRunAt time.Time  `db:"next_run_at"`
	LastRunAt *time.Time `db:"last_run_at"`
	Paused    bool       `db:"paused"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Category struct {
	ID          string    `db:"id"`
	AccountID   string    `db:"account_id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
