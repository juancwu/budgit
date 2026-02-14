package model

import "time"

type DailySpending struct {
	Date       time.Time `db:"date"`
	TotalCents int       `db:"total_cents"`
}

type MonthlySpending struct {
	Month      string `db:"month"`
	TotalCents int    `db:"total_cents"`
}

type SpendingReport struct {
	ByTag           []*TagExpenseSummary
	DailySpending   []*DailySpending
	MonthlySpending []*MonthlySpending
	TopExpenses     []*ExpenseWithTagsAndMethod
	TotalIncome     int
	TotalExpenses   int
	NetBalance      int
}
