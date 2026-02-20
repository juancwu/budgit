package service

import (
	"fmt"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreateMoneyAccountDTO struct {
	SpaceID   string
	Name      string
	CreatedBy string
}

type UpdateMoneyAccountDTO struct {
	ID   string
	Name string
}

type CreateTransferDTO struct {
	AccountID string
	Amount    int
	Direction model.TransferDirection
	Note      string
	CreatedBy string
}

type MoneyAccountService struct {
	accountRepo repository.MoneyAccountRepository
}

func NewMoneyAccountService(accountRepo repository.MoneyAccountRepository) *MoneyAccountService {
	return &MoneyAccountService{
		accountRepo: accountRepo,
	}
}

func (s *MoneyAccountService) CreateAccount(dto CreateMoneyAccountDTO) (*model.MoneyAccount, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, fmt.Errorf("account name cannot be empty")
	}

	now := time.Now()
	account := &model.MoneyAccount{
		ID:        uuid.NewString(),
		SpaceID:   dto.SpaceID,
		Name:      name,
		CreatedBy: dto.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.accountRepo.Create(account)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *MoneyAccountService) GetAccountsForSpace(spaceID string) ([]model.MoneyAccountWithBalance, error) {
	accounts, err := s.accountRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	result := make([]model.MoneyAccountWithBalance, len(accounts))
	for i, acct := range accounts {
		balance, err := s.accountRepo.GetAccountBalance(acct.ID)
		if err != nil {
			return nil, err
		}
		result[i] = model.MoneyAccountWithBalance{
			MoneyAccount: *acct,
			BalanceCents: balance,
		}
	}

	return result, nil
}

func (s *MoneyAccountService) GetAccount(id string) (*model.MoneyAccount, error) {
	return s.accountRepo.GetByID(id)
}

func (s *MoneyAccountService) UpdateAccount(dto UpdateMoneyAccountDTO) (*model.MoneyAccount, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, fmt.Errorf("account name cannot be empty")
	}

	account, err := s.accountRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	account.Name = name

	err = s.accountRepo.Update(account)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *MoneyAccountService) DeleteAccount(id string) error {
	return s.accountRepo.Delete(id)
}

func (s *MoneyAccountService) CreateTransfer(dto CreateTransferDTO, availableSpaceBalance int) (*model.AccountTransfer, error) {
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	if dto.Direction != model.TransferDirectionDeposit && dto.Direction != model.TransferDirectionWithdrawal {
		return nil, fmt.Errorf("invalid transfer direction")
	}

	if dto.Direction == model.TransferDirectionDeposit {
		if dto.Amount > availableSpaceBalance {
			return nil, fmt.Errorf("insufficient available balance")
		}
	}

	if dto.Direction == model.TransferDirectionWithdrawal {
		accountBalance, err := s.accountRepo.GetAccountBalance(dto.AccountID)
		if err != nil {
			return nil, err
		}
		if dto.Amount > accountBalance {
			return nil, fmt.Errorf("insufficient account balance")
		}
	}

	transfer := &model.AccountTransfer{
		ID:          uuid.NewString(),
		AccountID:   dto.AccountID,
		AmountCents: dto.Amount,
		Direction:   dto.Direction,
		Note:        strings.TrimSpace(dto.Note),
		CreatedBy:   dto.CreatedBy,
		CreatedAt:   time.Now(),
	}

	err := s.accountRepo.CreateTransfer(transfer)
	if err != nil {
		return nil, err
	}

	return transfer, nil
}

func (s *MoneyAccountService) GetTransfersForAccount(accountID string) ([]*model.AccountTransfer, error) {
	return s.accountRepo.GetTransfersByAccountID(accountID)
}

func (s *MoneyAccountService) DeleteTransfer(id string) error {
	return s.accountRepo.DeleteTransfer(id)
}

func (s *MoneyAccountService) GetAccountBalance(accountID string) (int, error) {
	return s.accountRepo.GetAccountBalance(accountID)
}

func (s *MoneyAccountService) GetTotalAllocatedForSpace(spaceID string) (int, error) {
	return s.accountRepo.GetTotalAllocatedForSpace(spaceID)
}

const TransfersPerPage = 25

func (s *MoneyAccountService) GetTransfersForSpacePaginated(spaceID string, page int) ([]*model.AccountTransferWithAccount, int, error) {
	total, err := s.accountRepo.CountTransfersBySpaceID(spaceID)
	if err != nil {
		return nil, 0, err
	}

	totalPages := (total + TransfersPerPage - 1) / TransfersPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * TransfersPerPage
	transfers, err := s.accountRepo.GetTransfersBySpaceIDPaginated(spaceID, TransfersPerPage, offset)
	if err != nil {
		return nil, 0, err
	}

	return transfers, totalPages, nil
}
