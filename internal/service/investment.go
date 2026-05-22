package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InvestmentService handles contribution rooms, holdings, trades, and the
// summary view for investment-flagged accounts. Cash movement (contributions
// and withdrawals) still goes through TransactionService / TransferService;
// this service reads from those tables but never mutates them directly.
type InvestmentService struct {
	accountRepo repository.AccountRepository
	roomRepo    repository.InvestmentContributionRoomRepository
	holdingRepo repository.InvestmentHoldingRepository
	tradeRepo   repository.InvestmentTradeRepository
	txRepo      repository.TransactionRepository
}

func NewInvestmentService(
	accountRepo repository.AccountRepository,
	roomRepo repository.InvestmentContributionRoomRepository,
	holdingRepo repository.InvestmentHoldingRepository,
	tradeRepo repository.InvestmentTradeRepository,
	txRepo repository.TransactionRepository,
) *InvestmentService {
	return &InvestmentService{
		accountRepo: accountRepo,
		roomRepo:    roomRepo,
		holdingRepo: holdingRepo,
		tradeRepo:   tradeRepo,
		txRepo:      txRepo,
	}
}

// ---------- Contribution room ----------

func (s *InvestmentService) SetContributionRoom(accountID string, year int, room decimal.Decimal) error {
	if accountID == "" {
		return fmt.Errorf("account id is required")
	}
	if year < 1900 || year > 9999 {
		return fmt.Errorf("year out of range")
	}
	if room.IsNegative() {
		return fmt.Errorf("contribution room cannot be negative")
	}
	account, err := s.accountRepo.ByID(accountID)
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}
	if !account.IsInvestment {
		return fmt.Errorf("account is not an investment account")
	}
	now := time.Now()
	return s.roomRepo.Upsert(&model.InvestmentContributionRoom{
		AccountID:  accountID,
		Year:       year,
		RoomAmount: room,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
}

func (s *InvestmentService) GetContributionRoom(accountID string, year int) (*model.InvestmentContributionRoom, error) {
	room, err := s.roomRepo.ByAccountAndYear(accountID, year)
	if err != nil {
		if err == repository.ErrContributionRoomNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load contribution room: %w", err)
	}
	return room, nil
}

func (s *InvestmentService) ListContributionRooms(accountID string) ([]*model.InvestmentContributionRoom, error) {
	rooms, err := s.roomRepo.ByAccountID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list contribution rooms: %w", err)
	}
	return rooms, nil
}

// ---------- Summary ----------

// SummarizeAccount produces the rollup view for an investment account in the
// given calendar year: contribution room, YTD cash flow, lifetime net
// contributions, and total cost basis across all holdings.
func (s *InvestmentService) SummarizeAccount(accountID string, year int) (*model.InvestmentAccountSummary, error) {
	account, err := s.accountRepo.ByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}
	if !account.IsInvestment {
		return nil, fmt.Errorf("account is not an investment account")
	}

	ytdContrib, err := s.txRepo.SumByAccountYearType(accountID, year, model.TransactionTypeDeposit)
	if err != nil {
		return nil, fmt.Errorf("failed to sum ytd contributions: %w", err)
	}
	ytdWithdraw, err := s.txRepo.SumByAccountYearType(accountID, year, model.TransactionTypeWithdrawal)
	if err != nil {
		return nil, fmt.Errorf("failed to sum ytd withdrawals: %w", err)
	}
	lifeContrib, err := s.txRepo.SumLifetimeByAccountType(accountID, model.TransactionTypeDeposit)
	if err != nil {
		return nil, fmt.Errorf("failed to sum lifetime contributions: %w", err)
	}
	lifeWithdraw, err := s.txRepo.SumLifetimeByAccountType(accountID, model.TransactionTypeWithdrawal)
	if err != nil {
		return nil, fmt.Errorf("failed to sum lifetime withdrawals: %w", err)
	}

	summary := &model.InvestmentAccountSummary{
		Account:          account,
		Year:             year,
		YTDContributions: ytdContrib,
		YTDWithdrawals:   ytdWithdraw,
		NetContributions: lifeContrib.Sub(lifeWithdraw),
	}

	room, err := s.roomRepo.ByAccountAndYear(accountID, year)
	if err == nil {
		amt := room.RoomAmount
		summary.RoomAmount = &amt
		rem := amt.Sub(ytdContrib)
		summary.RoomRemaining = &rem
	} else if err != repository.ErrContributionRoomNotFound {
		return nil, fmt.Errorf("failed to load contribution room: %w", err)
	}

	positions, err := s.HoldingPositions(accountID)
	if err != nil {
		return nil, err
	}
	summary.HoldingCount = len(positions)
	for _, p := range positions {
		summary.TotalCostBasis = summary.TotalCostBasis.Add(p.CostBasis)
	}
	return summary, nil
}

// ---------- Holdings ----------

func (s *InvestmentService) CreateHolding(accountID, symbol, displayName string) (*model.InvestmentHolding, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account id is required")
	}
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if displayName == "" {
		displayName = symbol
	}
	account, err := s.accountRepo.ByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}
	if !account.IsInvestment {
		return nil, fmt.Errorf("account is not an investment account")
	}
	now := time.Now()
	holding := &model.InvestmentHolding{
		ID:          uuid.NewString(),
		AccountID:   accountID,
		Symbol:      symbol,
		DisplayName: displayName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.holdingRepo.Create(holding); err != nil {
		return nil, fmt.Errorf("failed to create holding: %w", err)
	}
	return holding, nil
}

func (s *InvestmentService) GetHolding(id string) (*model.InvestmentHolding, error) {
	h, err := s.holdingRepo.ByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load holding: %w", err)
	}
	return h, nil
}

