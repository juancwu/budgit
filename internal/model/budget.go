package model

import "time"

type BudgetPeriod string

const (
	BudgetPeriodWeekly  BudgetPeriod = "weekly"
	BudgetPeriodMonthly BudgetPeriod = "monthly"
	BudgetPeriodYearly  BudgetPeriod = "yearly"
)

type BudgetStatus string

const (
	BudgetStatusOnTrack BudgetStatus = "on_track"
	BudgetStatusWarning BudgetStatus = "warning"
	BudgetStatusOver    BudgetStatus = "over"
)

type Budget struct {
	ID          string       `db:"id"`
	SpaceID     string       `db:"space_id"`
	TagID       string       `db:"tag_id"`
	AmountCents int          `db:"amount_cents"`
	Period      BudgetPeriod `db:"period"`
	StartDate   time.Time    `db:"start_date"`
	EndDate     *time.Time   `db:"end_date"`
	IsActive    bool         `db:"is_active"`
	CreatedBy   string       `db:"created_by"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"`
}

type BudgetWithSpent struct {
	Budget
	TagName    string  `db:"tag_name"`
	TagColor   *string `db:"tag_color"`
	SpentCents int
	Percentage float64
	Status     BudgetStatus
}
