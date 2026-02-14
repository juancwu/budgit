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

func TestShoppingListRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewShoppingListRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		now := time.Now()
		list := &model.ShoppingList{
			ID:        uuid.NewString(),
			SpaceID:   space.ID,
			Name:      "Groceries",
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(list)
		require.NoError(t, err)

		fetched, err := repo.GetByID(list.ID)
		require.NoError(t, err)
		assert.Equal(t, list.ID, fetched.ID)
		assert.Equal(t, space.ID, fetched.SpaceID)
		assert.Equal(t, "Groceries", fetched.Name)
	})
}

func TestShoppingListRepository_GetBySpaceID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewShoppingListRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		list1 := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "List A")
		// Small delay to ensure distinct created_at timestamps for ordering.
		time.Sleep(10 * time.Millisecond)
		list2 := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "List B")

		lists, err := repo.GetBySpaceID(space.ID)
		require.NoError(t, err)
		require.Len(t, lists, 2)

		// Ordered by created_at DESC, so list2 should be first.
		assert.Equal(t, list2.ID, lists[0].ID)
		assert.Equal(t, list1.ID, lists[1].ID)
	})
}

func TestShoppingListRepository_Update(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewShoppingListRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Original Name")

		list.Name = "Updated Name"
		err := repo.Update(list)
		require.NoError(t, err)

		fetched, err := repo.GetByID(list.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", fetched.Name)
	})
}

func TestShoppingListRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewShoppingListRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "To Delete")

		err := repo.Delete(list.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(list.ID)
		assert.ErrorIs(t, err, ErrShoppingListNotFound)
	})
}
