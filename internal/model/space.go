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
	Timezone  *string   `db:"timezone"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Location returns the *time.Location for this space's timezone.
// Returns nil if timezone is not set, so callers can distinguish "not set" from "UTC".
func (s *Space) Location() *time.Location {
	if s.Timezone == nil || *s.Timezone == "" {
		return nil
	}
	loc, err := time.LoadLocation(*s.Timezone)
	if err != nil {
		return nil
	}
	return loc
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
	Name     string    `db:"name"`
	Email    string    `db:"email"`
}
