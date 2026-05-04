package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/misc/currency"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const DefaultAccountName = "Money Account"

type AccountService struct {
	accountRepo    repository.AccountRepository
	allocationRepo repository.AllocationRepository
	auditSvc       *SpaceAuditLogService
}

func NewAccountService(accountRepo repository.AccountRepository) *AccountService {
	return &AccountService{accountRepo: accountRepo}
}

// SetAllocationRepository wires the allocation repository after construction.
// Required for currency conversion to rewrite allocation amounts atomically.
func (s *AccountService) SetAllocationRepository(repo repository.AllocationRepository) {
	s.allocationRepo = repo
}

// SetAuditLogger wires the audit log service after construction.
func (s *AccountService) SetAuditLogger(audit *SpaceAuditLogService) {
	s.auditSvc = audit
}

func (s *AccountService) CreateAccount(spaceID, name, currencyCode, actorID string) (*model.Account, error) {
	if spaceID == "" {
		return nil, fmt.Errorf("space id is required")
	}
	if name == "" {
		return nil, fmt.Errorf("account name cannot be empty")
	}

	code := currency.Normalize(currencyCode)
	if code == "" {
		code = currency.Default
	}
	if !currency.IsValid(code) {
		return nil, fmt.Errorf("unsupported currency code: %s", currencyCode)
	}

	now := time.Now()
	account := &model.Account{
		ID:        uuid.NewString(),
		Name:      name,
		SpaceID:   spaceID,
		Currency:  code,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.accountRepo.Create(account); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}
	s.auditSvc.Record(RecordOptions{
		SpaceID: spaceID,
		ActorID: actorID,
		Action:  model.SpaceAuditActionAccountCreated,
		Metadata: map[string]any{
			"account_id":   account.ID,
			"account_name": account.Name,
			"currency":     account.Currency,
		},
	})
	return account, nil
}

func (s *AccountService) GetAccount(id string) (*model.Account, error) {
	account, err := s.accountRepo.ByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return account, nil
}

func (s *AccountService) RenameAccount(id, name, actorID string) error {
	if id == "" {
		return fmt.Errorf("account id is required")
	}
	if name == "" {
		return fmt.Errorf("account name cannot be empty")
	}
	current, err := s.accountRepo.ByID(id)
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}
	oldName := current.Name
	if err := s.accountRepo.Rename(id, name); err != nil {
		return fmt.Errorf("failed to rename account: %w", err)
	}
	if oldName != name {
		s.auditSvc.Record(RecordOptions{
			SpaceID: current.SpaceID,
			ActorID: actorID,
			Action:  model.SpaceAuditActionAccountRenamed,
			Metadata: map[string]any{
				"account_id": id,
				"old_name":   oldName,
				"new_name":   name,
			},
		})
	}
	return nil
}

func (s *AccountService) DeleteAccount(id, actorID string) error {
	if id == "" {
		return fmt.Errorf("account id is required")
	}
	current, err := s.accountRepo.ByID(id)
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}
	// Record before deleting so the audit row references the pre-delete state.
	s.auditSvc.Record(RecordOptions{
		SpaceID: current.SpaceID,
		ActorID: actorID,
		Action:  model.SpaceAuditActionAccountDeleted,
		Metadata: map[string]any{
			"account_id":   id,
			"account_name": current.Name,
		},
	})
	if err := s.accountRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

// ChangeCurrency converts the account's currency. Every value held in the old
// currency (account balance, allocation amounts and targets) is multiplied by
// rate and rounded to 2 decimals. The whole change is applied in a single SQL
// transaction so the account never appears in a half-converted state.
//
// rate is "1 oldCurrency = rate newCurrency". Same-currency changes are
// rejected; callers should treat rate as required and positive.
func (s *AccountService) ChangeCurrency(accountID, newCurrencyCode string, rate decimal.Decimal, actorID string) error {
	if accountID == "" {
		return fmt.Errorf("account id is required")
	}
	code := currency.Normalize(newCurrencyCode)
	if !currency.IsValid(code) {
		return fmt.Errorf("unsupported currency code: %s", newCurrencyCode)
	}
	if !rate.IsPositive() {
		return fmt.Errorf("conversion rate must be greater than zero")
	}

	account, err := s.accountRepo.ByID(accountID)
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}
	if account.Currency == code {
		return fmt.Errorf("account is already in %s", code)
	}

	allocations, err := s.allocationRepo.ByAccountID(accountID)
	if err != nil {
		return fmt.Errorf("failed to load allocations: %w", err)
	}

	newBalance := account.Balance.Mul(rate).Round(2)
	conversions := make([]repository.AllocationConversion, 0, len(allocations))
	for _, a := range allocations {
		c := repository.AllocationConversion{
			ID:     a.ID,
			Amount: a.Amount.Mul(rate).Round(2),
		}
		if a.TargetAmount != nil {
			t := a.TargetAmount.Mul(rate).Round(2)
			c.TargetAmount = &t
		}
		conversions = append(conversions, c)
	}

	if err := s.accountRepo.ChangeCurrency(accountID, code, newBalance, conversions); err != nil {
		return fmt.Errorf("failed to change currency: %w", err)
	}

	s.auditSvc.Record(RecordOptions{
		SpaceID: account.SpaceID,
		ActorID: actorID,
		Action:  model.SpaceAuditActionAccountCurrencyChanged,
		Metadata: map[string]any{
			"account_id":      accountID,
			"account_name":    account.Name,
			"old_currency":    account.Currency,
			"new_currency":    code,
			"conversion_rate": rate.String(),
			"old_balance":     account.Balance.StringFixedBank(2),
			"new_balance":     newBalance.StringFixedBank(2),
		},
	})
	return nil
}

func (s *AccountService) GetAccountsForSpace(spaceID string) ([]*model.Account, error) {
	accounts, err := s.accountRepo.BySpaceID(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts for space: %w", err)
	}
	return accounts, nil
}
