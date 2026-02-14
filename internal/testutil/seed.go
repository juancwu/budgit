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
	user := &model.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	_, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, email_verified_at, created_at) VALUES ($1, $2, $3, $4, $5)`,
		user.ID, user.Email, user.PasswordHash, user.EmailVerifiedAt, user.CreatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestUser: %v", err)
	}
	return user
}

// CreateTestProfile inserts a profile directly into the database.
func CreateTestProfile(t *testing.T, db *sqlx.DB, userID, name string) *model.Profile {
	t.Helper()
	now := time.Now()
	profile := &model.Profile{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO profiles (id, user_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		profile.ID, profile.UserID, profile.Name, profile.CreatedAt, profile.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestProfile: %v", err)
	}
	return profile
}

// CreateTestUserWithProfile creates both a user and a profile.
func CreateTestUserWithProfile(t *testing.T, db *sqlx.DB, email, name string) (*model.User, *model.Profile) {
	t.Helper()
	user := CreateTestUser(t, db, email, nil)
	profile := CreateTestProfile(t, db, user.ID, name)
	return user, profile
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

// CreateTestTag inserts a tag directly into the database.
func CreateTestTag(t *testing.T, db *sqlx.DB, spaceID, name string, color *string) *model.Tag {
	t.Helper()
	now := time.Now()
	tag := &model.Tag{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO tags (id, space_id, name, color, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		tag.ID, tag.SpaceID, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestTag: %v", err)
	}
	return tag
}

// CreateTestShoppingList inserts a shopping list directly into the database.
func CreateTestShoppingList(t *testing.T, db *sqlx.DB, spaceID, name string) *model.ShoppingList {
	t.Helper()
	now := time.Now()
	list := &model.ShoppingList{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO shopping_lists (id, space_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		list.ID, list.SpaceID, list.Name, list.CreatedAt, list.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestShoppingList: %v", err)
	}
	return list
}

// CreateTestListItem inserts a list item directly into the database.
func CreateTestListItem(t *testing.T, db *sqlx.DB, listID, name, createdBy string) *model.ListItem {
	t.Helper()
	now := time.Now()
	item := &model.ListItem{
		ID:        uuid.NewString(),
		ListID:    listID,
		Name:      name,
		IsChecked: false,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO list_items (id, list_id, name, is_checked, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		item.ID, item.ListID, item.Name, item.IsChecked, item.CreatedBy, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestListItem: %v", err)
	}
	return item
}

// CreateTestExpense inserts an expense directly into the database.
func CreateTestExpense(t *testing.T, db *sqlx.DB, spaceID, userID, desc string, amount int, typ model.ExpenseType) *model.Expense {
	t.Helper()
	now := time.Now()
	expense := &model.Expense{
		ID:          uuid.NewString(),
		SpaceID:     spaceID,
		CreatedBy:   userID,
		Description: desc,
		AmountCents: amount,
		Type:        typ,
		Date:        now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := db.Exec(
		`INSERT INTO expenses (id, space_id, created_by, description, amount_cents, type, date, payment_method_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		expense.ID, expense.SpaceID, expense.CreatedBy, expense.Description, expense.AmountCents,
		expense.Type, expense.Date, expense.PaymentMethodID, expense.CreatedAt, expense.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestExpense: %v", err)
	}
	return expense
}

// CreateTestMoneyAccount inserts a money account directly into the database.
func CreateTestMoneyAccount(t *testing.T, db *sqlx.DB, spaceID, name, createdBy string) *model.MoneyAccount {
	t.Helper()
	now := time.Now()
	account := &model.MoneyAccount{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		Name:      name,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO money_accounts (id, space_id, name, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		account.ID, account.SpaceID, account.Name, account.CreatedBy, account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestMoneyAccount: %v", err)
	}
	return account
}

// CreateTestTransfer inserts an account transfer directly into the database.
func CreateTestTransfer(t *testing.T, db *sqlx.DB, accountID string, amount int, direction model.TransferDirection, createdBy string) *model.AccountTransfer {
	t.Helper()
	transfer := &model.AccountTransfer{
		ID:          uuid.NewString(),
		AccountID:   accountID,
		AmountCents: amount,
		Direction:   direction,
		Note:        "test transfer",
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}
	_, err := db.Exec(
		`INSERT INTO account_transfers (id, account_id, amount_cents, direction, note, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		transfer.ID, transfer.AccountID, transfer.AmountCents, transfer.Direction, transfer.Note, transfer.CreatedBy, transfer.CreatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestTransfer: %v", err)
	}
	return transfer
}

// CreateTestPaymentMethod inserts a payment method directly into the database.
func CreateTestPaymentMethod(t *testing.T, db *sqlx.DB, spaceID, name string, typ model.PaymentMethodType, createdBy string) *model.PaymentMethod {
	t.Helper()
	lastFour := "1234"
	now := time.Now()
	method := &model.PaymentMethod{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		Name:      name,
		Type:      typ,
		LastFour:  &lastFour,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Exec(
		`INSERT INTO payment_methods (id, space_id, name, type, last_four, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		method.ID, method.SpaceID, method.Name, method.Type, method.LastFour, method.CreatedBy, method.CreatedAt, method.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTestPaymentMethod: %v", err)
	}
	return method
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
