package model

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

type Space struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	OwnerID   string    `db:"owner_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type SpaceMember struct {
	SpaceID  string    `db:"space_id"`
	UserID   string    `db:"user_id"`
	Role     Role      `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
}

type SpaceMemberWithProfile struct {
	SpaceID  string    `db:"space_id"`
	UserID   string    `db:"user_id"`
	Role     Role      `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
	Name     *string   `db:"name"`
	Email    string    `db:"email"`
}

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
