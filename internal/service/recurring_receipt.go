package service

import (
	"fmt"
	"log/slog"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateRecurringReceiptDTO struct {
	LoanID         string
	SpaceID        string
	UserID         string
	Description    string
	TotalAmount    decimal.Decimal
	Frequency      model.Frequency
	StartDate      time.Time
	EndDate        *time.Time
	FundingSources []FundingSourceDTO
}

type UpdateRecurringReceiptDTO struct {
	ID             string
	Description    string
	TotalAmount    decimal.Decimal
	Frequency      model.Frequency
	StartDate      time.Time
	EndDate        *time.Time
	FundingSources []FundingSourceDTO
}

type RecurringReceiptService struct {
	recurringRepo  repository.RecurringReceiptRepository
	receiptService *ReceiptService
	loanRepo       repository.LoanRepository
	profileRepo    repository.ProfileRepository
	spaceRepo      repository.SpaceRepository
}

func NewRecurringReceiptService(
	recurringRepo repository.RecurringReceiptRepository,
	receiptService *ReceiptService,
	loanRepo repository.LoanRepository,
	profileRepo repository.ProfileRepository,
	spaceRepo repository.SpaceRepository,
) *RecurringReceiptService {
	return &RecurringReceiptService{
		recurringRepo:  recurringRepo,
		receiptService: receiptService,
		loanRepo:       loanRepo,
		profileRepo:    profileRepo,
		spaceRepo:      spaceRepo,
	}
}

func (s *RecurringReceiptService) CreateRecurringReceipt(dto CreateRecurringReceiptDTO) (*model.RecurringReceiptWithSources, error) {
	if dto.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive")
	}
	if len(dto.FundingSources) == 0 {
		return nil, fmt.Errorf("at least one funding source is required")
	}

	sum := decimal.Zero
	for _, src := range dto.FundingSources {
		sum = sum.Add(src.Amount)
	}
	if !sum.Equal(dto.TotalAmount) {
		return nil, fmt.Errorf("funding source amounts must equal total amount")
	}

	now := time.Now()
	rr := &model.RecurringReceipt{
		ID:             uuid.NewString(),
		LoanID:         dto.LoanID,
		SpaceID:        dto.SpaceID,
		Description:    dto.Description,
		TotalAmount:    dto.TotalAmount,
		Frequency:      dto.Frequency,
		StartDate:      dto.StartDate,
		EndDate:        dto.EndDate,
		NextOccurrence: dto.StartDate,
		IsActive:       true,
		CreatedBy:      dto.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	sources := make([]model.RecurringReceiptSource, len(dto.FundingSources))
	for i, src := range dto.FundingSources {
		sources[i] = model.RecurringReceiptSource{
			ID:                 uuid.NewString(),
			RecurringReceiptID: rr.ID,
			SourceType:         src.SourceType,
			Amount:             src.Amount,
		}
		if src.SourceType == model.FundingSourceAccount {
			acctID := src.AccountID
			sources[i].AccountID = &acctID
		}
	}

	if err := s.recurringRepo.Create(rr, sources); err != nil {
		return nil, err
	}

	return &model.RecurringReceiptWithSources{
		RecurringReceipt: *rr,
		Sources:          sources,
	}, nil
}

func (s *RecurringReceiptService) GetRecurringReceipt(id string) (*model.RecurringReceipt, error) {
	return s.recurringRepo.GetByID(id)
}

func (s *RecurringReceiptService) GetRecurringReceiptsForLoan(loanID string) ([]*model.RecurringReceipt, error) {
	return s.recurringRepo.GetByLoanID(loanID)
}

func (s *RecurringReceiptService) GetRecurringReceiptsWithSourcesForLoan(loanID string) ([]*model.RecurringReceiptWithSources, error) {
	rrs, err := s.recurringRepo.GetByLoanID(loanID)
	if err != nil {
		return nil, err
	}

	result := make([]*model.RecurringReceiptWithSources, len(rrs))
	for i, rr := range rrs {
		sources, err := s.recurringRepo.GetSourcesByRecurringReceiptID(rr.ID)
		if err != nil {
			return nil, err
		}
		result[i] = &model.RecurringReceiptWithSources{
			RecurringReceipt: *rr,
			Sources:          sources,
		}
	}
	return result, nil
}

func (s *RecurringReceiptService) UpdateRecurringReceipt(dto UpdateRecurringReceiptDTO) (*model.RecurringReceipt, error) {
	if dto.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive")
	}

	existing, err := s.recurringRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	existing.Description = dto.Description
	existing.TotalAmount = dto.TotalAmount
	existing.Frequency = dto.Frequency
	existing.StartDate = dto.StartDate
	existing.EndDate = dto.EndDate
	existing.UpdatedAt = time.Now()

	if existing.NextOccurrence.Before(dto.StartDate) {
		existing.NextOccurrence = dto.StartDate
	}

	sources := make([]model.RecurringReceiptSource, len(dto.FundingSources))
	for i, src := range dto.FundingSources {
		sources[i] = model.RecurringReceiptSource{
			ID:                 uuid.NewString(),
			RecurringReceiptID: existing.ID,
			SourceType:         src.SourceType,
			Amount:             src.Amount,
		}
		if src.SourceType == model.FundingSourceAccount {
			acctID := src.AccountID
			sources[i].AccountID = &acctID
		}
	}

	if err := s.recurringRepo.Update(existing, sources); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *RecurringReceiptService) DeleteRecurringReceipt(id string) error {
	return s.recurringRepo.Delete(id)
}

