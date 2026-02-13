package model

import "time"

type PaymentMethodType string

const (
	PaymentMethodTypeCredit PaymentMethodType = "credit"
	PaymentMethodTypeDebit  PaymentMethodType = "debit"
)

type PaymentMethod struct {
	ID        string            `db:"id"`
	SpaceID   string            `db:"space_id"`
	Name      string            `db:"name"`
	Type      PaymentMethodType `db:"type"`
	LastFour  *string           `db:"last_four"`
	CreatedBy string            `db:"created_by"`
	CreatedAt time.Time         `db:"created_at"`
	UpdatedAt time.Time         `db:"updated_at"`
}
