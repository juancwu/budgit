package model

import "time"

type User struct {
	ID    string  `db:"id"`
	Email string `db:"email"`
	// Allow null for passwordless users
	PasswordHash    *string    `db:"password_hash"`
	PendingEmail    *string    `db:"pending_email"`
	EmailVerifiedAt *time.Time `db:"email_verified_at"`
	CreatedAt       time.Time  `db:"created_at"`
}

func (u *User) HasPassword() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}
