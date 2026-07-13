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
	planRepo     repository.BudgetPlanRepository
	lineRepo     repository.BudgetPlanLineRepository
	categoryRepo repository.CategoryRepository
}

func NewBudgetPlanService(
	planRepo repository.BudgetPlanRepository,
	lineRepo repository.BudgetPlanLineRepository,
	categoryRepo repository.CategoryRepository,
) *BudgetPlanService {
	return &BudgetPlanService{
		planRepo:     planRepo,
		lineRepo:     lineRepo,
		categoryRepo: categoryRepo,
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

func (s *BudgetPlanService) CategoriesForSpace(spaceID string) ([]*model.Category, error) {
	return s.categoryRepo.ListBySpace(spaceID)
}

// ---------- Lines ----------

type AddPlanLineInput struct {
	PlanID     string
	Kind       model.PlanLineKind
	CategoryID *string
	Label      string
	Amount     decimal.Decimal
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
	plan, err := s.planRepo.ByID(in.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}
	categoryID, err := s.resolveCategory(in.Kind, in.CategoryID, plan.SpaceID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	line := &model.BudgetPlanLine{
		ID:         uuid.NewString(),
		PlanID:     in.PlanID,
		Kind:       in.Kind,
		CategoryID: categoryID,
		Label:      label,
		Amount:     in.Amount,
		SortOrder:  0,
		CreatedAt:  now,
		UpdatedAt:  now,
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
// kind is fixed; only its label, amount, and (for expenses) category change.
func (s *BudgetPlanService) UpdateLine(line *model.BudgetPlanLine, label string, amount decimal.Decimal, categoryID *string) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return fmt.Errorf("label is required")
	}
	if err := validatePlanAmount(amount); err != nil {
		return err
	}
	plan, err := s.planRepo.ByID(line.PlanID)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}
	resolved, err := s.resolveCategory(line.Kind, categoryID, plan.SpaceID)
	if err != nil {
		return err
	}
	return s.lineRepo.Update(line.ID, label, amount, resolved)
}

func (s *BudgetPlanService) DeleteLine(id string) error {
	return s.lineRepo.Delete(id)
}

// ---------- Summary ----------

// Summarize builds the derived view of a plan: income lines, expenses grouped
// by category (largest first), totals, surplus, and the aggregates used by the
// visualizations.
func (s *BudgetPlanService) Summarize(planID string) (*model.PlanSummary, error) {
	plan, err := s.planRepo.ByID(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}
	lines, err := s.lineRepo.ByPlanID(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan lines: %w", err)
	}
	names, err := s.categoryNames(plan.SpaceID)
	if err != nil {
		return nil, err
	}

	summary := &model.PlanSummary{Plan: plan}
	groupIdx := map[string]int{}
	var expenseLines []*model.BudgetPlanLine

	for _, l := range lines {
		switch l.Kind {
		case model.PlanLineKindIncome:
			summary.IncomeLines = append(summary.IncomeLines, l)
			summary.TotalIncome = summary.TotalIncome.Add(l.Amount)
		case model.PlanLineKindExpense:
			summary.TotalExpense = summary.TotalExpense.Add(l.Amount)
			expenseLines = append(expenseLines, l)

			key := "uncategorized"
			name := "Uncategorized"
			if l.CategoryID != nil {
				key = *l.CategoryID
				if n, ok := names[*l.CategoryID]; ok {
					name = n
				}
			}
			idx, ok := groupIdx[key]
			if !ok {
				summary.ExpenseGroups = append(summary.ExpenseGroups, model.ExpenseGroup{
					CategoryID:   l.CategoryID,
					CategoryName: name,
				})
				idx = len(summary.ExpenseGroups) - 1
				groupIdx[key] = idx
			}
			g := &summary.ExpenseGroups[idx]
			g.Lines = append(g.Lines, l)
			g.Subtotal = g.Subtotal.Add(l.Amount)
		}
	}
	summary.Surplus = summary.TotalIncome.Sub(summary.TotalExpense)

	sort.SliceStable(summary.ExpenseGroups, func(i, j int) bool {
		return summary.ExpenseGroups[i].Subtotal.GreaterThan(summary.ExpenseGroups[j].Subtotal)
	})

	hundred := decimal.NewFromInt(100)
	for _, g := range summary.ExpenseGroups {
		percent := decimal.Zero
		if summary.TotalExpense.IsPositive() {
			percent = g.Subtotal.Div(summary.TotalExpense).Mul(hundred)
		}
		summary.CategoryTotals = append(summary.CategoryTotals, model.CategoryTotal{
			CategoryID:   g.CategoryID,
			CategoryName: g.CategoryName,
			Total:        g.Subtotal,
			Percent:      percent,
		})
	}

	sorted := make([]*model.BudgetPlanLine, len(expenseLines))
	copy(sorted, expenseLines)
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

// resolveCategory enforces that income lines carry no category and that an
// expense's category, if any, refers to a real category owned by the plan's
// space.
func (s *BudgetPlanService) resolveCategory(kind model.PlanLineKind, categoryID *string, spaceID string) (*string, error) {
	if kind == model.PlanLineKindIncome {
		return nil, nil
	}
	if categoryID == nil {
		return nil, nil
	}
	if err := s.ensureCategory(spaceID, *categoryID); err != nil {
		return nil, err
	}
	return categoryID, nil
}

func (s *BudgetPlanService) ensureCategory(spaceID, id string) error {
	cat, err := s.categoryRepo.ByID(id)
	if err != nil {
		return fmt.Errorf("failed to load category: %w", err)
	}
	if cat == nil || cat.SpaceID != spaceID {
		return fmt.Errorf("unknown category")
	}
	return nil
}

func (s *BudgetPlanService) categoryNames(spaceID string) (map[string]string, error) {
	cats, err := s.categoryRepo.ListBySpace(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}
	m := make(map[string]string, len(cats))
	for _, c := range cats {
		m[c.ID] = c.Name
	}
	return m, nil
}

func validatePlanAmount(amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be greater than zero")
	}
	if amount.Exponent() < -2 {
		return fmt.Errorf("amount can have at most 2 decimal places")
	}
	return nil
}
