package service

import (
	"errors"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCategoryFixture(t *testing.T, dbi testutil.DBInfo) (*CategoryService, string) {
	t.Helper()
	svc := NewCategoryService(repository.NewCategoryRepository(dbi.DB))
	user := testutil.CreateTestUser(t, dbi.DB, "cat@example.com", nil)
	space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")
	account := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Acct")
	return svc, account.ID
}

func TestCategoryService_CreateAndList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, accountID := newCategoryFixture(t, dbi)

		// Nothing until created — no predefined categories.
		cats, err := svc.ListByAccount(accountID)
		require.NoError(t, err)
		assert.Empty(t, cats)

		created, err := svc.Create(accountID, "  Groceries  ", "food")
		require.NoError(t, err)
		assert.Equal(t, "Groceries", created.Name, "name is trimmed")
		assert.Equal(t, accountID, created.AccountID)

		cats, err = svc.ListByAccount(accountID)
		require.NoError(t, err)
		require.Len(t, cats, 1)
		assert.Equal(t, "Groceries", cats[0].Name)
	})
}

func TestCategoryService_Create_RejectsBlankAndDuplicate(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, accountID := newCategoryFixture(t, dbi)

		_, err := svc.Create(accountID, "   ", "")
		assert.Error(t, err, "blank name rejected")

		_, err = svc.Create(accountID, "Rent", "")
		require.NoError(t, err)

		// Case-insensitive duplicate within the same account.
		_, err = svc.Create(accountID, "rent", "")
		assert.ErrorIs(t, err, ErrCategoryNameTaken)
	})
}

func TestCategoryService_Create_SameNameDifferentAccounts(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, accountA := newCategoryFixture(t, dbi)
		user := testutil.CreateTestUser(t, dbi.DB, "b@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "B")
		accountB := testutil.CreateTestAccount(t, dbi.DB, space.ID, "AcctB").ID

		_, err := svc.Create(accountA, "Travel", "")
		require.NoError(t, err)
		// Same name is fine in a different account.
		_, err = svc.Create(accountB, "Travel", "")
		require.NoError(t, err)
	})
}

func TestCategoryService_GetAndDelete_ScopedToAccount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, accountA := newCategoryFixture(t, dbi)
		user := testutil.CreateTestUser(t, dbi.DB, "b2@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "B")
		accountB := testutil.CreateTestAccount(t, dbi.DB, space.ID, "AcctB").ID

		cat, err := svc.Create(accountA, "Health", "")
		require.NoError(t, err)

		// A different account cannot see or delete it.
		_, err = svc.Get(accountB, cat.ID)
		assert.ErrorIs(t, err, ErrCategoryNotFound)
		err = svc.Delete(accountB, cat.ID)
		assert.ErrorIs(t, err, ErrCategoryNotFound)

		// The owning account can.
		got, err := svc.Get(accountA, cat.ID)
		require.NoError(t, err)
		assert.Equal(t, cat.ID, got.ID)

		require.NoError(t, svc.Delete(accountA, cat.ID))
		cats, err := svc.ListByAccount(accountA)
		require.NoError(t, err)
		assert.Empty(t, cats)
	})
}

func TestCategoryService_Get_UnknownID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, accountID := newCategoryFixture(t, dbi)
		_, err := svc.Get(accountID, "does-not-exist")
		assert.True(t, errors.Is(err, ErrCategoryNotFound))
	})
}
