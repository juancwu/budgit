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
}

func NewAccountService(accountRepo repository.AccountRepository) *AccountService {
	return &AccountService{accountRepo: accountRepo}
}

func (s *AccountService) CreateAccount(spaceID, name string) (*model.Account, error) {
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
	return account, nil
}

func (s *AccountService) GetAccount(id string) (*model.Account, error) {
	account, err := s.accountRepo.ByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return account, nil
}

func (s *AccountService) GetAccountsForSpace(spaceID string) ([]*model.Account, error) {
	accounts, err := s.accountRepo.BySpaceID(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts for space: %w", err)
	}
	return accounts, nil
}
