package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ErrTransactionPartOfTransfer is returned when an operation that mutates a
// single transaction (edit, delete) is attempted on one half of a transfer.
// Transfers must be edited as a pair or not at all to keep both sides in sync;
// callers should surface a user-facing message and offer to undo the transfer
// instead.
var ErrTransactionPartOfTransfer = errors.New("transaction is part of a transfer")

type TransactionService struct {
	transactionRepo repository.TransactionRepository
	categoryRepo    repository.CategoryRepository
	accountService  *AccountService
	auditSvc        *TransactionAuditLogService
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

// SetAuditLogger wires the audit log service after construction.
func (s *TransactionService) SetAuditLogger(audit *TransactionAuditLogService) {
	s.auditSvc = audit
}

type PayBillInput struct {
	AccountID   string
	Title       string
	Amount      decimal.Decimal
	OccurredAt  time.Time
	Description string
	CategoryID  string
	ActorID     string
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

	s.auditSvc.Record(TransactionRecordOptions{
		TransactionID: txn.ID,
		ActorID:       input.ActorID,
		Action:        model.TransactionAuditActionCreated,
		Metadata: map[string]any{
			"account_id":       txn.AccountID,
			"transaction_type": string(model.TransactionTypeWithdrawal),
			"title":            txn.Title,
			"amount":           txn.Value.StringFixedBank(2),
		},
	})

	return txn, nil
}

type DepositInput struct {
	AccountID   string
	Title       string
	Amount      decimal.Decimal
	OccurredAt  time.Time
	Description string
	ActorID     string
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

	s.auditSvc.Record(TransactionRecordOptions{
		TransactionID: txn.ID,
		ActorID:       input.ActorID,
		Action:        model.TransactionAuditActionCreated,
		Metadata: map[string]any{
			"account_id":       txn.AccountID,
			"transaction_type": string(model.TransactionTypeDeposit),
			"title":            txn.Title,
			"amount":           txn.Value.StringFixedBank(2),
		},
	})

	return txn, nil
}

type TransferInput struct {
	SourceAccountID string
	DestAccountID   string
	Title           string
	Amount          decimal.Decimal
	// ConversionRate is the rate that converts one unit of the source currency
	// into the destination currency. Required when source and destination
	// accounts have different currencies; ignored otherwise. Must be positive.
	ConversionRate decimal.Decimal
	OccurredAt     time.Time
	Description    string
	ActorID        string
}

// TransferResult is what the service returns after a successful transfer — both
// halves are surfaced so callers can audit, redirect, or render either side.
type TransferResult struct {
	Withdrawal *model.Transaction
	Deposit    *model.Transaction
}

// Transfer moves funds from one account to another. It creates two linked
// transactions (a withdrawal on the source, a deposit on the destination) plus
// a row in related_transactions, all in a single SQL transaction.
//
// Negative balances are intentionally allowed — the product permits overdraft.
// Source must differ from destination; the amount must be positive (the sign is
// implicit in the transaction type).
func (s *TransactionService) Transfer(input TransferInput) (*TransferResult, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.SourceAccountID == "" || input.DestAccountID == "" {
		return nil, fmt.Errorf("source and destination account ids are required")
	}
	if input.SourceAccountID == input.DestAccountID {
		return nil, fmt.Errorf("source and destination must differ")
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if input.OccurredAt.IsZero() {
		return nil, fmt.Errorf("date is required")
	}

	source, err := s.accountService.GetAccount(input.SourceAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load source account: %w", err)
	}
	dest, err := s.accountService.GetAccount(input.DestAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load destination account: %w", err)
	}

	// Cross-currency transfers require a conversion rate; same-currency
	// transfers ignore it (or, for symmetry, accept rate=1).
	destAmount := input.Amount
	rate := decimal.NewFromInt(1)
	if source.Currency != dest.Currency {
		if !input.ConversionRate.IsPositive() {
			return nil, fmt.Errorf("conversion rate is required when transferring between accounts of different currencies")
		}
		rate = input.ConversionRate
		destAmount = input.Amount.Mul(rate).Round(2)
	}

	now := time.Now()
	var description *string
	if d := strings.TrimSpace(input.Description); d != "" {
		description = &d
	}

	withdrawal := &model.Transaction{
		ID:          uuid.NewString(),
		Value:       input.Amount,
		Type:        model.TransactionTypeWithdrawal,
		AccountID:   source.ID,
		Title:       title,
		Description: description,
		OccurredAt:  input.OccurredAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	deposit := &model.Transaction{
		ID:          uuid.NewString(),
		Value:       destAmount,
		Type:        model.TransactionTypeDeposit,
		AccountID:   dest.ID,
		Title:       title,
		Description: description,
		OccurredAt:  input.OccurredAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	sourceNewBalance := source.Balance.Sub(input.Amount)
	destNewBalance := dest.Balance.Add(destAmount)

	if err := s.transactionRepo.TransferAtomic(withdrawal, deposit, sourceNewBalance, destNewBalance); err != nil {
		return nil, fmt.Errorf("failed to record transfer: %w", err)
	}

	// Audit each side. Metadata captures the role and the other half so the
	// activity feed can render "Transferred to/from <other account>" without a
	// follow-up query.
	s.auditSvc.Record(TransactionRecordOptions{
		TransactionID: withdrawal.ID,
		ActorID:       input.ActorID,
		Action:        model.TransactionAuditActionCreated,
		Metadata: map[string]any{
			"account_id":          withdrawal.AccountID,
			"transaction_type":    string(withdrawal.Type),
			"title":               withdrawal.Title,
			"amount":              withdrawal.Value.StringFixedBank(2),
			"transfer_role":       "source",
			"transfer_pair_id":    deposit.ID,
			"transfer_other_acct": deposit.AccountID,
			"transfer_other_name": dest.Name,
			"source_currency":     source.Currency,
			"dest_currency":       dest.Currency,
			"conversion_rate":     rate.String(),
			"dest_amount":         destAmount.StringFixedBank(2),
		},
	})
	s.auditSvc.Record(TransactionRecordOptions{
		TransactionID: deposit.ID,
		ActorID:       input.ActorID,
		Action:        model.TransactionAuditActionCreated,
		Metadata: map[string]any{
			"account_id":          deposit.AccountID,
			"transaction_type":    string(deposit.Type),
			"title":               deposit.Title,
			"amount":              deposit.Value.StringFixedBank(2),
			"transfer_role":       "destination",
			"transfer_pair_id":    withdrawal.ID,
			"transfer_other_acct": withdrawal.AccountID,
			"transfer_other_name": source.Name,
			"source_currency":     source.Currency,
			"dest_currency":       dest.Currency,
			"conversion_rate":     rate.String(),
			"source_amount":       input.Amount.StringFixedBank(2),
		},
	})

	return &TransferResult{Withdrawal: withdrawal, Deposit: deposit}, nil
}

// TransferIDsIn returns the subset of the given transaction IDs that are part
// of a transfer pair. Empty input yields an empty (non-nil) map.
func (s *TransactionService) TransferIDsIn(ids []string) (map[string]bool, error) {
	hits, err := s.transactionRepo.TransferIDsIn(ids)
	if err != nil {
		return nil, fmt.Errorf("failed to look up transfer ids: %w", err)
	}
	return hits, nil
}

// GetRelatedTransactionID returns the other half of a transfer pair, or "" if
// the transaction is not part of a transfer.
func (s *TransactionService) GetRelatedTransactionID(transactionID string) (string, error) {
	id, err := s.transactionRepo.GetRelatedID(transactionID)
	if err != nil {
		return "", fmt.Errorf("failed to load related transaction: %w", err)
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

type UpdateBillInput struct {
	TransactionID string
	Title         string
	Amount        decimal.Decimal
	OccurredAt    time.Time
	Description   string
	CategoryID    string
	ActorID       string
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
	if related, err := s.transactionRepo.GetRelatedID(existing.ID); err != nil {
		return nil, fmt.Errorf("failed to check transfer linkage: %w", err)
	} else if related != nil {
		return nil, ErrTransactionPartOfTransfer
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

	oldCategoryID, _ := s.transactionRepo.GetCategoryID(input.TransactionID)
	changes := diffTransactionFields(existing, title, input.Amount, input.OccurredAt, description)
	if !ptrEq(oldCategoryID, categoryID) {
		changes["category_id"] = map[string]any{
			"old": ptrOrEmpty(oldCategoryID),
			"new": ptrOrEmpty(categoryID),
		}
	}

	existing.Value = input.Amount
	existing.Title = title
	existing.Description = description
	existing.OccurredAt = input.OccurredAt
	existing.UpdatedAt = time.Now()

	if err := s.transactionRepo.UpdateBillAtomic(existing, newBalance, categoryID); err != nil {
		return nil, fmt.Errorf("failed to update bill transaction: %w", err)
	}
	if len(changes) > 0 {
		s.auditSvc.Record(TransactionRecordOptions{
			TransactionID: input.TransactionID,
			ActorID:       input.ActorID,
			Action:        model.TransactionAuditActionEdited,
			Metadata: map[string]any{
				"account_id":       existing.AccountID,
				"transaction_type": string(existing.Type),
				"changes":          changes,
			},
		})
	}
	return existing, nil
}

type UpdateDepositInput struct {
	TransactionID string
	Title         string
	Amount        decimal.Decimal
	OccurredAt    time.Time
	Description   string
	ActorID       string
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
	if related, err := s.transactionRepo.GetRelatedID(existing.ID); err != nil {
		return nil, fmt.Errorf("failed to check transfer linkage: %w", err)
	} else if related != nil {
		return nil, ErrTransactionPartOfTransfer
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

	changes := diffTransactionFields(existing, title, input.Amount, input.OccurredAt, description)

	existing.Value = input.Amount
	existing.Title = title
	existing.Description = description
	existing.OccurredAt = input.OccurredAt
	existing.UpdatedAt = time.Now()

	if err := s.transactionRepo.UpdateDepositAtomic(existing, newBalance); err != nil {
		return nil, fmt.Errorf("failed to update deposit transaction: %w", err)
	}
	if len(changes) > 0 {
		s.auditSvc.Record(TransactionRecordOptions{
			TransactionID: input.TransactionID,
			ActorID:       input.ActorID,
			Action:        model.TransactionAuditActionEdited,
			Metadata: map[string]any{
				"account_id":       existing.AccountID,
				"transaction_type": string(existing.Type),
				"changes":          changes,
			},
		})
	}
	return existing, nil
}

type DeleteTransactionInput struct {
	TransactionID string
	ActorID       string
}

// DeleteTransaction removes a standalone bill or deposit. Transfers are
// rejected with ErrTransactionPartOfTransfer — they must be undone via the
// transfer flow so both halves stay consistent. Deleting a bill credits the
// account; deleting a deposit debits it.
func (s *TransactionService) DeleteTransaction(input DeleteTransactionInput) (*model.Transaction, error) {
	if input.TransactionID == "" {
		return nil, fmt.Errorf("transaction id is required")
	}

	existing, err := s.transactionRepo.GetByID(input.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load transaction: %w", err)
	}
	if related, err := s.transactionRepo.GetRelatedID(existing.ID); err != nil {
		return nil, fmt.Errorf("failed to check transfer linkage: %w", err)
	} else if related != nil {
		return nil, ErrTransactionPartOfTransfer
	}

	account, err := s.accountService.GetAccount(existing.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}

	var newBalance decimal.Decimal
	switch existing.Type {
	case model.TransactionTypeWithdrawal:
		newBalance = account.Balance.Add(existing.Value)
	case model.TransactionTypeDeposit:
		newBalance = account.Balance.Sub(existing.Value)
	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", existing.Type)
	}

	if err := s.transactionRepo.DeleteAtomic(existing.ID, existing.AccountID, newBalance); err != nil {
		return nil, fmt.Errorf("failed to delete transaction: %w", err)
	}

	s.auditSvc.Record(TransactionRecordOptions{
		TransactionID: existing.ID,
		ActorID:       input.ActorID,
		Action:        model.TransactionAuditActionDeleted,
		Metadata: map[string]any{
			"account_id":       existing.AccountID,
			"transaction_type": string(existing.Type),
			"title":            existing.Title,
			"amount":           existing.Value.StringFixedBank(2),
		},
	})

	return existing, nil
}

// diffTransactionFields returns a map of field name to {old, new} for fields whose
// new value differs from the existing transaction.
func diffTransactionFields(existing *model.Transaction, newTitle string, newAmount decimal.Decimal, newOccurredAt time.Time, newDescription *string) map[string]any {
	changes := map[string]any{}
	if existing.Title != newTitle {
		changes["title"] = map[string]any{"old": existing.Title, "new": newTitle}
	}
	if !existing.Value.Equal(newAmount) {
		changes["amount"] = map[string]any{
			"old": existing.Value.StringFixedBank(2),
			"new": newAmount.StringFixedBank(2),
		}
	}
	if !existing.OccurredAt.Equal(newOccurredAt) {
		changes["occurred_at"] = map[string]any{
			"old": existing.OccurredAt.Format("2006-01-02"),
			"new": newOccurredAt.Format("2006-01-02"),
		}
	}
	if !ptrStringEq(existing.Description, newDescription) {
		changes["description"] = map[string]any{
			"old": ptrOrEmpty(existing.Description),
			"new": ptrOrEmpty(newDescription),
		}
	}
	return changes
}

func ptrStringEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrEq(a, b *string) bool {
	return ptrStringEq(a, b)
}

func ptrOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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
