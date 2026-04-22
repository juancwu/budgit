package service

import (
	"fmt"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionService struct {
	transactionRepo repository.TransactionRepository
	categoryRepo    repository.CategoryRepository
	accountService  *AccountService
}

func NewTransactionService(
	transactionRepo repository.TransactionRepository,
	categoryRepo repository.CategoryRepository,
	accountService *AccountService,
) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
		accountService:  accountService,
	}
}

type PayBillInput struct {
	AccountID   string
	Title       string
	Amount      decimal.Decimal
	OccurredAt  time.Time
	Description string
	CategoryID  string
}

func (s *TransactionService) PayBill(input PayBillInput) (*model.Transaction, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.AccountID == "" {
		return nil, fmt.Errorf("account id is required")
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if input.OccurredAt.IsZero() {
		return nil, fmt.Errorf("date is required")
	}

	account, err := s.accountService.GetAccount(input.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}

	newBalance := account.Balance.Sub(input.Amount)

	now := time.Now()
	var description *string
	if d := strings.TrimSpace(input.Description); d != "" {
		description = &d
	}
	var categoryID *string
	if c := strings.TrimSpace(input.CategoryID); c != "" {
		categoryID = &c
	}

	txn := &model.Transaction{
		ID:          uuid.NewString(),
		Value:       input.Amount,
		Type:        model.TransactionTypeWithdrawal,
		AccountID:   input.AccountID,
		Title:       title,
		Description: description,
		OccurredAt:  input.OccurredAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.transactionRepo.CreateBillAtomic(txn, newBalance, categoryID); err != nil {
		return nil, fmt.Errorf("failed to create bill transaction: %w", err)
	}

	return txn, nil
}

func (s *TransactionService) ListCategories() ([]*model.Category, error) {
	categories, err := s.categoryRepo.All()
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return categories, nil
}
