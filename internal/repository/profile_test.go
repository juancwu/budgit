package repository

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewProfileRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "profile-create@example.com", nil)

		now := time.Now()
		profile := &model.Profile{
			ID:        uuid.NewString(),
			UserID:    user.ID,
			Name:      "Test User",
			CreatedAt: now,
			UpdatedAt: now,
		}

		id, err := repo.Create(profile)
		require.NoError(t, err)
		assert.Equal(t, profile.ID, id)

		fetched, err := repo.ByUserID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test User", fetched.Name)
		assert.Equal(t, user.ID, fetched.UserID)
	})
}

func TestProfileRepository_UpdateName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewProfileRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "profile-update@example.com", nil)
		testutil.CreateTestProfile(t, dbi.DB, user.ID, "Old Name")

		err := repo.UpdateName(user.ID, "New Name")
		require.NoError(t, err)

		fetched, err := repo.ByUserID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Name", fetched.Name)
	})
}

func TestProfileRepository_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewProfileRepository(dbi.DB)

		_, err := repo.ByUserID("nonexistent-id")
		assert.ErrorIs(t, err, ErrProfileNotFound)
	})
}
