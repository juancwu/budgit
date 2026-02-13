package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreateExpenseDTO struct {
	SpaceID         string
	UserID          string
	Description     string
	Amount          int
	Type            model.ExpenseType
	Date            time.Time
	TagIDs          []string
	ItemIDs         []string
	PaymentMethodID *string
}

type UpdateExpenseDTO struct {
	ID              string
	SpaceID         string
	Description     string
	Amount          int
	Type            model.ExpenseType
	Date            time.Time
	TagIDs          []string
	PaymentMethodID *string
}

const ExpensesPerPage = 25

type ExpenseService struct {
	expenseRepo repository.ExpenseRepository
}

func NewExpenseService(expenseRepo repository.ExpenseRepository) *ExpenseService {
	return &ExpenseService{
		expenseRepo: expenseRepo,
	}
}

func (s *ExpenseService) CreateExpense(dto CreateExpenseDTO) (*model.Expense, error) {
	if dto.Description == "" {
		return nil, fmt.Errorf("expense description cannot be empty")
	}
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	now := time.Now()
	expense := &model.Expense{
		ID:              uuid.NewString(),
		SpaceID:         dto.SpaceID,
		CreatedBy:       dto.UserID,
		Description:     dto.Description,
		AmountCents:     dto.Amount,
		Type:            dto.Type,
		Date:            dto.Date,
		PaymentMethodID: dto.PaymentMethodID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := s.expenseRepo.Create(expense, dto.TagIDs, dto.ItemIDs)
	if err != nil {
		return nil, err
	}

	return expense, nil
}

func (s *ExpenseService) GetExpensesForSpace(spaceID string) ([]*model.Expense, error) {
	return s.expenseRepo.GetBySpaceID(spaceID)
}

func (s *ExpenseService) GetBalanceForSpace(spaceID string) (int, error) {
	expenses, err := s.expenseRepo.GetBySpaceID(spaceID)
	if err != nil {
		return 0, err
	}

	var balance int
	for _, expense := range expenses {
		if expense.Type == model.ExpenseTypeExpense {
			balance -= expense.AmountCents
		} else if expense.Type == model.ExpenseTypeTopup {
			balance += expense.AmountCents
		}
	}

	return balance, nil
}

func (s *ExpenseService) GetExpensesByTag(spaceID string, fromDate, toDate time.Time) ([]*model.TagExpenseSummary, error) {
	return s.expenseRepo.GetExpensesByTag(spaceID, fromDate, toDate)
}

func (s *ExpenseService) GetExpensesWithTagsForSpace(spaceID string) ([]*model.ExpenseWithTags, error) {
	expenses, err := s.expenseRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(expenses))
	for i, e := range expenses {
		ids[i] = e.ID
	}

	tagsMap, err := s.expenseRepo.GetTagsByExpenseIDs(ids)
	if err != nil {
		return nil, err
	}

	result := make([]*model.ExpenseWithTags, len(expenses))
	for i, e := range expenses {
		result[i] = &model.ExpenseWithTags{
			Expense: *e,
			Tags:    tagsMap[e.ID],
		}
	}
	return result, nil
}

func (s *ExpenseService) GetExpensesWithTagsForSpacePaginated(spaceID string, page int) ([]*model.ExpenseWithTags, int, error) {
	total, err := s.expenseRepo.CountBySpaceID(spaceID)
	if err != nil {
		return nil, 0, err
	}

	totalPages := (total + ExpensesPerPage - 1) / ExpensesPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * ExpensesPerPage
	expenses, err := s.expenseRepo.GetBySpaceIDPaginated(spaceID, ExpensesPerPage, offset)
	if err != nil {
		return nil, 0, err
	}

	ids := make([]string, len(expenses))
	for i, e := range expenses {
		ids[i] = e.ID
	}

	tagsMap, err := s.expenseRepo.GetTagsByExpenseIDs(ids)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*model.ExpenseWithTags, len(expenses))
	for i, e := range expenses {
		result[i] = &model.ExpenseWithTags{
			Expense: *e,
			Tags:    tagsMap[e.ID],
		}
	}
	return result, totalPages, nil
}

func (s *ExpenseService) GetExpensesWithTagsAndMethodsForSpacePaginated(spaceID string, page int) ([]*model.ExpenseWithTagsAndMethod, int, error) {
	total, err := s.expenseRepo.CountBySpaceID(spaceID)
	if err != nil {
		return nil, 0, err
	}

	totalPages := (total + ExpensesPerPage - 1) / ExpensesPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * ExpensesPerPage
	expenses, err := s.expenseRepo.GetBySpaceIDPaginated(spaceID, ExpensesPerPage, offset)
	if err != nil {
		return nil, 0, err
	}

	ids := make([]string, len(expenses))
	for i, e := range expenses {
		ids[i] = e.ID
	}

	tagsMap, err := s.expenseRepo.GetTagsByExpenseIDs(ids)
	if err != nil {
		return nil, 0, err
	}

	methodsMap, err := s.expenseRepo.GetPaymentMethodsByExpenseIDs(ids)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*model.ExpenseWithTagsAndMethod, len(expenses))
	for i, e := range expenses {
		result[i] = &model.ExpenseWithTagsAndMethod{
			Expense:       *e,
			Tags:          tagsMap[e.ID],
			PaymentMethod: methodsMap[e.ID],
		}
	}
	return result, totalPages, nil
}

func (s *ExpenseService) GetPaymentMethodsByExpenseIDs(expenseIDs []string) (map[string]*model.PaymentMethod, error) {
	return s.expenseRepo.GetPaymentMethodsByExpenseIDs(expenseIDs)
}

func (s *ExpenseService) GetExpense(id string) (*model.Expense, error) {
	return s.expenseRepo.GetByID(id)
}

func (s *ExpenseService) GetTagsByExpenseIDs(expenseIDs []string) (map[string][]*model.Tag, error) {
	return s.expenseRepo.GetTagsByExpenseIDs(expenseIDs)
}

func (s *ExpenseService) UpdateExpense(dto UpdateExpenseDTO) (*model.Expense, error) {
	if dto.Description == "" {
		return nil, fmt.Errorf("expense description cannot be empty")
	}
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	existing, err := s.expenseRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	existing.Description = dto.Description
	existing.AmountCents = dto.Amount
	existing.Type = dto.Type
	existing.Date = dto.Date
	existing.PaymentMethodID = dto.PaymentMethodID
	existing.UpdatedAt = time.Now()

	if err := s.expenseRepo.Update(existing, dto.TagIDs); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *ExpenseService) DeleteExpense(id string, spaceID string) error {
	if err := s.expenseRepo.Delete(id); err != nil {
		return err
	}

	return nil
}
