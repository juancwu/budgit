package service

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpenseService_CreateExpense(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "exp-svc-create@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Expense Svc Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Food", nil)

		expense, err := svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     space.ID,
			UserID:      user.ID,
			Description: "Lunch",
			Amount:      1500,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
			TagIDs:      []string{tag.ID},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, expense.ID)
		assert.Equal(t, "Lunch", expense.Description)
		assert.Equal(t, 1500, expense.AmountCents)
		assert.Equal(t, model.ExpenseTypeExpense, expense.Type)
	})
}

func TestExpenseService_CreateExpense_EmptyDescription(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		expense, err := svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     "some-space",
			UserID:      "some-user",
			Description: "",
			Amount:      1000,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		assert.Error(t, err)
		assert.Nil(t, expense)
	})
}

func TestExpenseService_CreateExpense_ZeroAmount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		expense, err := svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     "some-space",
			UserID:      "some-user",
			Description: "Something",
			Amount:      0,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		assert.Error(t, err)
		assert.Nil(t, expense)
	})
}

func TestExpenseService_GetExpensesWithTagsForSpacePaginated(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "exp-svc-paginate@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Expense Svc Paginate Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Transport", nil)

		// Create expense with tag via the service
		_, err := svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     space.ID,
			UserID:      user.ID,
			Description: "Bus fare",
			Amount:      250,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
			TagIDs:      []string{tag.ID},
		})
		require.NoError(t, err)

		// Create expense without tag
		_, err = svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     space.ID,
			UserID:      user.ID,
			Description: "Coffee",
			Amount:      500,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		require.NoError(t, err)

		results, totalPages, err := svc.GetExpensesWithTagsForSpacePaginated(space.ID, 1)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, 1, totalPages)

		// Verify at least one result has tags and one does not
		var withTags, withoutTags int
		for _, r := range results {
			if len(r.Tags) > 0 {
				withTags++
			} else {
				withoutTags++
			}
		}
		assert.Equal(t, 1, withTags)
		assert.Equal(t, 1, withoutTags)
	})
}

func TestExpenseService_GetBalanceForSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "exp-svc-balance@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Expense Svc Balance Space")

		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Topup", 10000, model.ExpenseTypeTopup)
		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Groceries", 3000, model.ExpenseTypeExpense)

		balance, err := svc.GetBalanceForSpace(space.ID)
		require.NoError(t, err)
		assert.Equal(t, 7000, balance)
	})
}

func TestExpenseService_GetExpensesByTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "exp-svc-bytag@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Expense Svc ByTag Space")
		tagColor := "#ff0000"
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Dining", &tagColor)

		now := time.Now()
		_, err := svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     space.ID,
			UserID:      user.ID,
			Description: "Dinner",
			Amount:      2500,
			Type:        model.ExpenseTypeExpense,
			Date:        now,
			TagIDs:      []string{tag.ID},
		})
		require.NoError(t, err)

		fromDate := now.Add(-24 * time.Hour)
		toDate := now.Add(24 * time.Hour)
		summaries, err := svc.GetExpensesByTag(space.ID, fromDate, toDate)
		require.NoError(t, err)
		require.Len(t, summaries, 1)
		assert.Equal(t, tag.ID, summaries[0].TagID)
		assert.Equal(t, 2500, summaries[0].TotalAmount)
	})
}

func TestExpenseService_UpdateExpense(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "exp-svc-update@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Expense Svc Update Space")

		created, err := svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     space.ID,
			UserID:      user.ID,
			Description: "Old Description",
			Amount:      1000,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		require.NoError(t, err)

		updated, err := svc.UpdateExpense(UpdateExpenseDTO{
			ID:          created.ID,
			SpaceID:     space.ID,
			Description: "New Description",
			Amount:      2000,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		require.NoError(t, err)
		assert.Equal(t, "New Description", updated.Description)
		assert.Equal(t, 2000, updated.AmountCents)
	})
}

func TestExpenseService_DeleteExpense(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		svc := NewExpenseService(expenseRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "exp-svc-delete@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Expense Svc Delete Space")

		created, err := svc.CreateExpense(CreateExpenseDTO{
			SpaceID:     space.ID,
			UserID:      user.ID,
			Description: "Doomed Expense",
			Amount:      500,
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		require.NoError(t, err)

		err = svc.DeleteExpense(created.ID, space.ID)
		require.NoError(t, err)

		_, err = svc.GetExpense(created.ID)
		assert.Error(t, err)
	})
}
