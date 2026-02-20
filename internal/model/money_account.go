package model

import "time"

type TransferDirection string

const (
	TransferDirectionDeposit    TransferDirection = "deposit"
	TransferDirectionWithdrawal TransferDirection = "withdrawal"
)

type MoneyAccount struct {
	ID        string    `db:"id"`
	SpaceID   string    `db:"space_id"`
	Name      string    `db:"name"`
	CreatedBy string    `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type AccountTransfer struct {
	ID                 string            `db:"id"`
	AccountID          string            `db:"account_id"`
	AmountCents        int               `db:"amount_cents"`
	Direction          TransferDirection `db:"direction"`
	Note               string            `db:"note"`
	RecurringDepositID *string           `db:"recurring_deposit_id"`
	CreatedBy          string            `db:"created_by"`
	CreatedAt          time.Time         `db:"created_at"`
}

type MoneyAccountWithBalance struct {
	MoneyAccount
	BalanceCents int
}
