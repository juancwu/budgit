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

func TestListItemRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewListItemRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Test List")

		now := time.Now()
		item := &model.ListItem{
			ID:        uuid.NewString(),
			ListID:    list.ID,
			Name:      "Apples",
			IsChecked: false,
			CreatedBy: user.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(item)
		require.NoError(t, err)

		fetched, err := repo.GetByID(item.ID)
		require.NoError(t, err)
		assert.Equal(t, item.ID, fetched.ID)
		assert.Equal(t, list.ID, fetched.ListID)
		assert.Equal(t, "Apples", fetched.Name)
		assert.False(t, fetched.IsChecked)
		assert.Equal(t, user.ID, fetched.CreatedBy)
	})
}

func TestListItemRepository_GetByListID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewListItemRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Test List")

		item1 := testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item A", user.ID)
		time.Sleep(10 * time.Millisecond)
		item2 := testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item B", user.ID)

		items, err := repo.GetByListID(list.ID)
		require.NoError(t, err)
		require.Len(t, items, 2)

		// Ordered by created_at ASC, so item1 should be first.
		assert.Equal(t, item1.ID, items[0].ID)
		assert.Equal(t, item2.ID, items[1].ID)
	})
}

func TestListItemRepository_GetByListIDPaginated(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewListItemRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Test List")

		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item A", user.ID)
		time.Sleep(10 * time.Millisecond)
		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item B", user.ID)
		time.Sleep(10 * time.Millisecond)
		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item C", user.ID)

		items, err := repo.GetByListIDPaginated(list.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, items, 2)

		count, err := repo.CountByListID(list.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}

func TestListItemRepository_CountByListID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewListItemRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Test List")

		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item A", user.ID)
		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item B", user.ID)
		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item C", user.ID)

		count, err := repo.CountByListID(list.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}

func TestListItemRepository_Update(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewListItemRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Test List")

		item := testutil.CreateTestListItem(t, dbi.DB, list.ID, "Original", user.ID)

		item.Name = "Updated"
		item.IsChecked = true
		err := repo.Update(item)
		require.NoError(t, err)

		fetched, err := repo.GetByID(item.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", fetched.Name)
		assert.True(t, fetched.IsChecked)
	})
}

func TestListItemRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewListItemRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Test List")

		item := testutil.CreateTestListItem(t, dbi.DB, list.ID, "To Delete", user.ID)

		err := repo.Delete(item.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(item.ID)
		assert.ErrorIs(t, err, ErrListItemNotFound)
	})
}

func TestListItemRepository_DeleteByListID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewListItemRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Test List")

		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item A", user.ID)
		testutil.CreateTestListItem(t, dbi.DB, list.ID, "Item B", user.ID)

		err := repo.DeleteByListID(list.ID)
		require.NoError(t, err)

		items, err := repo.GetByListID(list.ID)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}
