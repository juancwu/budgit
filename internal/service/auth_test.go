package service

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuthService(dbi testutil.DBInfo) *AuthService {
	cfg := testutil.TestConfig()
	userRepo := repository.NewUserRepository(dbi.DB)
	tokenRepo := repository.NewTokenRepository(dbi.DB)
	spaceRepo := repository.NewSpaceRepository(dbi.DB)
	accountRepo := repository.NewAccountRepository(dbi.DB)
	spaceSvc := NewSpaceService(spaceRepo)
	accountSvc := NewAccountService(accountRepo)
	emailSvc := NewEmailService(nil, "test@example.com", "http://localhost:9999", "Budgit Test", false)
	return NewAuthService(
		emailSvc,
		userRepo,
		tokenRepo,
		spaceSvc,
		accountSvc,
		cfg.JWTSecret,
		cfg.JWTExpiry,
		cfg.TokenMagicLinkExpiry,
		false,
	)
}

func TestAuthService_SendMagicLink(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		err := svc.SendMagicLink("newuser@example.com")
		require.NoError(t, err)

		// Verify user was created in DB
		userRepo := repository.NewUserRepository(dbi.DB)
		user, err := userRepo.ByEmail("newuser@example.com")
		require.NoError(t, err)
		assert.Equal(t, "newuser@example.com", user.Email)

		// Verify token was created in DB
		var tokenCount int
		err = dbi.DB.Get(&tokenCount, `SELECT COUNT(*) FROM tokens WHERE user_id = $1 AND type = $2`, user.ID, model.TokenTypeMagicLink)
		require.NoError(t, err)
		assert.Equal(t, 1, tokenCount)
	})
}

func TestAuthService_VerifyMagicLink(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		user := testutil.CreateTestUser(t, dbi.DB, "verify@example.com", nil)
		testutil.CreateTestToken(t, dbi.DB, user.ID, model.TokenTypeMagicLink, "test-token-123", time.Now().Add(10*time.Minute))

		got, err := svc.VerifyMagicLink("test-token-123")
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
		assert.Equal(t, user.Email, got.Email)
		assert.NotNil(t, got.EmailVerifiedAt, "email should be marked as verified")
	})
}

func TestAuthService_LoginWithPassword(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		hash, err := svc.HashPassword("testpassword1")
		require.NoError(t, err)

		user := testutil.CreateTestUser(t, dbi.DB, "login@example.com", &hash)

		got, err := svc.LoginWithPassword("login@example.com", "testpassword1")
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
		assert.Equal(t, user.Email, got.Email)
	})
}

func TestAuthService_LoginWithPassword_Wrong(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		hash, err := svc.HashPassword("testpassword1")
		require.NoError(t, err)

		testutil.CreateTestUser(t, dbi.DB, "wrongpw@example.com", &hash)

		_, err = svc.LoginWithPassword("wrongpw@example.com", "wrongpassword!")
		assert.Error(t, err)
	})
}

func TestAuthService_HashAndComparePassword(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		hash, err := svc.HashPassword("testpassword1")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)

		// Correct password should succeed
		err = svc.ComparePassword("testpassword1", hash)
		assert.NoError(t, err)

		// Wrong password should fail
		err = svc.ComparePassword("wrongpassword!", hash)
		assert.Error(t, err)
	})
}

func TestAuthService_GenerateAndVerifyJWT(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		user := testutil.CreateTestUser(t, dbi.DB, "jwt@example.com", nil)

		tokenString, err := svc.GenerateJWT(user)
		require.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		claims, err := svc.VerifyJWT(tokenString)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims["user_id"])
		assert.Equal(t, user.Email, claims["email"])
	})
}

func TestAuthService_SetPassword(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		user := testutil.CreateTestUser(t, dbi.DB, "setpw@example.com", nil)
		assert.False(t, user.HasPassword())

		err := svc.SetPassword(user.ID, "", "newpassword12", "newpassword12")
		require.NoError(t, err)

		// Verify user now has a password
		userRepo := repository.NewUserRepository(dbi.DB)
		updated, err := userRepo.ByID(user.ID)
		require.NoError(t, err)
		assert.True(t, updated.HasPassword())
	})
}

func TestAuthService_NeedsOnboarding(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		// User with no name needs onboarding
		userEmpty := testutil.CreateTestUser(t, dbi.DB, "empty@example.com", nil)

		needs, err := svc.NeedsOnboarding(userEmpty.ID)
		require.NoError(t, err)
		assert.True(t, needs)

		// User with a name does not need onboarding
		err = svc.CompleteOnboarding(userEmpty.ID, "Jane Doe")
		require.NoError(t, err)

		needs, err = svc.NeedsOnboarding(userEmpty.ID)
		require.NoError(t, err)
		assert.False(t, needs)
	})
}

func TestAuthService_CompleteOnboarding(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		user := testutil.CreateTestUser(t, dbi.DB, "onboard@example.com", nil)

		err := svc.CompleteOnboarding(user.ID, "New Name")
		require.NoError(t, err)

		// User name is updated
		userRepo := repository.NewUserRepository(dbi.DB)
		updated, err := userRepo.ByID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, updated.Name)
		assert.Equal(t, "New Name", *updated.Name)

		// A space named "<name>'s Space" was provisioned
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		spaces, err := spaceRepo.ByUserID(user.ID)
		require.NoError(t, err)
		require.Len(t, spaces, 1)
		assert.Equal(t, "New Name's Space", spaces[0].Name)

		// With a default account inside it
		accountRepo := repository.NewAccountRepository(dbi.DB)
		accounts, err := accountRepo.BySpaceID(spaces[0].ID)
		require.NoError(t, err)
		require.Len(t, accounts, 1)
		assert.Equal(t, DefaultAccountName, accounts[0].Name)
	})
}

func TestAuthService_CompleteOnboarding_Idempotent(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestAuthService(dbi)

		user := testutil.CreateTestUser(t, dbi.DB, "idempotent@example.com", nil)

		require.NoError(t, svc.CompleteOnboarding(user.ID, "Repeat User"))
		require.NoError(t, svc.CompleteOnboarding(user.ID, "Repeat User"))

		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		spaces, err := spaceRepo.ByUserID(user.ID)
		require.NoError(t, err)
		assert.Len(t, spaces, 1, "second onboarding call must not duplicate the space")

		accountRepo := repository.NewAccountRepository(dbi.DB)
		accounts, err := accountRepo.BySpaceID(spaces[0].ID)
		require.NoError(t, err)
		assert.Len(t, accounts, 1, "second onboarding call must not duplicate the default account")
	})
}
