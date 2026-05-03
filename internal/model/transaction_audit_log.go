package model

import "time"

type TransactionAuditAction string

const (
	TransactionAuditActionCreated TransactionAuditAction = "transaction.created"
	TransactionAuditActionEdited  TransactionAuditAction = "transaction.edited"
	TransactionAuditActionDeleted TransactionAuditAction = "transaction.deleted"
)

type TransactionAuditLog struct {
	ID            string                 `db:"id"`
	TransactionID string                 `db:"transaction_id"`
	ActorID       *string                `db:"actor_id"`
	Action        TransactionAuditAction `db:"action"`
	Metadata      []byte                 `db:"metadata"`
	CreatedAt     time.Time              `db:"created_at"`
}

type TransactionAuditLogWithActor struct {
	TransactionAuditLog
	ActorName  *string `db:"actor_name"`
	ActorEmail *string `db:"actor_email"`
}