func (s *RecurringReceiptService) ToggleRecurringReceipt(id string) (*model.RecurringReceipt, error) {
	rr, err := s.recurringRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	newActive := !rr.IsActive
	if err := s.recurringRepo.SetActive(id, newActive); err != nil {
		return nil, err
	}
	rr.IsActive = newActive
	return rr, nil
}

func (s *RecurringReceiptService) ProcessDueRecurrences(now time.Time) error {
	dues, err := s.recurringRepo.GetDueRecurrences(now)
	if err != nil {
		return fmt.Errorf("failed to get due recurring receipts: %w", err)
	}

	tzCache := make(map[string]*time.Location)
	for _, rr := range dues {
		localNow := s.getLocalNow(rr.SpaceID, rr.CreatedBy, now, tzCache)
		if err := s.processRecurrence(rr, localNow); err != nil {
			slog.Error("failed to process recurring receipt", "id", rr.ID, "error", err)
		}
	}
	return nil
}

func (s *RecurringReceiptService) ProcessDueRecurrencesForSpace(spaceID string, now time.Time) error {
	dues, err := s.recurringRepo.GetDueRecurrencesForSpace(spaceID, now)
	if err != nil {
		return fmt.Errorf("failed to get due recurring receipts for space: %w", err)
	}

	tzCache := make(map[string]*time.Location)
	for _, rr := range dues {
		localNow := s.getLocalNow(rr.SpaceID, rr.CreatedBy, now, tzCache)
		if err := s.processRecurrence(rr, localNow); err != nil {
			slog.Error("failed to process recurring receipt", "id", rr.ID, "error", err)
		}
	}
	return nil
}

func (s *RecurringReceiptService) processRecurrence(rr *model.RecurringReceipt, now time.Time) error {
	sources, err := s.recurringRepo.GetSourcesByRecurringReceiptID(rr.ID)
	if err != nil {
		return err
	}

	for !rr.NextOccurrence.After(now) {
		if rr.EndDate != nil && rr.NextOccurrence.After(*rr.EndDate) {
			return s.recurringRepo.Deactivate(rr.ID)
		}

		// Check if loan is already paid off
		loan, err := s.loanRepo.GetByID(rr.LoanID)
		if err != nil {
			return fmt.Errorf("failed to get loan: %w", err)
		}
		if loan.IsPaidOff {
			return s.recurringRepo.Deactivate(rr.ID)
		}

		// Build funding source DTOs from template
		fundingSources := make([]FundingSourceDTO, len(sources))
		for i, src := range sources {
			accountID := ""
			if src.AccountID != nil {
				accountID = *src.AccountID
			}
			fundingSources[i] = FundingSourceDTO{
				SourceType: src.SourceType,
				AccountID:  accountID,
				Amount:     src.Amount,
			}
		}

		rrID := rr.ID
		dto := CreateReceiptDTO{
			LoanID:             rr.LoanID,
			SpaceID:            rr.SpaceID,
			UserID:             rr.CreatedBy,
			Description:        rr.Description,
			TotalAmount:        rr.TotalAmount,
			Date:               rr.NextOccurrence,
			FundingSources:     fundingSources,
			RecurringReceiptID: &rrID,
		}

		if _, err := s.receiptService.CreateReceipt(dto); err != nil {
			slog.Warn("recurring receipt skipped", "id", rr.ID, "error", err)
		}

		rr.NextOccurrence = AdvanceDate(rr.NextOccurrence, rr.Frequency)
	}

	if rr.EndDate != nil && rr.NextOccurrence.After(*rr.EndDate) {
		if err := s.recurringRepo.Deactivate(rr.ID); err != nil {
			return err
		}
	}

	return s.recurringRepo.UpdateNextOccurrence(rr.ID, rr.NextOccurrence)
}

func (s *RecurringReceiptService) getLocalNow(spaceID, userID string, now time.Time, cache map[string]*time.Location) time.Time {
	spaceKey := "space:" + spaceID
	if loc, ok := cache[spaceKey]; ok {
		return now.In(loc)
	}

	space, err := s.spaceRepo.ByID(spaceID)
	if err == nil && space != nil {
		if loc := space.Location(); loc != nil {
			cache[spaceKey] = loc
			return now.In(loc)
		}
	}

	userKey := "user:" + userID
	if loc, ok := cache[userKey]; ok {
		return now.In(loc)
	}

	loc := time.UTC
	profile, err := s.profileRepo.ByUserID(userID)
	if err == nil && profile != nil {
		loc = profile.Location()
	}
	cache[userKey] = loc
	return now.In(loc)
}
