package service

import (
	"fmt"
	"log/slog"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreateRecurringExpenseDTO struct {
	SpaceID         string
	UserID          string
	Description     string
	Amount          int
	Type            model.ExpenseType
	PaymentMethodID *string
	Frequency       model.Frequency
	StartDate       time.Time
	EndDate         *time.Time
	TagIDs          []string
}

type UpdateRecurringExpenseDTO struct {
	ID              string
	Description     string
	Amount          int
	Type            model.ExpenseType
	PaymentMethodID *string
	Frequency       model.Frequency
	StartDate       time.Time
	EndDate         *time.Time
	TagIDs          []string
}

type RecurringExpenseService struct {
	recurringRepo repository.RecurringExpenseRepository
	expenseRepo   repository.ExpenseRepository
}

func NewRecurringExpenseService(recurringRepo repository.RecurringExpenseRepository, expenseRepo repository.ExpenseRepository) *RecurringExpenseService {
	return &RecurringExpenseService{
		recurringRepo: recurringRepo,
		expenseRepo:   expenseRepo,
	}
}

func (s *RecurringExpenseService) CreateRecurringExpense(dto CreateRecurringExpenseDTO) (*model.RecurringExpense, error) {
	if dto.Description == "" {
		return nil, fmt.Errorf("description cannot be empty")
	}
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	now := time.Now()
	re := &model.RecurringExpense{
		ID:              uuid.NewString(),
		SpaceID:         dto.SpaceID,
		CreatedBy:       dto.UserID,
		Description:     dto.Description,
		AmountCents:     dto.Amount,
		Type:            dto.Type,
		PaymentMethodID: dto.PaymentMethodID,
		Frequency:       dto.Frequency,
		StartDate:       dto.StartDate,
		EndDate:         dto.EndDate,
		NextOccurrence:  dto.StartDate,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.recurringRepo.Create(re, dto.TagIDs); err != nil {
		return nil, err
	}
	return re, nil
}

func (s *RecurringExpenseService) GetRecurringExpense(id string) (*model.RecurringExpense, error) {
	return s.recurringRepo.GetByID(id)
}

func (s *RecurringExpenseService) GetRecurringExpensesForSpace(spaceID string) ([]*model.RecurringExpense, error) {
	return s.recurringRepo.GetBySpaceID(spaceID)
}

func (s *RecurringExpenseService) GetRecurringExpensesWithTagsAndMethodsForSpace(spaceID string) ([]*model.RecurringExpenseWithTagsAndMethod, error) {
	recs, err := s.recurringRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(recs))
	for i, re := range recs {
		ids[i] = re.ID
	}

	tagsMap, err := s.recurringRepo.GetTagsByRecurringExpenseIDs(ids)
	if err != nil {
		return nil, err
	}

	methodsMap, err := s.recurringRepo.GetPaymentMethodsByRecurringExpenseIDs(ids)
	if err != nil {
		return nil, err
	}

	result := make([]*model.RecurringExpenseWithTagsAndMethod, len(recs))
	for i, re := range recs {
		result[i] = &model.RecurringExpenseWithTagsAndMethod{
			RecurringExpense: *re,
			Tags:             tagsMap[re.ID],
			PaymentMethod:    methodsMap[re.ID],
		}
	}
	return result, nil
}

func (s *RecurringExpenseService) UpdateRecurringExpense(dto UpdateRecurringExpenseDTO) (*model.RecurringExpense, error) {
	if dto.Description == "" {
		return nil, fmt.Errorf("description cannot be empty")
	}
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	existing, err := s.recurringRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	existing.Description = dto.Description
	existing.AmountCents = dto.Amount
	existing.Type = dto.Type
	existing.PaymentMethodID = dto.PaymentMethodID
	existing.Frequency = dto.Frequency
	existing.StartDate = dto.StartDate
	existing.EndDate = dto.EndDate
	existing.UpdatedAt = time.Now()

	// Recalculate next occurrence if frequency or start changed
	if existing.NextOccurrence.Before(dto.StartDate) {
		existing.NextOccurrence = dto.StartDate
	}

	if err := s.recurringRepo.Update(existing, dto.TagIDs); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *RecurringExpenseService) DeleteRecurringExpense(id string) error {
	return s.recurringRepo.Delete(id)
}

func (s *RecurringExpenseService) ToggleRecurringExpense(id string) (*model.RecurringExpense, error) {
	re, err := s.recurringRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	newActive := !re.IsActive
	if err := s.recurringRepo.SetActive(id, newActive); err != nil {
		return nil, err
	}
	re.IsActive = newActive
	return re, nil
}

func (s *RecurringExpenseService) ProcessDueRecurrences(now time.Time) error {
	dues, err := s.recurringRepo.GetDueRecurrences(now)
	if err != nil {
		return fmt.Errorf("failed to get due recurrences: %w", err)
	}

	for _, re := range dues {
		if err := s.processRecurrence(re, now); err != nil {
			slog.Error("failed to process recurring expense", "id", re.ID, "error", err)
		}
	}
	return nil
}

func (s *RecurringExpenseService) ProcessDueRecurrencesForSpace(spaceID string, now time.Time) error {
	dues, err := s.recurringRepo.GetDueRecurrencesForSpace(spaceID, now)
	if err != nil {
		return fmt.Errorf("failed to get due recurrences for space: %w", err)
	}

	for _, re := range dues {
		if err := s.processRecurrence(re, now); err != nil {
			slog.Error("failed to process recurring expense", "id", re.ID, "error", err)
		}
	}
	return nil
}

func (s *RecurringExpenseService) processRecurrence(re *model.RecurringExpense, now time.Time) error {
	// Get tag IDs for this recurring expense
	tagsMap, err := s.recurringRepo.GetTagsByRecurringExpenseIDs([]string{re.ID})
	if err != nil {
		return err
	}
	var tagIDs []string
	for _, t := range tagsMap[re.ID] {
		tagIDs = append(tagIDs, t.ID)
	}

	// Generate expenses for each missed occurrence up to now
	for !re.NextOccurrence.After(now) {
		// Check if end_date has been passed
		if re.EndDate != nil && re.NextOccurrence.After(*re.EndDate) {
			return s.recurringRepo.Deactivate(re.ID)
		}

		expense := &model.Expense{
			ID:                 uuid.NewString(),
			SpaceID:            re.SpaceID,
			CreatedBy:          re.CreatedBy,
			Description:        re.Description,
			AmountCents:        re.AmountCents,
			Type:               re.Type,
			Date:               re.NextOccurrence,
			PaymentMethodID:    re.PaymentMethodID,
			RecurringExpenseID: &re.ID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if err := s.expenseRepo.Create(expense, tagIDs, nil); err != nil {
			return fmt.Errorf("failed to create expense from recurring: %w", err)
		}

		re.NextOccurrence = AdvanceDate(re.NextOccurrence, re.Frequency)
	}

	// Check if the new next occurrence exceeds end date
	if re.EndDate != nil && re.NextOccurrence.After(*re.EndDate) {
		if err := s.recurringRepo.Deactivate(re.ID); err != nil {
			return err
		}
	}

	return s.recurringRepo.UpdateNextOccurrence(re.ID, re.NextOccurrence)
}

func AdvanceDate(date time.Time, freq model.Frequency) time.Time {
	switch freq {
	case model.FrequencyDaily:
		return date.AddDate(0, 0, 1)
	case model.FrequencyWeekly:
		return date.AddDate(0, 0, 7)
	case model.FrequencyBiweekly:
		return date.AddDate(0, 0, 14)
	case model.FrequencyMonthly:
		return date.AddDate(0, 1, 0)
	case model.FrequencyYearly:
		return date.AddDate(1, 0, 0)
	default:
		return date.AddDate(0, 1, 0)
	}
}
