package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type FundingSourceDTO struct {
	SourceType model.FundingSourceType
	AccountID  string
	Amount     decimal.Decimal
}

type CreateReceiptDTO struct {
	LoanID             string
	SpaceID            string
	UserID             string
	Description        string
	TotalAmount        decimal.Decimal
	Date               time.Time
	FundingSources     []FundingSourceDTO
	RecurringReceiptID *string
}

type UpdateReceiptDTO struct {
	ID             string
	SpaceID        string
	UserID         string
	Description    string
	TotalAmount    decimal.Decimal
	Date           time.Time
	FundingSources []FundingSourceDTO
}

const ReceiptsPerPage = 25

type ReceiptService struct {
	receiptRepo repository.ReceiptRepository
	loanRepo    repository.LoanRepository
	accountRepo repository.MoneyAccountRepository
}

func NewReceiptService(
	receiptRepo repository.ReceiptRepository,
	loanRepo repository.LoanRepository,
	accountRepo repository.MoneyAccountRepository,
) *ReceiptService {
	return &ReceiptService{
		receiptRepo: receiptRepo,
		loanRepo:    loanRepo,
		accountRepo: accountRepo,
	}
}

func (s *ReceiptService) CreateReceipt(dto CreateReceiptDTO) (*model.ReceiptWithSources, error) {
	if dto.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive")
	}
	if len(dto.FundingSources) == 0 {
		return nil, fmt.Errorf("at least one funding source is required")
	}

	// Validate funding sources sum to total
	sum := decimal.Zero
	for _, src := range dto.FundingSources {
		if src.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("each funding source amount must be positive")
		}
		sum = sum.Add(src.Amount)
	}
	if !sum.Equal(dto.TotalAmount) {
		return nil, fmt.Errorf("funding source amounts (%s) must equal total amount (%s)", sum, dto.TotalAmount)
	}

	// Validate loan exists and is not paid off
	loan, err := s.loanRepo.GetByID(dto.LoanID)
	if err != nil {
		return nil, fmt.Errorf("loan not found: %w", err)
	}
	if loan.IsPaidOff {
		return nil, fmt.Errorf("loan is already paid off")
	}

	now := time.Now()
	receipt := &model.Receipt{
		ID:                 uuid.NewString(),
		LoanID:             dto.LoanID,
		SpaceID:            dto.SpaceID,
		Description:        dto.Description,
		TotalAmount:        dto.TotalAmount,
		Date:               dto.Date,
		RecurringReceiptID: dto.RecurringReceiptID,
		CreatedBy:          dto.UserID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	sources, balanceExpense, accountTransfers := s.buildLinkedRecords(receipt, dto.FundingSources, dto.SpaceID, dto.UserID, dto.Description, dto.Date)

	if err := s.receiptRepo.CreateWithSources(receipt, sources, balanceExpense, accountTransfers); err != nil {
		return nil, err
	}

	// Check if loan is now fully paid off
	totalPaid, err := s.loanRepo.GetTotalPaidForLoan(dto.LoanID)
	if err == nil && totalPaid.GreaterThanOrEqual(loan.OriginalAmount) {
		_ = s.loanRepo.SetPaidOff(loan.ID, true)
	}

	return &model.ReceiptWithSources{Receipt: *receipt, Sources: sources}, nil
}

