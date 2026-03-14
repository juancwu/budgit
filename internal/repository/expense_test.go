package repository

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpenseRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Food", nil)

		now := time.Now()
		expense := &model.Expense{
			ID:          uuid.NewString(),
			SpaceID:     space.ID,
			CreatedBy:   user.ID,
			Description: "Lunch",
			Amount:      decimal.RequireFromString("15.49"),
			Type:        model.ExpenseTypeExpense,
			Date:        now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err := repo.Create(expense, []string{tag.ID}, nil)
		require.NoError(t, err)

		fetched, err := repo.GetByID(expense.ID)
		require.NoError(t, err)
		assert.Equal(t, expense.ID, fetched.ID)
		assert.Equal(t, "Lunch", fetched.Description)
		assert.True(t, decimal.RequireFromString("15.49").Equal(fetched.Amount))
		assert.Equal(t, model.ExpenseTypeExpense, fetched.Type)
	})
}

func TestExpenseRepository_GetBySpaceIDPaginated(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Expense 1", decimal.RequireFromString("10.75"), model.ExpenseTypeExpense)
		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Expense 2", decimal.RequireFromString("20.50"), model.ExpenseTypeExpense)
		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Expense 3", decimal.RequireFromString("30.25"), model.ExpenseTypeExpense)

		expenses, err := repo.GetBySpaceIDPaginated(space.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, expenses, 2)
	})
}

func TestExpenseRepository_CountBySpaceID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Expense 1", decimal.RequireFromString("10.75"), model.ExpenseTypeExpense)
		testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "Expense 2", decimal.RequireFromString("20.50"), model.ExpenseTypeExpense)

		count, err := repo.CountBySpaceID(space.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestExpenseRepository_GetTagsByExpenseIDs(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Groceries", nil)

		now := time.Now()
		expense := &model.Expense{
			ID:          uuid.NewString(),
			SpaceID:     space.ID,
			CreatedBy:   user.ID,
			Description: "Weekly groceries",
			Amount:      decimal.RequireFromString("49.99"),
			Type:        model.ExpenseTypeExpense,
			Date:        now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err := repo.Create(expense, []string{tag.ID}, nil)
		require.NoError(t, err)

		tagMap, err := repo.GetTagsByExpenseIDs([]string{expense.ID})
		require.NoError(t, err)
		require.Contains(t, tagMap, expense.ID)
		require.Len(t, tagMap[expense.ID], 1)
		assert.Equal(t, tag.ID, tagMap[expense.ID][0].ID)
		assert.Equal(t, "Groceries", tagMap[expense.ID][0].Name)
	})
}

func TestExpenseRepository_GetPaymentMethodsByExpenseIDs(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		method := testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Visa", model.PaymentMethodTypeCredit, user.ID)

		now := time.Now()
		expense := &model.Expense{
			ID:              uuid.NewString(),
			SpaceID:         space.ID,
			CreatedBy:       user.ID,
			Description:     "Online purchase",
			Amount:          decimal.RequireFromString("29.95"),
			Type:            model.ExpenseTypeExpense,
			Date:            now,
			PaymentMethodID: &method.ID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		err := repo.Create(expense, nil, nil)
		require.NoError(t, err)

		methodMap, err := repo.GetPaymentMethodsByExpenseIDs([]string{expense.ID})
		require.NoError(t, err)
		require.Contains(t, methodMap, expense.ID)
		assert.Equal(t, method.ID, methodMap[expense.ID].ID)
		assert.Equal(t, "Visa", methodMap[expense.ID].Name)
		assert.Equal(t, model.PaymentMethodTypeCredit, methodMap[expense.ID].Type)
	})
}

func TestExpenseRepository_GetExpensesByTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		color := "#ff0000"
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Food", &color)

		now := time.Now()
		fromDate := now.Add(-24 * time.Hour)
		toDate := now.Add(24 * time.Hour)

		expense1 := &model.Expense{
			ID:          uuid.NewString(),
			SpaceID:     space.ID,
			CreatedBy:   user.ID,
			Description: "Lunch",
			Amount:      decimal.RequireFromString("15.49"),
			Type:        model.ExpenseTypeExpense,
			Date:        now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		err := repo.Create(expense1, []string{tag.ID}, nil)
		require.NoError(t, err)

		expense2 := &model.Expense{
			ID:          uuid.NewString(),
			SpaceID:     space.ID,
			CreatedBy:   user.ID,
			Description: "Dinner",
			Amount:      decimal.RequireFromString("24.52"),
			Type:        model.ExpenseTypeExpense,
			Date:        now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		err = repo.Create(expense2, []string{tag.ID}, nil)
		require.NoError(t, err)

		summaries, err := repo.GetExpensesByTag(space.ID, fromDate, toDate)
		require.NoError(t, err)
		require.Len(t, summaries, 1)
		assert.Equal(t, tag.ID, summaries[0].TagID)
		assert.Equal(t, "Food", summaries[0].TagName)
		assert.True(t, decimal.RequireFromString("40.01").Equal(summaries[0].TotalAmount))
	})
}

func TestExpenseRepository_Update(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		tag1 := testutil.CreateTestTag(t, dbi.DB, space.ID, "Tag A", nil)
		tag2 := testutil.CreateTestTag(t, dbi.DB, space.ID, "Tag B", nil)

		now := time.Now()
		expense := &model.Expense{
			ID:          uuid.NewString(),
			SpaceID:     space.ID,
			CreatedBy:   user.ID,
			Description: "Original",
			Amount:      decimal.RequireFromString("10.75"),
			Type:        model.ExpenseTypeExpense,
			Date:        now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err := repo.Create(expense, []string{tag1.ID}, nil)
		require.NoError(t, err)

		expense.Description = "Updated"
		expense.UpdatedAt = time.Now()
		err = repo.Update(expense, []string{tag2.ID})
		require.NoError(t, err)

		fetched, err := repo.GetByID(expense.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", fetched.Description)

		tagMap, err := repo.GetTagsByExpenseIDs([]string{expense.ID})
		require.NoError(t, err)
		require.Len(t, tagMap[expense.ID], 1)
		assert.Equal(t, tag2.ID, tagMap[expense.ID][0].ID)
	})
}

func TestExpenseRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewExpenseRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		expense := testutil.CreateTestExpense(t, dbi.DB, space.ID, user.ID, "To Delete", decimal.RequireFromString("4.99"), model.ExpenseTypeExpense)

		err := repo.Delete(expense.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(expense.ID)
		assert.ErrorIs(t, err, ErrExpenseNotFound)
	})
}
