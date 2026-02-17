package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreateBudgetDTO struct {
	SpaceID   string
	TagIDs    []string
	Amount    int
	Period    model.BudgetPeriod
	StartDate time.Time
	EndDate   *time.Time
	CreatedBy string
}

type UpdateBudgetDTO struct {
	ID        string
	TagIDs    []string
	Amount    int
	Period    model.BudgetPeriod
	StartDate time.Time
	EndDate   *time.Time
}

type BudgetService struct {
	budgetRepo repository.BudgetRepository
}

func NewBudgetService(budgetRepo repository.BudgetRepository) *BudgetService {
	return &BudgetService{budgetRepo: budgetRepo}
}

func (s *BudgetService) CreateBudget(dto CreateBudgetDTO) (*model.Budget, error) {
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("budget amount must be positive")
	}

	if len(dto.TagIDs) == 0 {
		return nil, fmt.Errorf("at least one tag is required")
	}

	now := time.Now()
	budget := &model.Budget{
		ID:          uuid.NewString(),
		SpaceID:     dto.SpaceID,
		AmountCents: dto.Amount,
		Period:      dto.Period,
		StartDate:   dto.StartDate,
		EndDate:     dto.EndDate,
		IsActive:    true,
		CreatedBy:   dto.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.budgetRepo.Create(budget, dto.TagIDs); err != nil {
		return nil, err
	}
	return budget, nil
}

func (s *BudgetService) GetBudget(id string) (*model.Budget, error) {
	return s.budgetRepo.GetByID(id)
}

func (s *BudgetService) GetBudgetsWithSpent(spaceID string) ([]*model.BudgetWithSpent, error) {
	budgets, err := s.budgetRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	// Collect budget IDs for batch tag fetch
	budgetIDs := make([]string, len(budgets))
	for i, b := range budgets {
		budgetIDs[i] = b.ID
	}

	budgetTagsMap, err := s.budgetRepo.GetTagsByBudgetIDs(budgetIDs)
	if err != nil {
		return nil, err
	}

	result := make([]*model.BudgetWithSpent, 0, len(budgets))
	for _, b := range budgets {
		tags := budgetTagsMap[b.ID]

		// Extract tag IDs for spending calculation
		tagIDs := make([]string, len(tags))
		for i, t := range tags {
			tagIDs[i] = t.ID
		}

		start, end := GetCurrentPeriodBounds(b.Period, time.Now())
		spent, err := s.budgetRepo.GetSpentForBudget(spaceID, tagIDs, start, end)
		if err != nil {
			spent = 0
		}

		var percentage float64
		if b.AmountCents > 0 {
			percentage = float64(spent) / float64(b.AmountCents) * 100
		}

		var status model.BudgetStatus
		switch {
		case percentage > 100:
			status = model.BudgetStatusOver
		case percentage >= 75:
			status = model.BudgetStatusWarning
		default:
			status = model.BudgetStatusOnTrack
		}

		bws := &model.BudgetWithSpent{
			Budget:     *b,
			Tags:       tags,
			SpentCents: spent,
			Percentage: percentage,
			Status:     status,
		}

		result = append(result, bws)
	}
	return result, nil
}

func (s *BudgetService) UpdateBudget(dto UpdateBudgetDTO) (*model.Budget, error) {
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("budget amount must be positive")
	}

	if len(dto.TagIDs) == 0 {
		return nil, fmt.Errorf("at least one tag is required")
	}

	existing, err := s.budgetRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	existing.AmountCents = dto.Amount
	existing.Period = dto.Period
	existing.StartDate = dto.StartDate
	existing.EndDate = dto.EndDate
	existing.UpdatedAt = time.Now()

	if err := s.budgetRepo.Update(existing, dto.TagIDs); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *BudgetService) DeleteBudget(id string) error {
	return s.budgetRepo.Delete(id)
}

func GetCurrentPeriodBounds(period model.BudgetPeriod, now time.Time) (time.Time, time.Time) {
	switch period {
	case model.BudgetPeriodWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := now.AddDate(0, 0, -(weekday - 1))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, 6)
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, now.Location())
		return start, end
	case model.BudgetPeriodYearly:
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
		return start, end
	default: // monthly
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, -1)
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, now.Location())
		return start, end
	}
}