func (s *ReceiptService) buildLinkedRecords(
	receipt *model.Receipt,
	fundingSources []FundingSourceDTO,
	spaceID, userID, description string,
	date time.Time,
) ([]model.ReceiptFundingSource, *model.Expense, []*model.AccountTransfer) {
	now := time.Now()
	var sources []model.ReceiptFundingSource
	var balanceExpense *model.Expense
	var accountTransfers []*model.AccountTransfer

	for _, src := range fundingSources {
		fs := model.ReceiptFundingSource{
			ID:         uuid.NewString(),
			ReceiptID:  receipt.ID,
			SourceType: src.SourceType,
			Amount:     src.Amount,
		}

		if src.SourceType == model.FundingSourceBalance {
			expense := &model.Expense{
				ID:          uuid.NewString(),
				SpaceID:     spaceID,
				CreatedBy:   userID,
				Description: fmt.Sprintf("Loan payment: %s", description),
				Amount:      src.Amount,
				Type:        model.ExpenseTypeExpense,
				Date:        date,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			balanceExpense = expense
			fs.LinkedExpenseID = &expense.ID
		} else {
			acctID := src.AccountID
			fs.AccountID = &acctID
			transfer := &model.AccountTransfer{
				ID:        uuid.NewString(),
				AccountID: src.AccountID,
				Amount:    src.Amount,
				Direction: model.TransferDirectionWithdrawal,
				Note:      fmt.Sprintf("Loan payment: %s", description),
				CreatedBy: userID,
				CreatedAt: now,
			}
			accountTransfers = append(accountTransfers, transfer)
			fs.LinkedTransferID = &transfer.ID
		}

		sources = append(sources, fs)
	}

	return sources, balanceExpense, accountTransfers
}

func (s *ReceiptService) GetReceipt(id string) (*model.ReceiptWithSourcesAndAccounts, error) {
	receipt, err := s.receiptRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	sourcesMap, err := s.receiptRepo.GetFundingSourcesWithAccountsByReceiptIDs([]string{id})
	if err != nil {
		return nil, err
	}

	return &model.ReceiptWithSourcesAndAccounts{
		Receipt: *receipt,
		Sources: sourcesMap[id],
	}, nil
}

func (s *ReceiptService) GetReceiptsForLoanPaginated(loanID string, page int) ([]*model.ReceiptWithSourcesAndAccounts, int, error) {
	total, err := s.receiptRepo.CountByLoanID(loanID)
	if err != nil {
		return nil, 0, err
	}

	totalPages := (total + ReceiptsPerPage - 1) / ReceiptsPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * ReceiptsPerPage
	receipts, err := s.receiptRepo.GetByLoanIDPaginated(loanID, ReceiptsPerPage, offset)
	if err != nil {
		return nil, 0, err
	}

	return s.attachSources(receipts, totalPages)
}

func (s *ReceiptService) attachSources(receipts []*model.Receipt, totalPages int) ([]*model.ReceiptWithSourcesAndAccounts, int, error) {
	ids := make([]string, len(receipts))
	for i, r := range receipts {
		ids[i] = r.ID
	}

	sourcesMap, err := s.receiptRepo.GetFundingSourcesWithAccountsByReceiptIDs(ids)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*model.ReceiptWithSourcesAndAccounts, len(receipts))
	for i, r := range receipts {
		result[i] = &model.ReceiptWithSourcesAndAccounts{
			Receipt: *r,
			Sources: sourcesMap[r.ID],
		}
	}
	return result, totalPages, nil
}

func (s *ReceiptService) DeleteReceipt(id string, spaceID string) error {
	receipt, err := s.receiptRepo.GetByID(id)
	if err != nil {
		return err
	}
	if receipt.SpaceID != spaceID {
		return fmt.Errorf("receipt not found")
	}

	if err := s.receiptRepo.DeleteWithReversal(id); err != nil {
		return err
	}

	// Check if loan should be un-marked as paid off
	totalPaid, err := s.loanRepo.GetTotalPaidForLoan(receipt.LoanID)
	if err != nil {
		return nil // receipt deleted successfully, paid-off check is best-effort
	}
	loan, err := s.loanRepo.GetByID(receipt.LoanID)
	if err != nil {
		return nil
	}
	if loan.IsPaidOff && totalPaid.LessThan(loan.OriginalAmount) {
		_ = s.loanRepo.SetPaidOff(loan.ID, false)
	}

	return nil
}

func (s *ReceiptService) UpdateReceipt(dto UpdateReceiptDTO) (*model.ReceiptWithSources, error) {
	if dto.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive")
	}
	if len(dto.FundingSources) == 0 {
		return nil, fmt.Errorf("at least one funding source is required")
	}

	sum := decimal.Zero
	for _, src := range dto.FundingSources {
		if src.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("each funding source amount must be positive")
		}
		sum = sum.Add(src.Amount)
	}
	if !sum.Equal(dto.TotalAmount) {
		return nil, fmt.Errorf("funding source amounts (%s) must equal total amount (%s)", sum, dto.TotalAmount)
	}

	existing, err := s.receiptRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}
	if existing.SpaceID != dto.SpaceID {
		return nil, fmt.Errorf("receipt not found")
	}

	existing.Description = dto.Description
	existing.TotalAmount = dto.TotalAmount
	existing.Date = dto.Date
	existing.UpdatedAt = time.Now()

	sources, balanceExpense, accountTransfers := s.buildLinkedRecords(existing, dto.FundingSources, dto.SpaceID, dto.UserID, dto.Description, dto.Date)

	if err := s.receiptRepo.UpdateWithSources(existing, sources, balanceExpense, accountTransfers); err != nil {
		return nil, err
	}

	// Re-check paid-off status
	loan, err := s.loanRepo.GetByID(existing.LoanID)
	if err == nil {
		totalPaid, err := s.loanRepo.GetTotalPaidForLoan(existing.LoanID)
		if err == nil {
			if totalPaid.GreaterThanOrEqual(loan.OriginalAmount) && !loan.IsPaidOff {
				_ = s.loanRepo.SetPaidOff(loan.ID, true)
			} else if totalPaid.LessThan(loan.OriginalAmount) && loan.IsPaidOff {
				_ = s.loanRepo.SetPaidOff(loan.ID, false)
			}
		}
	}

	return &model.ReceiptWithSources{Receipt: *existing, Sources: sources}, nil
}
