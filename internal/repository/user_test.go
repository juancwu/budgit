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

func TestUserRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewUserRepository(dbi.DB)

		user := &model.User{
			ID:        uuid.NewString(),
			Email:     "create@example.com",
			CreatedAt: time.Now(),
		}

		id, err := repo.Create(user)
		require.NoError(t, err)
		assert.Equal(t, user.ID, id)

		fetched, err := repo.ByID(id)
		require.NoError(t, err)
		assert.Equal(t, user.Email, fetched.Email)
	})
}

func TestUserRepository_ByEmail(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewUserRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "byemail@example.com", nil)

		fetched, err := repo.ByEmail("byemail@example.com")
		require.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)
		assert.Equal(t, "byemail@example.com", fetched.Email)
	})
}

func TestUserRepository_Update(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewUserRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "before@example.com", nil)

		user.Email = "after@example.com"
		err := repo.Update(user)
		require.NoError(t, err)

		fetched, err := repo.ByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, "after@example.com", fetched.Email)
	})
}

func TestUserRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewUserRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "delete@example.com", nil)

		err := repo.Delete(user.ID)
		require.NoError(t, err)

		_, err = repo.ByID(user.ID)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestUserRepository_DuplicateEmail(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewUserRepository(dbi.DB)

		testutil.CreateTestUser(t, dbi.DB, "dup@example.com", nil)

		duplicate := &model.User{
			ID:        uuid.NewString(),
			Email:     "dup@example.com",
			CreatedAt: time.Now(),
		}

		_, err := repo.Create(duplicate)
		assert.ErrorIs(t, err, ErrDuplicateEmail)
	})
}

func TestUserRepository_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewUserRepository(dbi.DB)

		_, err := repo.ByID("nonexistent-id")
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}
