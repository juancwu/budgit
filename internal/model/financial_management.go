package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID        string          `db:"id"`
	Name      string          `db:"name"`
	SpaceID   string          `db:"space_id"`
	Balance   decimal.Decimal `db:"balance"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
)

type Transaction struct {
	ID                   string          `db:"id"`
	Value                decimal.Decimal `db:"value"`
	Type                 TransactionType `db:"type"`
	AccountID            string          `db:"account_id"`
	Description          *string         `db:"description"`
	RelatedTransactionID *string         `db:"related_transaction_id"`
	CreatedAt            time.Time       `db:"created_at"`
	UpdatedAt            time.Time       `db:"updated_at"`
}

type Tag struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	SpaceID   string    `db:"space_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Category struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
