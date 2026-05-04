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

type AllocationService struct {
	repo           repository.AllocationRepository
	accountService *AccountService
	auditSvc       *SpaceAuditLogService
}

func NewAllocationService(repo repository.AllocationRepository, accountService *AccountService) *AllocationService {
	return &AllocationService{repo: repo, accountService: accountService}
}

func (s *AllocationService) SetAuditLogger(audit *SpaceAuditLogService) {
	s.auditSvc = audit
}

// AllocationSummary bundles the allocations for an account with derived totals
// the UI cares about (Available cash, over-allocation flag).
type AllocationSummary struct {
	Allocations []*model.Allocation
	Allocated   decimal.Decimal
	Available   decimal.Decimal
	Overflow    bool // true when sum(allocations) > account.balance
}

type CreateAllocationInput struct {
	AccountID    string
	Name         string
	Amount       decimal.Decimal
	TargetAmount *decimal.Decimal
	ActorID      string
}

func (s *AllocationService) Create(input CreateAllocationInput) (*model.Allocation, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.AccountID == "" {
		return nil, fmt.Errorf("account id is required")
	}
	if input.Amount.IsNegative() {
		return nil, fmt.Errorf("amount cannot be negative")
	}
	if input.TargetAmount != nil && input.TargetAmount.IsNegative() {
		return nil, fmt.Errorf("target cannot be negative")
	}

	account, err := s.accountService.GetAccount(input.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}

	now := time.Now()
	a := &model.Allocation{
		ID:           uuid.NewString(),
		AccountID:    input.AccountID,
		Name:         name,
		Amount:       input.Amount,
		TargetAmount: input.TargetAmount,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(a); err != nil {
		return nil, fmt.Errorf("failed to create allocation: %w", err)
	}

	s.auditSvc.Record(RecordOptions{
		SpaceID: account.SpaceID,
		ActorID: input.ActorID,
		Action:  model.SpaceAuditActionAllocationCreated,
		Metadata: map[string]any{
			"account_id":    a.AccountID,
			"allocation_id": a.ID,
			"name":          a.Name,
			"amount":        a.Amount.StringFixedBank(2),
			"target":        targetString(a.TargetAmount),
		},
	})
	return a, nil
}

type UpdateAllocationInput struct {
	AllocationID string
	Name         string
	Amount       decimal.Decimal
	TargetAmount *decimal.Decimal
	ActorID      string
}

func (s *AllocationService) Update(input UpdateAllocationInput) (*model.Allocation, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.AllocationID == "" {
		return nil, fmt.Errorf("allocation id is required")
	}
	if input.Amount.IsNegative() {
		return nil, fmt.Errorf("amount cannot be negative")
	}
	if input.TargetAmount != nil && input.TargetAmount.IsNegative() {
		return nil, fmt.Errorf("target cannot be negative")
	}

	existing, err := s.repo.ByID(input.AllocationID)
	if err != nil {
		return nil, fmt.Errorf("failed to load allocation: %w", err)
	}
	account, err := s.accountService.GetAccount(existing.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}

	changes := map[string]any{}
	if existing.Name != name {
		changes["name"] = map[string]any{"old": existing.Name, "new": name}
	}
	if !existing.Amount.Equal(input.Amount) {
		changes["amount"] = map[string]any{
			"old": existing.Amount.StringFixedBank(2),
			"new": input.Amount.StringFixedBank(2),
		}
	}
	if !decimalPtrEq(existing.TargetAmount, input.TargetAmount) {
		changes["target"] = map[string]any{
			"old": targetString(existing.TargetAmount),
			"new": targetString(input.TargetAmount),
		}
	}

	if err := s.repo.Update(input.AllocationID, name, input.Amount, input.TargetAmount); err != nil {
		return nil, fmt.Errorf("failed to update allocation: %w", err)
	}

	existing.Name = name
	existing.Amount = input.Amount
	existing.TargetAmount = input.TargetAmount
	existing.UpdatedAt = time.Now()

	if len(changes) > 0 {
		s.auditSvc.Record(RecordOptions{
			SpaceID: account.SpaceID,
			ActorID: input.ActorID,
			Action:  model.SpaceAuditActionAllocationUpdated,
			Metadata: map[string]any{
				"account_id":    existing.AccountID,
				"allocation_id": existing.ID,
				"changes":       changes,
			},
		})
	}
	return existing, nil
}

func (s *AllocationService) Delete(allocationID, actorID string) error {
	if allocationID == "" {
		return fmt.Errorf("allocation id is required")
	}
	existing, err := s.repo.ByID(allocationID)
	if err != nil {
		return fmt.Errorf("failed to load allocation: %w", err)
	}
	account, err := s.accountService.GetAccount(existing.AccountID)
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}
	// Record before delete so the row references pre-delete state.
	s.auditSvc.Record(RecordOptions{
		SpaceID: account.SpaceID,
		ActorID: actorID,
		Action:  model.SpaceAuditActionAllocationDeleted,
		Metadata: map[string]any{
			"account_id":    existing.AccountID,
			"allocation_id": existing.ID,
			"name":          existing.Name,
			"amount":        existing.Amount.StringFixedBank(2),
		},
	})
	if err := s.repo.Delete(allocationID); err != nil {
		return fmt.Errorf("failed to delete allocation: %w", err)
	}
	return nil
}

func (s *AllocationService) Get(id string) (*model.Allocation, error) {
	a, err := s.repo.ByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load allocation: %w", err)
	}
	return a, nil
}

// SummaryForAccount returns the allocations for an account along with the
// derived Allocated/Available figures used by the UI banner.
func (s *AllocationService) SummaryForAccount(accountID string) (*AllocationSummary, error) {
	account, err := s.accountService.GetAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}
	allocs, err := s.repo.ByAccountID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load allocations: %w", err)
	}
	allocated := decimal.Zero
	for _, a := range allocs {
		allocated = allocated.Add(a.Amount)
	}
	available := account.Balance.Sub(allocated)
	return &AllocationSummary{
		Allocations: allocs,
		Allocated:   allocated,
		Available:   available,
		Overflow:    available.IsNegative(),
	}, nil
}

func targetString(t *decimal.Decimal) string {
	if t == nil {
		return ""
	}
	return t.StringFixedBank(2)
}

func decimalPtrEq(a, b *decimal.Decimal) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}
