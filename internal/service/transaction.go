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

type DepositInput struct {
	AccountID   string
	Title       string
	Amount      decimal.Decimal
	OccurredAt  time.Time
	Description string
}

func (s *TransactionService) Deposit(input DepositInput) (*model.Transaction, error) {
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

	newBalance := account.Balance.Add(input.Amount)

	now := time.Now()
	var description *string
	if d := strings.TrimSpace(input.Description); d != "" {
		description = &d
	}

	txn := &model.Transaction{
		ID:          uuid.NewString(),
		Value:       input.Amount,
		Type:        model.TransactionTypeDeposit,
		AccountID:   input.AccountID,
		Title:       title,
		Description: description,
		OccurredAt:  input.OccurredAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.transactionRepo.CreateDepositAtomic(txn, newBalance); err != nil {
		return nil, fmt.Errorf("failed to create deposit transaction: %w", err)
	}

	return txn, nil
}

type UpdateBillInput struct {
	TransactionID string
	Title         string
	Amount        decimal.Decimal
	OccurredAt    time.Time
	Description   string
	CategoryID    string
}

func (s *TransactionService) UpdateBill(input UpdateBillInput) (*model.Transaction, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.TransactionID == "" {
		return nil, fmt.Errorf("transaction id is required")
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if input.OccurredAt.IsZero() {
		return nil, fmt.Errorf("date is required")
	}

	existing, err := s.transactionRepo.GetByID(input.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load transaction: %w", err)
	}
	if existing.Type != model.TransactionTypeWithdrawal {
		return nil, fmt.Errorf("transaction is not a bill")
	}

	account, err := s.accountService.GetAccount(existing.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}

	newBalance := account.Balance.Add(existing.Value).Sub(input.Amount)

	var description *string
	if d := strings.TrimSpace(input.Description); d != "" {
		description = &d
	}
	var categoryID *string
	if c := strings.TrimSpace(input.CategoryID); c != "" {
		categoryID = &c
	}

	existing.Value = input.Amount
	existing.Title = title
	existing.Description = description
	existing.OccurredAt = input.OccurredAt
	existing.UpdatedAt = time.Now()

	if err := s.transactionRepo.UpdateBillAtomic(existing, newBalance, categoryID); err != nil {
		return nil, fmt.Errorf("failed to update bill transaction: %w", err)
	}
	return existing, nil
}

type UpdateDepositInput struct {
	TransactionID string
	Title         string
	Amount        decimal.Decimal
	OccurredAt    time.Time
	Description   string
}

func (s *TransactionService) UpdateDeposit(input UpdateDepositInput) (*model.Transaction, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.TransactionID == "" {
		return nil, fmt.Errorf("transaction id is required")
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if input.OccurredAt.IsZero() {
		return nil, fmt.Errorf("date is required")
	}

	existing, err := s.transactionRepo.GetByID(input.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load transaction: %w", err)
	}
	if existing.Type != model.TransactionTypeDeposit {
		return nil, fmt.Errorf("transaction is not a deposit")
	}

	account, err := s.accountService.GetAccount(existing.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}

	newBalance := account.Balance.Sub(existing.Value).Add(input.Amount)

	var description *string
	if d := strings.TrimSpace(input.Description); d != "" {
		description = &d
	}

	existing.Value = input.Amount
	existing.Title = title
	existing.Description = description
	existing.OccurredAt = input.OccurredAt
	existing.UpdatedAt = time.Now()

	if err := s.transactionRepo.UpdateDepositAtomic(existing, newBalance); err != nil {
		return nil, fmt.Errorf("failed to update deposit transaction: %w", err)
	}
	return existing, nil
}

func (s *TransactionService) GetTransaction(id string) (*model.Transaction, error) {
	txn, err := s.transactionRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load transaction: %w", err)
	}
	return txn, nil
}

func (s *TransactionService) GetTransactionCategoryID(transactionID string) (string, error) {
	id, err := s.transactionRepo.GetCategoryID(transactionID)
	if err != nil {
		return "", fmt.Errorf("failed to load transaction category: %w", err)
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

func (s *TransactionService) ListByAccount(accountID string, limit, offset int) ([]*model.Transaction, error) {
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	txns, err := s.transactionRepo.ListByAccount(accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	return txns, nil
}

func (s *TransactionService) CountByAccount(accountID string) (int, error) {
	count, err := s.transactionRepo.CountByAccount(accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}
	return count, nil
}

func (s *TransactionService) ListCategories() ([]*model.Category, error) {
	categories, err := s.categoryRepo.All()
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return categories, nil
}
