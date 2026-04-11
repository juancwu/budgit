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

func TestAccountRepository_CreateAndRead(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewAccountRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "account-create@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Space With Account")

		now := time.Now()
		account := &model.Account{
			ID:        uuid.NewString(),
			Name:      "Money Account",
			SpaceID:   space.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(account)
		require.NoError(t, err)

		fetched, err := repo.ByID(account.ID)
		require.NoError(t, err)
		assert.Equal(t, "Money Account", fetched.Name)
		assert.Equal(t, space.ID, fetched.SpaceID)

		accounts, err := repo.BySpaceID(space.ID)
		require.NoError(t, err)
		require.Len(t, accounts, 1)
		assert.Equal(t, account.ID, accounts[0].ID)
	})
}

func TestAccountRepository_ByID_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewAccountRepository(dbi.DB)

		_, err := repo.ByID(uuid.NewString())
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
}

func TestAccountRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewAccountRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "account-delete@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Delete Space")

		now := time.Now()
		account := &model.Account{
			ID:        uuid.NewString(),
			Name:      "To Delete",
			SpaceID:   space.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, repo.Create(account))

		err := repo.Delete(account.ID)
		require.NoError(t, err)

		_, err = repo.ByID(account.ID)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
}
