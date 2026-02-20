package service

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreateRecurringDepositDTO struct {
	SpaceID   string
	AccountID string
	Amount    int
	Frequency model.Frequency
	StartDate time.Time
	EndDate   *time.Time
	Title     string
	CreatedBy string
}

type UpdateRecurringDepositDTO struct {
	ID        string
	AccountID string
	Amount    int
	Frequency model.Frequency
	StartDate time.Time
	EndDate   *time.Time
	Title     string
}

type RecurringDepositService struct {
	recurringRepo  repository.RecurringDepositRepository
	accountRepo    repository.MoneyAccountRepository
	expenseService *ExpenseService
}

func NewRecurringDepositService(recurringRepo repository.RecurringDepositRepository, accountRepo repository.MoneyAccountRepository, expenseService *ExpenseService) *RecurringDepositService {
	return &RecurringDepositService{
		recurringRepo:  recurringRepo,
		accountRepo:    accountRepo,
		expenseService: expenseService,
	}
}

func (s *RecurringDepositService) CreateRecurringDeposit(dto CreateRecurringDepositDTO) (*model.RecurringDeposit, error) {
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	now := time.Now()
	rd := &model.RecurringDeposit{
		ID:             uuid.NewString(),
		SpaceID:        dto.SpaceID,
		AccountID:      dto.AccountID,
		AmountCents:    dto.Amount,
		Frequency:      dto.Frequency,
		StartDate:      dto.StartDate,
		EndDate:        dto.EndDate,
		NextOccurrence: dto.StartDate,
		IsActive:       true,
		Title:          strings.TrimSpace(dto.Title),
		CreatedBy:      dto.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.recurringRepo.Create(rd); err != nil {
		return nil, err
	}
	return rd, nil
}

func (s *RecurringDepositService) GetRecurringDeposit(id string) (*model.RecurringDeposit, error) {
	return s.recurringRepo.GetByID(id)
}

func (s *RecurringDepositService) GetRecurringDepositsForSpace(spaceID string) ([]*model.RecurringDeposit, error) {
	return s.recurringRepo.GetBySpaceID(spaceID)
}

func (s *RecurringDepositService) GetRecurringDepositsWithAccountsForSpace(spaceID string) ([]*model.RecurringDepositWithAccount, error) {
	deposits, err := s.recurringRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	accounts, err := s.accountRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	accountNames := make(map[string]string, len(accounts))
	for _, acct := range accounts {
		accountNames[acct.ID] = acct.Name
	}

	result := make([]*model.RecurringDepositWithAccount, len(deposits))
	for i, rd := range deposits {
		result[i] = &model.RecurringDepositWithAccount{
			RecurringDeposit: *rd,
			AccountName:      accountNames[rd.AccountID],
		}
	}
	return result, nil
}

func (s *RecurringDepositService) UpdateRecurringDeposit(dto UpdateRecurringDepositDTO) (*model.RecurringDeposit, error) {
	if dto.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	existing, err := s.recurringRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	existing.AccountID = dto.AccountID
	existing.AmountCents = dto.Amount
	existing.Frequency = dto.Frequency
	existing.StartDate = dto.StartDate
	existing.EndDate = dto.EndDate
	existing.Title = strings.TrimSpace(dto.Title)
	existing.UpdatedAt = time.Now()

	// Recalculate next occurrence if start date moved forward
	if existing.NextOccurrence.Before(dto.StartDate) {
		existing.NextOccurrence = dto.StartDate
	}

	if err := s.recurringRepo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *RecurringDepositService) DeleteRecurringDeposit(id string) error {
	return s.recurringRepo.Delete(id)
}

func (s *RecurringDepositService) ToggleRecurringDeposit(id string) (*model.RecurringDeposit, error) {
	rd, err := s.recurringRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	newActive := !rd.IsActive
	if err := s.recurringRepo.SetActive(id, newActive); err != nil {
		return nil, err
	}
	rd.IsActive = newActive
	return rd, nil
}

func (s *RecurringDepositService) ProcessDueRecurrences(now time.Time) error {
	dues, err := s.recurringRepo.GetDueRecurrences(now)
	if err != nil {
		return fmt.Errorf("failed to get due recurring deposits: %w", err)
	}

	for _, rd := range dues {
		if err := s.processRecurrence(rd, now); err != nil {
			slog.Error("failed to process recurring deposit", "id", rd.ID, "error", err)
		}
	}
	return nil
}

func (s *RecurringDepositService) ProcessDueRecurrencesForSpace(spaceID string, now time.Time) error {
	dues, err := s.recurringRepo.GetDueRecurrencesForSpace(spaceID, now)
	if err != nil {
		return fmt.Errorf("failed to get due recurring deposits for space: %w", err)
	}

	for _, rd := range dues {
		if err := s.processRecurrence(rd, now); err != nil {
			slog.Error("failed to process recurring deposit", "id", rd.ID, "error", err)
		}
	}
	return nil
}

func (s *RecurringDepositService) getAvailableBalance(spaceID string) (int, error) {
	totalBalance, err := s.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get space balance: %w", err)
	}
	totalAllocated, err := s.accountRepo.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get total allocated: %w", err)
	}
	return totalBalance - totalAllocated, nil
}

func (s *RecurringDepositService) processRecurrence(rd *model.RecurringDeposit, now time.Time) error {
	for !rd.NextOccurrence.After(now) {
		// Check if end_date has been passed
		if rd.EndDate != nil && rd.NextOccurrence.After(*rd.EndDate) {
			return s.recurringRepo.Deactivate(rd.ID)
		}

		// Check available balance
		availableBalance, err := s.getAvailableBalance(rd.SpaceID)
		if err != nil {
			return err
		}

		if availableBalance >= rd.AmountCents {
			rdID := rd.ID
			transfer := &model.AccountTransfer{
				ID:                 uuid.NewString(),
				AccountID:          rd.AccountID,
				AmountCents:        rd.AmountCents,
				Direction:          model.TransferDirectionDeposit,
				Note:               rd.Title,
				RecurringDepositID: &rdID,
				CreatedBy:          rd.CreatedBy,
				CreatedAt:          time.Now(),
			}
			if err := s.accountRepo.CreateTransfer(transfer); err != nil {
				return fmt.Errorf("failed to create deposit transfer: %w", err)
			}
		} else {
			slog.Warn("recurring deposit skipped: insufficient available balance",
				"recurring_deposit_id", rd.ID,
				"space_id", rd.SpaceID,
				"needed", rd.AmountCents,
				"available", availableBalance,
			)
		}

		rd.NextOccurrence = AdvanceDate(rd.NextOccurrence, rd.Frequency)
	}

	// Check if the new next occurrence exceeds end date
	if rd.EndDate != nil && rd.NextOccurrence.After(*rd.EndDate) {
		if err := s.recurringRepo.Deactivate(rd.ID); err != nil {
			return err
		}
	}

	return s.recurringRepo.UpdateNextOccurrence(rd.ID, rd.NextOccurrence)
}
