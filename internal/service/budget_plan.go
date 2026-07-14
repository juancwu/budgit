package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BudgetPlanService manages standalone budget planning sheets and their income
// and expense lines. Plans are purely for planning and never touch accounts or
// transactions.
type BudgetPlanService struct {
	planRepo repository.BudgetPlanRepository
	lineRepo repository.BudgetPlanLineRepository
}

func NewBudgetPlanService(
	planRepo repository.BudgetPlanRepository,
	lineRepo repository.BudgetPlanLineRepository,
) *BudgetPlanService {
	return &BudgetPlanService{
		planRepo: planRepo,
		lineRepo: lineRepo,
	}
}

// ---------- Plans ----------

func (s *BudgetPlanService) CreatePlan(spaceID, name, note, currency string) (*model.BudgetPlan, error) {
	if spaceID == "" {
		return nil, fmt.Errorf("space id is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Untitled plan"
	}
	currency = strings.TrimSpace(currency)
	if currency == "" {
		currency = "USD"
	}
	var notePtr *string
	if n := strings.TrimSpace(note); n != "" {
		notePtr = &n
	}
	now := time.Now()
	plan := &model.BudgetPlan{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		Name:      name,
		Note:      notePtr,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.planRepo.Create(plan); err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	return plan, nil
}

func (s *BudgetPlanService) GetPlan(id string) (*model.BudgetPlan, error) {
	return s.planRepo.ByID(id)
}

func (s *BudgetPlanService) ListPlans(spaceID string) ([]*model.BudgetPlan, error) {
	return s.planRepo.BySpaceID(spaceID)
}

func (s *BudgetPlanService) RenamePlan(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	return s.planRepo.Rename(id, name)
}

func (s *BudgetPlanService) DeletePlan(id string) error {
	return s.planRepo.Delete(id)
}

// ---------- Lines ----------

type AddPlanLineInput struct {
	PlanID string
	Kind   model.PlanLineKind
	Label  string
	Amount decimal.Decimal
}

func (s *BudgetPlanService) AddLine(in AddPlanLineInput) (*model.BudgetPlanLine, error) {
	if in.PlanID == "" {
		return nil, fmt.Errorf("plan id is required")
	}
	if !model.IsValidPlanLineKind(string(in.Kind)) {
		return nil, fmt.Errorf("invalid line kind")
	}
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return nil, fmt.Errorf("label is required")
	}
	if err := validatePlanAmount(in.Amount); err != nil {
		return nil, err
	}
	now := time.Now()
	line := &model.BudgetPlanLine{
		ID:        uuid.NewString(),
		PlanID:    in.PlanID,
		Kind:      in.Kind,
		Label:     label,
		Amount:    in.Amount,
		SortOrder: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.lineRepo.Create(line); err != nil {
		return nil, fmt.Errorf("failed to create line: %w", err)
	}
	return line, nil
}

func (s *BudgetPlanService) GetLine(id string) (*model.BudgetPlanLine, error) {
	return s.lineRepo.ByID(id)
}

// UpdateLine validates and persists changes to an existing line. The line's
// kind is fixed; only its label and amount change.
func (s *BudgetPlanService) UpdateLine(line *model.BudgetPlanLine, label string, amount decimal.Decimal) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return fmt.Errorf("label is required")
	}
	if err := validatePlanAmount(amount); err != nil {
		return err
	}
	return s.lineRepo.Update(line.ID, label, amount)
}

func (s *BudgetPlanService) DeleteLine(id string) error {
	return s.lineRepo.Delete(id)
}

// ---------- Summary ----------

// Summarize builds the derived view of a plan: income and expense lines,
// rolled-up totals, surplus, and the largest individual expenses.
func (s *BudgetPlanService) Summarize(planID string) (*model.PlanSummary, error) {
	plan, err := s.planRepo.ByID(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}
	lines, err := s.lineRepo.ByPlanID(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan lines: %w", err)
	}

	summary := &model.PlanSummary{Plan: plan}
	for _, l := range lines {
		switch l.Kind {
		case model.PlanLineKindIncome:
			summary.IncomeLines = append(summary.IncomeLines, l)
			summary.TotalIncome = summary.TotalIncome.Add(l.Amount)
		case model.PlanLineKindExpense:
			summary.ExpenseLines = append(summary.ExpenseLines, l)
			summary.TotalExpense = summary.TotalExpense.Add(l.Amount)
		}
	}
	summary.Surplus = summary.TotalIncome.Sub(summary.TotalExpense)

	sorted := make([]*model.BudgetPlanLine, len(summary.ExpenseLines))
	copy(sorted, summary.ExpenseLines)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Amount.GreaterThan(sorted[j].Amount)
	})
	if len(sorted) > 5 {
		sorted = sorted[:5]
	}
	summary.TopExpenses = sorted

	return summary, nil
}

// ---------- helpers ----------

func validatePlanAmount(amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be greater than zero")
	}
	if amount.Exponent() < -2 {
		return fmt.Errorf("amount can have at most 2 decimal places")
	}
	return nil
}
