package service

import (
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileService_ByUserID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		profileRepo := repository.NewProfileRepository(dbi.DB)
		svc := NewProfileService(profileRepo)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "profile@example.com", "Test User")

		got, err := svc.ByUserID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, profile.ID, got.ID)
		assert.Equal(t, user.ID, got.UserID)
		assert.Equal(t, "Test User", got.Name)
	})
}

func TestProfileService_ByUserID_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		profileRepo := repository.NewProfileRepository(dbi.DB)
		svc := NewProfileService(profileRepo)

		_, err := svc.ByUserID("nonexistent-id")
		assert.Error(t, err)
	})
}
