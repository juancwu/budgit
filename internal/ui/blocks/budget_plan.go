package blocks

import (
	"strconv"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/shopspring/decimal"
)

// LineFormState echoes a submitted add/edit line form so the board can be
// re-rendered with the user's input and an error message after a failed submit.
type LineFormState struct {
	Label      string
	Amount     string
	CategoryID string
	Err        string
}

// BudgetPlanBoardProps drives the #plan-board fragment: the live summary, the
// income and expense lists, and the inline add/edit forms.
type BudgetPlanBoardProps struct {
	SpaceID    string
	PlanID     string
	Currency   string
	Summary    *model.PlanSummary
	Categories []*model.Category

	// Add-line form echo state. Set after a failed add so the form re-opens
	// with the user's values and an error.
	IncomeForm      LineFormState
	ShowIncomeForm  bool
	ExpenseForm     LineFormState
	ShowExpenseForm bool

	// Edit-line echo state. When EditLineID matches a line, that row renders
	// its edit form open with EditForm's values and error.
	EditLineID string
	EditForm   LineFormState
}

// barWidthStyle returns an inline width style scaling value against max,
// clamped to 0–100%. Used for the CSS bar visualizations.
func barWidthStyle(value, max decimal.Decimal) string {
	if !max.IsPositive() {
		return "width: 0%;"
	}
	pct := value.Div(max).Mul(decimal.NewFromInt(100))
	if pct.GreaterThan(decimal.NewFromInt(100)) {
		pct = decimal.NewFromInt(100)
	}
	if pct.IsNegative() {
		pct = decimal.Zero
	}
	return "width: " + pct.Round(2).String() + "%;"
}

func percentLabel(p decimal.Decimal) string {
	return p.Round(1).String() + "%"
}

func rankLabel(i int) string {
	return strconv.Itoa(i+1) + "."
}
