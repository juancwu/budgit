package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type InvestmentContributionRoom struct {
	AccountID  string          `db:"account_id"`
	Year       int             `db:"year"`
	RoomAmount decimal.Decimal `db:"room_amount"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

type InvestmentHolding struct {
	ID          string    `db:"id"`
	AccountID   string    `db:"account_id"`
	Symbol      string    `db:"symbol"`
	DisplayName string    `db:"display_name"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type InvestmentTradeType string

const (
	InvestmentTradeTypeBuy  InvestmentTradeType = "buy"
	InvestmentTradeTypeSell InvestmentTradeType = "sell"
)

func IsValidInvestmentTradeType(t string) bool {
	switch InvestmentTradeType(t) {
	case InvestmentTradeTypeBuy, InvestmentTradeTypeSell:
		return true
	}
	return false
}

type InvestmentTrade struct {
	ID           string              `db:"id"`
	HoldingID    string              `db:"holding_id"`
	Type         InvestmentTradeType `db:"type"`
	Quantity     decimal.Decimal     `db:"quantity"`
	PricePerUnit decimal.Decimal     `db:"price_per_unit"`
	Fees         *decimal.Decimal    `db:"fees"`
	OccurredAt   time.Time           `db:"occurred_at"`
	Notes        *string             `db:"notes"`
	CreatedAt    time.Time           `db:"created_at"`
}

// HoldingPosition aggregates a holding with its derived figures across all
// trades. Quantity is the net of buys minus sells. AvgCost is the weighted
// average per-unit cost of remaining shares (reduced proportionally on sells).
// RealizedPL is the cumulative realized profit/loss from sells.
type HoldingPosition struct {
	Holding        InvestmentHolding
	Quantity       decimal.Decimal
	AvgCost        decimal.Decimal
	CostBasis      decimal.Decimal
	LastBuyPrice   *decimal.Decimal
	LastSellPrice  *decimal.Decimal
	RealizedPL     decimal.Decimal
	TotalBuyQty    decimal.Decimal
	TotalSellQty   decimal.Decimal
	TotalFees      decimal.Decimal
}

// InvestmentAccountSummary is the rolled-up view for an investment-flagged
// account: contribution room and YTD cash flow plus aggregate cost basis across
// holdings.
type InvestmentAccountSummary struct {
	Account            *Account
	Year               int
	RoomAmount         *decimal.Decimal // nil if room not yet set for the year
	YTDContributions   decimal.Decimal
	YTDWithdrawals     decimal.Decimal
	RoomRemaining      *decimal.Decimal // nil if RoomAmount is nil
	NetContributions   decimal.Decimal  // lifetime: all deposits minus all withdrawals
	TotalCostBasis     decimal.Decimal
	HoldingCount       int
}
