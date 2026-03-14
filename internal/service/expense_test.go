package service

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/shopspring/decimal"
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
			Amount:      decimal.RequireFromString("15.49"),
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
			TagIDs:      []string{tag.ID},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, expense.ID)
		assert.Equal(t, "Lunch", expense.Description)
		assert.True(t, decimal.RequireFromString("15.49").Equal(expense.Amount))
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
			Amount:      decimal.RequireFromString("10.75"),
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
			Amount:      decimal.Zero,
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
			Amount:      decimal.RequireFromString("2.49"),
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
			Amount:      decimal.RequireFromString("5.01"),
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

		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Topup", decimal.RequireFromString("100.50"), model.ExpenseTypeTopup)
		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Groceries", decimal.RequireFromString("30.75"), model.ExpenseTypeExpense)

		balance, err := svc.GetBalanceForSpace(space.ID)
		require.NoError(t, err)
		assert.True(t, decimal.RequireFromString("69.75").Equal(balance))
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
			Amount:      decimal.RequireFromString("24.99"),
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
		assert.True(t, decimal.RequireFromString("24.99").Equal(summaries[0].TotalAmount))
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
			Amount:      decimal.RequireFromString("10.75"),
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		require.NoError(t, err)

		updated, err := svc.UpdateExpense(UpdateExpenseDTO{
			ID:          created.ID,
			SpaceID:     space.ID,
			Description: "New Description",
			Amount:      decimal.RequireFromString("19.49"),
			Type:        model.ExpenseTypeExpense,
			Date:        time.Now(),
		})
		require.NoError(t, err)
		assert.Equal(t, "New Description", updated.Description)
		assert.True(t, decimal.RequireFromString("19.49").Equal(updated.Amount))
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
			Amount:      decimal.RequireFromString("4.99"),
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
