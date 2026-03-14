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

type SpendingReport struct {
	ByTag           []*TagExpenseSummary
	DailySpending   []*DailySpending
	MonthlySpending []*MonthlySpending
	TopExpenses     []*ExpenseWithTagsAndMethod
	TotalIncome     decimal.Decimal
	TotalExpenses   decimal.Decimal
	NetBalance      decimal.Decimal
}
