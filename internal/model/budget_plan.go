package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// PlanLineKind distinguishes planned income from planned expenses within a
// budget plan.
type PlanLineKind string

const (
	PlanLineKindIncome  PlanLineKind = "income"
	PlanLineKindExpense PlanLineKind = "expense"
)

func IsValidPlanLineKind(k string) bool {
	switch PlanLineKind(k) {
	case PlanLineKindIncome, PlanLineKindExpense:
		return true
	}
	return false
}

// BudgetPlan is a standalone, one-time budgeting sheet scoped to a space. It is
// purely for planning and is intentionally decoupled from accounts and
// transactions.
type BudgetPlan struct {
	ID        string    `db:"id"`
	SpaceID   string    `db:"space_id"`
	Name      string    `db:"name"`
	Note      *string   `db:"note"`
	Currency  string    `db:"currency"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// BudgetPlanLine is a single planned income or expense entry within a plan.
type BudgetPlanLine struct {
	ID        string          `db:"id"`
	PlanID    string          `db:"plan_id"`
	Kind      PlanLineKind    `db:"kind"`
	Label     string          `db:"label"`
	Amount    decimal.Decimal `db:"amount"`
	SortOrder int             `db:"sort_order"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

// PlanSummary is the fully derived view of a budget plan: its lines split into
// income and expenses, plus rolled-up totals. Everything here is computed at
// read time.
type PlanSummary struct {
	Plan         *BudgetPlan
	IncomeLines  []*BudgetPlanLine
	ExpenseLines []*BudgetPlanLine
	TotalIncome  decimal.Decimal
	TotalExpense decimal.Decimal
	Surplus      decimal.Decimal

	TopExpenses []*BudgetPlanLine // largest individual lines first
}
