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
// CategoryID is only meaningful for expense lines; a nil value renders as
// "Uncategorized".
type BudgetPlanLine struct {
	ID         string          `db:"id"`
	PlanID     string          `db:"plan_id"`
	Kind       PlanLineKind    `db:"kind"`
	CategoryID *string         `db:"category_id"`
	Label      string          `db:"label"`
	Amount     decimal.Decimal `db:"amount"`
	SortOrder  int             `db:"sort_order"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

// ExpenseGroup is a set of expense lines that share a category, with their
// rolled-up subtotal. Used to render expenses grouped on the plan editor.
type ExpenseGroup struct {
	CategoryID   *string
	CategoryName string
	Lines        []*BudgetPlanLine
	Subtotal     decimal.Decimal
}

// CategoryTotal is the planned-expense total for a single category, plus its
// share of total expenses. Feeds the per-category bar visualization.
type CategoryTotal struct {
	CategoryID   *string
	CategoryName string
	Total        decimal.Decimal
	Percent      decimal.Decimal // 0–100, share of total expenses
}

// PlanSummary is the fully derived view of a budget plan: its lines split into
// income and grouped expenses, plus rolled-up totals and the aggregates the
// visualizations consume. Everything here is computed at read time.
type PlanSummary struct {
	Plan          *BudgetPlan
	IncomeLines   []*BudgetPlanLine
	ExpenseGroups []ExpenseGroup
	TotalIncome   decimal.Decimal
	TotalExpense  decimal.Decimal
	Surplus       decimal.Decimal

	CategoryTotals []CategoryTotal   // largest first, for per-category bars
	TopExpenses    []*BudgetPlanLine // largest individual lines first
}
