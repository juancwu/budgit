package model

import "time"

type Frequency string

const (
	FrequencyDaily    Frequency = "daily"
	FrequencyWeekly   Frequency = "weekly"
	FrequencyBiweekly Frequency = "biweekly"
	FrequencyMonthly  Frequency = "monthly"
	FrequencyYearly   Frequency = "yearly"
)

type RecurringExpense struct {
	ID              string      `db:"id"`
	SpaceID         string      `db:"space_id"`
	CreatedBy       string      `db:"created_by"`
	Description     string      `db:"description"`
	AmountCents     int         `db:"amount_cents"`
	Type            ExpenseType `db:"type"`
	PaymentMethodID *string     `db:"payment_method_id"`
	Frequency       Frequency   `db:"frequency"`
	StartDate       time.Time   `db:"start_date"`
	EndDate         *time.Time  `db:"end_date"`
	NextOccurrence  time.Time   `db:"next_occurrence"`
	IsActive        bool        `db:"is_active"`
	CreatedAt       time.Time   `db:"created_at"`
	UpdatedAt       time.Time   `db:"updated_at"`
}

type RecurringExpenseWithTags struct {
	RecurringExpense
	Tags []*Tag
}

type RecurringExpenseWithTagsAndMethod struct {
	RecurringExpense
	Tags          []*Tag
	PaymentMethod *PaymentMethod
}
