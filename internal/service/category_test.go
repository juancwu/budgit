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
	return svc, space.ID
}

func TestCategoryService_CreateAndList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, spaceID := newCategoryFixture(t, dbi)

		// Nothing until created — no predefined categories.
		cats, err := svc.ListBySpace(spaceID)
		require.NoError(t, err)
		assert.Empty(t, cats)

		created, err := svc.Create(spaceID, "  Groceries  ", "food")
		require.NoError(t, err)
		assert.Equal(t, "Groceries", created.Name, "name is trimmed")
		assert.Equal(t, spaceID, created.SpaceID)

		cats, err = svc.ListBySpace(spaceID)
		require.NoError(t, err)
		require.Len(t, cats, 1)
		assert.Equal(t, "Groceries", cats[0].Name)
	})
}

func TestCategoryService_Create_RejectsBlankAndDuplicate(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, spaceID := newCategoryFixture(t, dbi)

		_, err := svc.Create(spaceID, "   ", "")
		assert.Error(t, err, "blank name rejected")

		_, err = svc.Create(spaceID, "Rent", "")
		require.NoError(t, err)

		// Case-insensitive duplicate within the same space.
		_, err = svc.Create(spaceID, "rent", "")
		assert.ErrorIs(t, err, ErrCategoryNameTaken)
	})
}

func TestCategoryService_Create_SameNameDifferentSpaces(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, spaceA := newCategoryFixture(t, dbi)
		userB := testutil.CreateTestUser(t, dbi.DB, "b@example.com", nil)
		spaceB := testutil.CreateTestSpace(t, dbi.DB, userB.ID, "B").ID

		_, err := svc.Create(spaceA, "Travel", "")
		require.NoError(t, err)
		// Same name is fine in a different space.
		_, err = svc.Create(spaceB, "Travel", "")
		require.NoError(t, err)
	})
}

func TestCategoryService_GetAndDelete_ScopedToSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, spaceA := newCategoryFixture(t, dbi)
		userB := testutil.CreateTestUser(t, dbi.DB, "b2@example.com", nil)
		spaceB := testutil.CreateTestSpace(t, dbi.DB, userB.ID, "B").ID

		cat, err := svc.Create(spaceA, "Health", "")
		require.NoError(t, err)

		// A different space cannot see or delete it.
		_, err = svc.Get(spaceB, cat.ID)
		assert.ErrorIs(t, err, ErrCategoryNotFound)
		err = svc.Delete(spaceB, cat.ID)
		assert.ErrorIs(t, err, ErrCategoryNotFound)

		// The owning space can.
		got, err := svc.Get(spaceA, cat.ID)
		require.NoError(t, err)
		assert.Equal(t, cat.ID, got.ID)

		require.NoError(t, svc.Delete(spaceA, cat.ID))
		cats, err := svc.ListBySpace(spaceA)
		require.NoError(t, err)
		assert.Empty(t, cats)
	})
}

func TestCategoryService_Get_UnknownID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc, spaceID := newCategoryFixture(t, dbi)
		_, err := svc.Get(spaceID, "does-not-exist")
		assert.True(t, errors.Is(err, ErrCategoryNotFound))
	})
}
