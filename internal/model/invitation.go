package model

import "time"

type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusExpired  InvitationStatus = "expired"
)

type SpaceInvitation struct {
	Token     string           `db:"token"`
	SpaceID   string           `db:"space_id"`
	InviterID string           `db:"inviter_id"`
	Email     string           `db:"email"`
	Status    InvitationStatus `db:"status"`
	ExpiresAt time.Time        `db:"expires_at"`
	CreatedAt time.Time        `db:"created_at"`
	UpdatedAt time.Time        `db:"updated_at"`
}
