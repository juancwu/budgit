package service

import (
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoneyAccountService_CreateAccount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-create@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc Space")

		account, err := svc.CreateAccount(CreateMoneyAccountDTO{
			SpaceID:   space.ID,
			Name:      "Savings",
			CreatedBy: user.ID,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, account.ID)
		assert.Equal(t, "Savings", account.Name)
		assert.Equal(t, space.ID, account.SpaceID)
	})
}

func TestMoneyAccountService_CreateAccount_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		account, err := svc.CreateAccount(CreateMoneyAccountDTO{
			SpaceID:   "some-space",
			Name:      "",
			CreatedBy: "some-user",
		})
		assert.Error(t, err)
		assert.Nil(t, account)
	})
}

func TestMoneyAccountService_GetAccountsForSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-list@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc List Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Checking", user.ID)
		testutil.CreateTestTransfer(t, dbi.DB, account.ID, 5000, model.TransferDirectionDeposit, user.ID)

		accounts, err := svc.GetAccountsForSpace(space.ID)
		require.NoError(t, err)
		require.Len(t, accounts, 1)
		assert.Equal(t, "Checking", accounts[0].Name)
		assert.Equal(t, 5000, accounts[0].BalanceCents)
	})
}

func TestMoneyAccountService_CreateTransfer_Deposit(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-deposit@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc Deposit Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Deposit Account", user.ID)

		transfer, err := svc.CreateTransfer(CreateTransferDTO{
			AccountID: account.ID,
			Amount:    3000,
			Direction: model.TransferDirectionDeposit,
			Note:      "Initial deposit",
			CreatedBy: user.ID,
		}, 10000)
		require.NoError(t, err)
		assert.NotEmpty(t, transfer.ID)
		assert.Equal(t, 3000, transfer.AmountCents)
		assert.Equal(t, model.TransferDirectionDeposit, transfer.Direction)
	})
}

func TestMoneyAccountService_CreateTransfer_InsufficientBalance(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-insuf@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc Insuf Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Insuf Account", user.ID)

		transfer, err := svc.CreateTransfer(CreateTransferDTO{
			AccountID: account.ID,
			Amount:    5000,
			Direction: model.TransferDirectionDeposit,
			Note:      "Too much",
			CreatedBy: user.ID,
		}, 1000)
		assert.Error(t, err)
		assert.Nil(t, transfer)
	})
}

func TestMoneyAccountService_CreateTransfer_Withdrawal(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-withdraw@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc Withdraw Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Withdraw Account", user.ID)
		testutil.CreateTestTransfer(t, dbi.DB, account.ID, 5000, model.TransferDirectionDeposit, user.ID)

		transfer, err := svc.CreateTransfer(CreateTransferDTO{
			AccountID: account.ID,
			Amount:    2000,
			Direction: model.TransferDirectionWithdrawal,
			Note:      "Withdrawal",
			CreatedBy: user.ID,
		}, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, transfer.ID)
		assert.Equal(t, 2000, transfer.AmountCents)
		assert.Equal(t, model.TransferDirectionWithdrawal, transfer.Direction)
	})
}

func TestMoneyAccountService_GetTotalAllocatedForSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-total@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc Total Space")

		account1 := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Account 1", user.ID)
		testutil.CreateTestTransfer(t, dbi.DB, account1.ID, 3000, model.TransferDirectionDeposit, user.ID)

		account2 := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Account 2", user.ID)
		testutil.CreateTestTransfer(t, dbi.DB, account2.ID, 2000, model.TransferDirectionDeposit, user.ID)

		total, err := svc.GetTotalAllocatedForSpace(space.ID)
		require.NoError(t, err)
		assert.Equal(t, 5000, total)
	})
}

func TestMoneyAccountService_DeleteAccount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-del@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc Del Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "Doomed Account", user.ID)

		err := svc.DeleteAccount(account.ID)
		require.NoError(t, err)

		accounts, err := svc.GetAccountsForSpace(space.ID)
		require.NoError(t, err)
		assert.Empty(t, accounts)
	})
}

func TestMoneyAccountService_DeleteTransfer(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
		svc := NewMoneyAccountService(accountRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-svc-deltx@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Account Svc DelTx Space")
		account := testutil.CreateTestMoneyAccount(t, dbi.DB, space.ID, "DelTx Account", user.ID)
		transfer := testutil.CreateTestTransfer(t, dbi.DB, account.ID, 1000, model.TransferDirectionDeposit, user.ID)

		err := svc.DeleteTransfer(transfer.ID)
		require.NoError(t, err)

		transfers, err := svc.GetTransfersForAccount(account.ID)
		require.NoError(t, err)
		assert.Empty(t, transfers)
	})
}
