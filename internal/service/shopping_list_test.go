package service

import (
	"fmt"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShoppingListService_CreateList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-create@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Space")

		list, err := svc.CreateList(space.ID, "Weekly Groceries")
		require.NoError(t, err)
		assert.NotEmpty(t, list.ID)
		assert.Equal(t, "Weekly Groceries", list.Name)
		assert.Equal(t, space.ID, list.SpaceID)
	})
}

func TestShoppingListService_CreateList_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-empty@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Empty Space")

		list, err := svc.CreateList(space.ID, "")
		assert.Error(t, err)
		assert.Nil(t, list)
	})
}

func TestShoppingListService_GetList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-get@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Get Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Seeded List")

		list, err := svc.GetList(seeded.ID)
		require.NoError(t, err)
		assert.Equal(t, seeded.ID, list.ID)
		assert.Equal(t, "Seeded List", list.Name)
	})
}

func TestShoppingListService_UpdateList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-update@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Update Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Old Name")

		updated, err := svc.UpdateList(seeded.ID, "New Name")
		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.Name)
	})
}

func TestShoppingListService_DeleteList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-del@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Del Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Doomed List")
		testutil.CreateTestListItem(t, dbi.DB, seeded.ID, "Item 1", user.ID)
		testutil.CreateTestListItem(t, dbi.DB, seeded.ID, "Item 2", user.ID)

		err := svc.DeleteList(seeded.ID)
		require.NoError(t, err)

		_, err = svc.GetList(seeded.ID)
		assert.Error(t, err)

		items, err := itemRepo.GetByListID(seeded.ID)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestShoppingListService_AddItemToList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-additem@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc AddItem Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Add Item List")

		item, err := svc.AddItemToList(seeded.ID, "Milk", user.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, item.ID)
		assert.Equal(t, "Milk", item.Name)
		assert.Equal(t, seeded.ID, item.ListID)
		assert.False(t, item.IsChecked)
	})
}

func TestShoppingListService_GetItemsForListPaginated(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-paginate@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Paginate Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Paginate List")

		for i := 0; i < 6; i++ {
			testutil.CreateTestListItem(t, dbi.DB, seeded.ID, fmt.Sprintf("Item %d", i), user.ID)
		}

		items, totalPages, err := svc.GetItemsForListPaginated(seeded.ID, 1)
		require.NoError(t, err)
		assert.Len(t, items, 5)
		assert.Equal(t, 2, totalPages)
	})
}

func TestShoppingListService_CheckItem(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-check@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Check Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Check List")
		item := testutil.CreateTestListItem(t, dbi.DB, seeded.ID, "Check Me", user.ID)

		err := svc.CheckItem(item.ID)
		require.NoError(t, err)

		fetched, err := svc.GetItem(item.ID)
		require.NoError(t, err)
		assert.True(t, fetched.IsChecked)
	})
}

func TestShoppingListService_GetListsWithUncheckedItems(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-unchecked@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc Unchecked Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Unchecked List")

		checkedItem := testutil.CreateTestListItem(t, dbi.DB, seeded.ID, "Checked Item", user.ID)
		testutil.CreateTestListItem(t, dbi.DB, seeded.ID, "Unchecked Item", user.ID)

		_, err := dbi.DB.Exec("UPDATE list_items SET is_checked = true WHERE id = $1", checkedItem.ID)
		require.NoError(t, err)

		result, err := svc.GetListsWithUncheckedItems(space.ID)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, seeded.ID, result[0].List.ID)
		require.Len(t, result[0].Items, 1)
		assert.Equal(t, "Unchecked Item", result[0].Items[0].Name)
	})
}

func TestShoppingListService_DeleteItem(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		listRepo := repository.NewShoppingListRepository(dbi.DB)
		itemRepo := repository.NewListItemRepository(dbi.DB)
		svc := NewShoppingListService(listRepo, itemRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "list-svc-delitem@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "List Svc DelItem Space")
		seeded := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "DelItem List")
		item := testutil.CreateTestListItem(t, dbi.DB, seeded.ID, "Doomed Item", user.ID)

		err := svc.DeleteItem(item.ID)
		require.NoError(t, err)

		_, err = svc.GetItem(item.ID)
		assert.Error(t, err)
	})
}
