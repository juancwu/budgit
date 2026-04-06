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
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type SpaceMember struct {
	SpaceID  string    `db:"space_id"`
	UserID   string    `db:"user_id"`
	Role     Role      `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
}

type SpaceInvitation struct {
	Token        string    `db:"token"`
	SpaceID      string    `db:"space_id"`
	InviterID    string    `db:"inviter_id"`
	InviteeEmail string    `db:"invitee_email"`
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}
