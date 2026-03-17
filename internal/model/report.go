package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type DailySpending struct {
	Date  time.Time       `db:"date"`
	Total decimal.Decimal `db:"total"`
}

type MonthlySpending struct {
	Month string          `db:"month"`
	Total decimal.Decimal `db:"total"`
}

type PaymentMethodExpenseSummary struct {
	PaymentMethodID   string          `db:"payment_method_id"`
	PaymentMethodName string          `db:"payment_method_name"`
	PaymentMethodType string          `db:"payment_method_type"`
	TotalAmount       decimal.Decimal `db:"total_amount"`
}

type SpendingReport struct {
	ByTag           []*TagExpenseSummary
	ByPaymentMethod []*PaymentMethodExpenseSummary
	DailySpending   []*DailySpending
	MonthlySpending []*MonthlySpending
	TopExpenses     []*ExpenseWithTagsAndMethod
	TotalIncome     decimal.Decimal
	TotalExpenses   decimal.Decimal
	NetBalance      decimal.Decimal
}
