package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

const DefaultAccountName = "Money Account"

type AccountService struct {
	accountRepo repository.AccountRepository
	auditSvc    *SpaceAuditLogService
}

func NewAccountService(accountRepo repository.AccountRepository) *AccountService {
	return &AccountService{accountRepo: accountRepo}
}

// SetAuditLogger wires the audit log service after construction.
func (s *AccountService) SetAuditLogger(audit *SpaceAuditLogService) {
	s.auditSvc = audit
}

func (s *AccountService) CreateAccount(spaceID, name, actorID string) (*model.Account, error) {
	if spaceID == "" {
		return nil, fmt.Errorf("space id is required")
	}
	if name == "" {
		return nil, fmt.Errorf("account name cannot be empty")
	}

	now := time.Now()
	account := &model.Account{
		ID:        uuid.NewString(),
		Name:      name,
		SpaceID:   spaceID,
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

func (s *AccountService) GetAccountsForSpace(spaceID string) ([]*model.Account, error) {
	accounts, err := s.accountRepo.BySpaceID(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts for space: %w", err)
	}
	return accounts, nil
}
