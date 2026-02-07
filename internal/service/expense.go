package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreateExpenseDTO struct {
	SpaceID     string
	UserID      string
	Description string
	Amount      int
	Type        model.ExpenseType
	Date        time.Time
	TagIDs      []string
	ItemIDs     []string
}

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
		ID:          uuid.NewString(),
		SpaceID:     dto.SpaceID,
		CreatedBy:   dto.UserID,
		Description: dto.Description,
		AmountCents: dto.Amount,
		Type:        dto.Type,
		Date:        dto.Date,
		CreatedAt:   now,
		UpdatedAt:   now,
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
