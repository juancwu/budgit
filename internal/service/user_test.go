package service

import (
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_ByID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		userRepo := repository.NewUserRepository(dbi.DB)
		svc := NewUserService(userRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)

		got, err := svc.ByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
		assert.Equal(t, user.Email, got.Email)
	})
}

func TestUserService_ByID_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		userRepo := repository.NewUserRepository(dbi.DB)
		svc := NewUserService(userRepo)

		_, err := svc.ByID("nonexistent-id")
		assert.Error(t, err)
	})
}
