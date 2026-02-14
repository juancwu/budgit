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

func TestMoneyAccountRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		now := time.Now()
		account := &model.MoneyAccount{
			ID:        uuid.NewString(),
			SpaceID:   space.ID,
			Name:      "Savings",
			CreatedBy: user.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(account)
		require.NoError(t, err)

		fetched, err := repo.GetByID(account.ID)
		require.NoError(t, err)
		assert.Equal(t, account.ID, fetched.ID)
		assert.Equal(t, space.ID, fetched.SpaceID)
		assert.Equal(t, "Savings", fetched.Name)
		assert.Equal(t, user.ID, fetched.CreatedBy)
	})
}

func TestMoneyAccountRepository_GetBySpaceID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Account A", user.ID)
		testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Account B", user.ID)

		accounts, err := repo.GetBySpaceID(space.ID)
		require.NoError(t, err)
		assert.Len(t, accounts, 2)
	})
}

func TestMoneyAccountRepository_Update(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Original", user.ID)

		account.Name = "Renamed"
		err := repo.Update(account)
		require.NoError(t, err)

		fetched, err := repo.GetByID(account.ID)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", fetched.Name)
	})
}

func TestMoneyAccountRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "To Delete", user.ID)

		err := repo.Delete(account.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(account.ID)
		assert.ErrorIs(t, err, ErrMoneyAccountNotFound)
	})
}

func TestMoneyAccountRepository_CreateTransfer(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Checking", user.ID)

		transfer := &model.AccountTransfer{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			AmountCents: 5000,
			Direction:   model.TransferDirectionDeposit,
			Note:        "Initial deposit",
			CreatedBy:   user.ID,
			CreatedAt:   time.Now(),
		}

		err := repo.CreateTransfer(transfer)
		require.NoError(t, err)

		transfers, err := repo.GetTransfersByAccountID(account.ID)
		require.NoError(t, err)
		require.Len(t, transfers, 1)
		assert.Equal(t, transfer.ID, transfers[0].ID)
		assert.Equal(t, 5000, transfers[0].AmountCents)
		assert.Equal(t, model.TransferDirectionDeposit, transfers[0].Direction)
	})
}

func TestMoneyAccountRepository_DeleteTransfer(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Checking", user.ID)
		transfer := testutil.CreateTestTransfer(t, dbi.DB, account.ID, 1000, model.TransferDirectionDeposit, user.ID)

		err := repo.DeleteTransfer(transfer.ID)
		require.NoError(t, err)

		transfers, err := repo.GetTransfersByAccountID(account.ID)
		require.NoError(t, err)
		assert.Empty(t, transfers)
	})
}

func TestMoneyAccountRepository_GetAccountBalance(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Checking", user.ID)

		testutil.CreateTestTransfer(t, dbi.DB, account.ID, 1000, model.TransferDirectionDeposit, user.ID)
		testutil.CreateTestTransfer(t, dbi.DB, account.ID, 300, model.TransferDirectionWithdrawal, user.ID)

		balance, err := repo.GetAccountBalance(account.ID)
		require.NoError(t, err)
		assert.Equal(t, 700, balance)
	})
}

func TestMoneyAccountRepository_GetTotalAllocatedForSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewMoneyAccountRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		account1 := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Account A", user.ID)
		account2 := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Account B", user.ID)

		testutil.CreateTestTransfer(t, dbi.DB, account1.ID, 2000, model.TransferDirectionDeposit, user.ID)
		testutil.CreateTestTransfer(t, dbi.DB, account2.ID, 3000, model.TransferDirectionDeposit, user.ID)

		total, err := repo.GetTotalAllocatedForSpace(space.ID)
		require.NoError(t, err)
		assert.Equal(t, 5000, total)
	})
}
