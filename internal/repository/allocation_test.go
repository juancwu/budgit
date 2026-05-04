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

func TestAllocationRepository_CRUD(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := NewAccountRepository(dbi.DB)
		repo := NewAllocationRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "alloc-crud@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Alloc Space")

		now := time.Now()
		account := &model.Account{
			ID:        uuid.NewString(),
			Name:      "Savings",
			SpaceID:   space.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, accountRepo.Create(account))

		target := decimal.NewFromInt(3000)
		alloc := &model.Allocation{
			ID:           uuid.NewString(),
			AccountID:    account.ID,
			Name:         "Emergency Fund",
			Amount:       decimal.NewFromInt(500),
			TargetAmount: &target,
			SortOrder:    0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		require.NoError(t, repo.Create(alloc))

		fetched, err := repo.ByID(alloc.ID)
		require.NoError(t, err)
		assert.Equal(t, "Emergency Fund", fetched.Name)
		assert.True(t, fetched.Amount.Equal(decimal.NewFromInt(500)))
		require.NotNil(t, fetched.TargetAmount)
		assert.True(t, fetched.TargetAmount.Equal(target))

		alloc2 := &model.Allocation{
			ID:        uuid.NewString(),
			AccountID: account.ID,
			Name:      "Trip",
			Amount:    decimal.NewFromInt(250),
			SortOrder: 1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, repo.Create(alloc2))

		list, err := repo.ByAccountID(account.ID)
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, "Emergency Fund", list[0].Name)
		assert.Equal(t, "Trip", list[1].Name)

		sum, err := repo.SumByAccountID(account.ID)
		require.NoError(t, err)
		assert.True(t, sum.Equal(decimal.NewFromInt(750)))

		require.NoError(t, repo.Update(alloc.ID, "Rainy Day", decimal.NewFromInt(800), nil))
		fetched, err = repo.ByID(alloc.ID)
		require.NoError(t, err)
		assert.Equal(t, "Rainy Day", fetched.Name)
		assert.True(t, fetched.Amount.Equal(decimal.NewFromInt(800)))
		assert.Nil(t, fetched.TargetAmount)

		require.NoError(t, repo.Delete(alloc2.ID))
		_, err = repo.ByID(alloc2.ID)
		assert.ErrorIs(t, err, ErrAllocationNotFound)

		assert.ErrorIs(t, repo.Delete(uuid.NewString()), ErrAllocationNotFound)
	})
}

func TestAllocationRepository_UniqueNamePerAccount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := NewAccountRepository(dbi.DB)
		repo := NewAllocationRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "alloc-unique@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Unique Space")

		now := time.Now()
		account := &model.Account{ID: uuid.NewString(), Name: "A", SpaceID: space.ID, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, accountRepo.Create(account))

		a := &model.Allocation{ID: uuid.NewString(), AccountID: account.ID, Name: "Goal", Amount: decimal.Zero, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, repo.Create(a))

		dup := &model.Allocation{ID: uuid.NewString(), AccountID: account.ID, Name: "Goal", Amount: decimal.Zero, CreatedAt: now, UpdatedAt: now}
		assert.Error(t, repo.Create(dup))
	})
}
