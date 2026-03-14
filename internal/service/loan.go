package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreateLoanDTO struct {
	SpaceID         string
	UserID          string
	Name            string
	Description     string
	OriginalAmount  int
	InterestRateBps int
	StartDate       time.Time
	EndDate         *time.Time
}

type UpdateLoanDTO struct {
	ID              string
	Name            string
	Description     string
	OriginalAmount  int
	InterestRateBps int
	StartDate       time.Time
	EndDate         *time.Time
}

const LoansPerPage = 25

type LoanService struct {
	loanRepo    repository.LoanRepository
	receiptRepo repository.ReceiptRepository
}

func NewLoanService(loanRepo repository.LoanRepository, receiptRepo repository.ReceiptRepository) *LoanService {
	return &LoanService{
		loanRepo:    loanRepo,
		receiptRepo: receiptRepo,
	}
}

func (s *LoanService) CreateLoan(dto CreateLoanDTO) (*model.Loan, error) {
	if dto.Name == "" {
		return nil, fmt.Errorf("loan name cannot be empty")
	}
	if dto.OriginalAmount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	now := time.Now()
	loan := &model.Loan{
		ID:                  uuid.NewString(),
		SpaceID:             dto.SpaceID,
		Name:                dto.Name,
		Description:         dto.Description,
		OriginalAmountCents: dto.OriginalAmount,
		InterestRateBps:     dto.InterestRateBps,
		StartDate:           dto.StartDate,
		EndDate:             dto.EndDate,
		IsPaidOff:           false,
		CreatedBy:           dto.UserID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.loanRepo.Create(loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *LoanService) GetLoan(id string) (*model.Loan, error) {
	return s.loanRepo.GetByID(id)
}

func (s *LoanService) GetLoanWithSummary(id string) (*model.LoanWithPaymentSummary, error) {
	loan, err := s.loanRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	totalPaid, err := s.loanRepo.GetTotalPaidForLoan(id)
	if err != nil {
		return nil, err
	}

	receiptCount, err := s.loanRepo.GetReceiptCountForLoan(id)
	if err != nil {
		return nil, err
	}

	return &model.LoanWithPaymentSummary{
		Loan:           *loan,
		TotalPaidCents: totalPaid,
		RemainingCents: loan.OriginalAmountCents - totalPaid,
		ReceiptCount:   receiptCount,
	}, nil
}

func (s *LoanService) GetLoansWithSummaryForSpace(spaceID string) ([]*model.LoanWithPaymentSummary, error) {
	loans, err := s.loanRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	return s.attachSummaries(loans)
}

func (s *LoanService) GetLoansWithSummaryForSpacePaginated(spaceID string, page int) ([]*model.LoanWithPaymentSummary, int, error) {
	total, err := s.loanRepo.CountBySpaceID(spaceID)
	if err != nil {
		return nil, 0, err
	}

	totalPages := (total + LoansPerPage - 1) / LoansPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * LoansPerPage
	loans, err := s.loanRepo.GetBySpaceIDPaginated(spaceID, LoansPerPage, offset)
	if err != nil {
		return nil, 0, err
	}

	result, err := s.attachSummaries(loans)
	if err != nil {
		return nil, 0, err
	}

	return result, totalPages, nil
}

func (s *LoanService) attachSummaries(loans []*model.Loan) ([]*model.LoanWithPaymentSummary, error) {
	result := make([]*model.LoanWithPaymentSummary, len(loans))
	for i, loan := range loans {
		totalPaid, err := s.loanRepo.GetTotalPaidForLoan(loan.ID)
		if err != nil {
			return nil, err
		}
		receiptCount, err := s.loanRepo.GetReceiptCountForLoan(loan.ID)
		if err != nil {
			return nil, err
		}
		result[i] = &model.LoanWithPaymentSummary{
			Loan:           *loan,
			TotalPaidCents: totalPaid,
			RemainingCents: loan.OriginalAmountCents - totalPaid,
			ReceiptCount:   receiptCount,
		}
	}
	return result, nil
}

func (s *LoanService) UpdateLoan(dto UpdateLoanDTO) (*model.Loan, error) {
	if dto.Name == "" {
		return nil, fmt.Errorf("loan name cannot be empty")
	}
	if dto.OriginalAmount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	existing, err := s.loanRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	existing.Name = dto.Name
	existing.Description = dto.Description
	existing.OriginalAmountCents = dto.OriginalAmount
	existing.InterestRateBps = dto.InterestRateBps
	existing.StartDate = dto.StartDate
	existing.EndDate = dto.EndDate
	existing.UpdatedAt = time.Now()

	if err := s.loanRepo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *LoanService) DeleteLoan(id string) error {
	return s.loanRepo.Delete(id)
}
