package model

import "time"

const (
	AccountDeletionStatusPending    = "pending"
	AccountDeletionStatusProcessing = "processing"
	AccountDeletionStatusCompleted  = "completed"
	AccountDeletionStatusFailed     = "failed"
)

// AccountDeletionRequest is both the work queue entry and the historical
// audit record for an account deletion. The row is created when the user
// confirms deletion, transitions through processing, and is kept after
// completion as the audit trail (the related user row is gone by then).
type AccountDeletionRequest struct {
	ID            string     `db:"id"`
	UserID        string     `db:"user_id"`
	Email         string     `db:"email"`
	Name          *string    `db:"name"`
	Reason        *string    `db:"reason"`
	IPAddress     *string    `db:"ip_address"`
	Status        string     `db:"status"`
	Attempts      int        `db:"attempts"`
	LastError     *string    `db:"last_error"`
	SpacesDeleted *int       `db:"spaces_deleted"`
	RequestedAt   time.Time  `db:"requested_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
	CompletedAt   *time.Time `db:"completed_at"`
}
