package testutil

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CreateTestUser inserts a user directly into the database.
func CreateTestUser(t *testing.T, db *sqlx.DB, email string, passwordHash *string) *model.User {
	t.Helper()
	now := time.Now()
	user := &model.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err := db.Exec(
		`INSERT INTO users (id, email, name, password_hash, email_verified_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email, user.Name, user.PasswordHash, user.EmailVerifiedAt, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestUser: %v", err)
	}
	return user
}

// CreateTestUserWithName inserts a user with a name directly into the database.
func CreateTestUserWithName(t *testing.T, db *sqlx.DB, email string, name *string) *model.User {
	t.Helper()
	now := time.Now()
	user := &model.User{
		ID:        uuid.NewString(),
		Email:     email,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO users (id, email, name, password_hash, email_verified_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email, user.Name, user.PasswordHash, user.EmailVerifiedAt, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestUserWithName: %v", err)
	}
	return user
}

// CreateTestSpace inserts a space and adds the owner as a member.
func CreateTestSpace(t *testing.T, db *sqlx.DB, ownerID, name string) *model.Space {
	t.Helper()
	now := time.Now()
	space := &model.Space{
		ID:        uuid.NewString(),
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO spaces (id, name, owner_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		space.ID, space.Name, space.OwnerID, space.CreatedAt, space.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestSpace (space): %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO space_members (space_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)`,
		space.ID, ownerID, model.RoleOwner, now,
	)
	if err != nil {
		t.Fatalf("CreateTestSpace (member): %v", err)
	}
	return space
}

// CreateTestToken inserts a token directly into the database.
func CreateTestToken(t *testing.T, db *sqlx.DB, userID, tokenType, tokenString string, expiresAt time.Time) *model.Token {
	t.Helper()
	token := &model.Token{
		ID:        uuid.NewString(),
		UserID:    userID,
		Type:      tokenType,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	_, err := db.Exec(
		`INSERT INTO tokens (id, user_id, type, token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.UserID, token.Type, token.Token, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestToken: %v", err)
	}
	return token
}

// CreateTestInvitation inserts a space invitation directly into the database.
func CreateTestInvitation(t *testing.T, db *sqlx.DB, spaceID, inviterID, email string) *model.SpaceInvitation {
	t.Helper()
	now := time.Now()
	invitation := &model.SpaceInvitation{
		Token:     uuid.NewString(),
		SpaceID:   spaceID,
		InviterID: inviterID,
		Email:     email,
		Status:    model.InvitationStatusPending,
		ExpiresAt: now.Add(48 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO space_invitations (token, space_id, inviter_id, email, status, expires_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		invitation.Token, invitation.SpaceID, invitation.InviterID, invitation.Email,
		invitation.Status, invitation.ExpiresAt, invitation.CreatedAt, invitation.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestInvitation: %v", err)
	}
	return invitation
}