func (s *InvestmentService) UpdateHolding(id, symbol, displayName string) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if displayName == "" {
		displayName = symbol
	}
	if err := s.holdingRepo.Update(id, symbol, displayName); err != nil {
		return fmt.Errorf("failed to update holding: %w", err)
	}
	return nil
}

func (s *InvestmentService) DeleteHolding(id string) error {
	return s.holdingRepo.Delete(id)
}

func (s *InvestmentService) ListHoldings(accountID string) ([]*model.InvestmentHolding, error) {
	return s.holdingRepo.ByAccountID(accountID)
}

// ---------- Trades ----------

type RecordTradeInput struct {
	HoldingID    string
	Type         model.InvestmentTradeType
	Quantity     decimal.Decimal
	PricePerUnit decimal.Decimal
	Fees         *decimal.Decimal
	OccurredAt   time.Time
	Notes        *string
}

func (s *InvestmentService) RecordTrade(input RecordTradeInput) (*model.InvestmentTrade, error) {
	if input.HoldingID == "" {
		return nil, fmt.Errorf("holding id is required")
	}
	if !model.IsValidInvestmentTradeType(string(input.Type)) {
		return nil, fmt.Errorf("invalid trade type: %s", input.Type)
	}
	if !input.Quantity.IsPositive() {
		return nil, fmt.Errorf("quantity must be greater than zero")
	}
	if input.PricePerUnit.IsNegative() {
		return nil, fmt.Errorf("price per unit cannot be negative")
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now()
	}
	trade := &model.InvestmentTrade{
		ID:           uuid.NewString(),
		HoldingID:    input.HoldingID,
		Type:         input.Type,
		Quantity:     input.Quantity,
		PricePerUnit: input.PricePerUnit,
		Fees:         input.Fees,
		OccurredAt:   input.OccurredAt,
		Notes:        input.Notes,
		CreatedAt:    time.Now(),
	}
	if err := s.tradeRepo.Create(trade); err != nil {
		return nil, fmt.Errorf("failed to record trade: %w", err)
	}
	return trade, nil
}

func (s *InvestmentService) UpdateTrade(id string, qty, price decimal.Decimal, fees *decimal.Decimal, occurredAt time.Time, notes *string) error {
	if !qty.IsPositive() {
		return fmt.Errorf("quantity must be greater than zero")
	}
	if price.IsNegative() {
		return fmt.Errorf("price per unit cannot be negative")
	}
	return s.tradeRepo.Update(id, qty, price, fees, occurredAt, notes)
}

func (s *InvestmentService) DeleteTrade(id string) error {
	return s.tradeRepo.Delete(id)
}

func (s *InvestmentService) GetTrade(id string) (*model.InvestmentTrade, error) {
	return s.tradeRepo.ByID(id)
}

func (s *InvestmentService) ListTrades(holdingID string) ([]*model.InvestmentTrade, error) {
	return s.tradeRepo.ByHoldingID(holdingID)
}

// HoldingPositions returns the derived position for every holding in the
// account. Positions are computed by replaying each trade in chronological
// order, maintaining a running weighted-average cost basis. Each sell reduces
// the remaining quantity at the current avg cost; realized P/L accumulates on
// each sell as (sell.price − avg cost) × qty − fees.
func (s *InvestmentService) HoldingPositions(accountID string) ([]model.HoldingPosition, error) {
	holdings, err := s.holdingRepo.ByAccountID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load holdings: %w", err)
	}
	out := make([]model.HoldingPosition, 0, len(holdings))
	for _, h := range holdings {
		pos, err := s.holdingPosition(*h)
		if err != nil {
			return nil, err
		}
		out = append(out, pos)
	}
	return out, nil
}

func (s *InvestmentService) HoldingPosition(holdingID string) (*model.HoldingPosition, error) {
	h, err := s.holdingRepo.ByID(holdingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load holding: %w", err)
	}
	pos, err := s.holdingPosition(*h)
	if err != nil {
		return nil, err
	}
	return &pos, nil
}

func (s *InvestmentService) holdingPosition(h model.InvestmentHolding) (model.HoldingPosition, error) {
	trades, err := s.tradeRepo.ByHoldingID(h.ID)
	if err != nil {
		return model.HoldingPosition{}, fmt.Errorf("failed to load trades: %w", err)
	}
	pos := model.HoldingPosition{Holding: h}
	qty := decimal.Zero
	avgCost := decimal.Zero
	for _, t := range trades {
		fees := decimal.Zero
		if t.Fees != nil {
			fees = *t.Fees
		}
		pos.TotalFees = pos.TotalFees.Add(fees)
		switch t.Type {
		case model.InvestmentTradeTypeBuy:
			newQty := qty.Add(t.Quantity)
			if newQty.IsPositive() {
				// weighted average including fees in cost basis
				newCost := qty.Mul(avgCost).Add(t.Quantity.Mul(t.PricePerUnit)).Add(fees)
				avgCost = newCost.Div(newQty)
			}
			qty = newQty
			pos.TotalBuyQty = pos.TotalBuyQty.Add(t.Quantity)
			price := t.PricePerUnit
			pos.LastBuyPrice = &price
		case model.InvestmentTradeTypeSell:
			realized := t.PricePerUnit.Sub(avgCost).Mul(t.Quantity).Sub(fees)
			pos.RealizedPL = pos.RealizedPL.Add(realized)
			qty = qty.Sub(t.Quantity)
			pos.TotalSellQty = pos.TotalSellQty.Add(t.Quantity)
			price := t.PricePerUnit
			pos.LastSellPrice = &price
			if !qty.IsPositive() {
				qty = decimal.Zero
				avgCost = decimal.Zero
			}
		}
	}
	pos.Quantity = qty
	pos.AvgCost = avgCost
	pos.CostBasis = qty.Mul(avgCost)
	return pos, nil
}
