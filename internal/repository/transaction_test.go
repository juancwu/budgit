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

func TestTransactionRepository_TransferAtomic_LinksPairAndUpdatesBalances(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTransactionRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "transfer-repo@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")
		src := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Src")
		dst := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Dst")

		now := time.Now()
		withdrawal := &model.Transaction{
			ID: uuid.NewString(), Value: decimal.NewFromInt(40), Type: model.TransactionTypeWithdrawal,
			AccountID: src.ID, Title: "Move", OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		deposit := &model.Transaction{
			ID: uuid.NewString(), Value: decimal.NewFromInt(40), Type: model.TransactionTypeDeposit,
			AccountID: dst.ID, Title: "Move", OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}

		err := repo.TransferAtomic(withdrawal, deposit, decimal.NewFromInt(-40), decimal.NewFromInt(40))
		require.NoError(t, err)

		// Both transactions exist.
		w, err := repo.GetByID(withdrawal.ID)
		require.NoError(t, err)
		assert.Equal(t, model.TransactionTypeWithdrawal, w.Type)
		d, err := repo.GetByID(deposit.ID)
		require.NoError(t, err)
		assert.Equal(t, model.TransactionTypeDeposit, d.Type)

		// Balances were applied.
		accountRepo := NewAccountRepository(dbi.DB)
		srcAfter, err := accountRepo.ByID(src.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(-40).Equal(srcAfter.Balance))
		dstAfter, err := accountRepo.ByID(dst.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(40).Equal(dstAfter.Balance))

		// Linked both ways via related_transactions.
		other, err := repo.GetRelatedID(withdrawal.ID)
		require.NoError(t, err)
		require.NotNil(t, other)
		assert.Equal(t, deposit.ID, *other)

		other, err = repo.GetRelatedID(deposit.ID)
		require.NoError(t, err)
		require.NotNil(t, other)
		assert.Equal(t, withdrawal.ID, *other)
	})
}

func TestTransactionRepository_TransferIDsIn_ReturnsOnlyTransferHalves(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTransactionRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "transferids@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")
		src := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Src")
		dst := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Dst")

		// One transfer pair (source `w`, deposit `d`) plus one standalone txn.
		now := time.Now()
		w := &model.Transaction{ID: uuid.NewString(), Value: decimal.NewFromInt(5), Type: model.TransactionTypeWithdrawal, AccountID: src.ID, Title: "T-w", OccurredAt: now, CreatedAt: now, UpdatedAt: now}
		d := &model.Transaction{ID: uuid.NewString(), Value: decimal.NewFromInt(5), Type: model.TransactionTypeDeposit, AccountID: dst.ID, Title: "T-d", OccurredAt: now, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, repo.TransferAtomic(w, d, decimal.NewFromInt(-5), decimal.NewFromInt(5)))
		standalone := testutil.CreateTestTransaction(t, dbi.DB, src.ID, "solo", model.TransactionTypeDeposit, decimal.NewFromInt(1))

		hits, err := repo.TransferIDsIn([]string{w.ID, d.ID, standalone.ID})
		require.NoError(t, err)
		assert.True(t, hits[w.ID], "withdrawal half should be flagged")
		assert.True(t, hits[d.ID], "deposit half should be flagged")
		assert.False(t, hits[standalone.ID], "standalone transaction should not be flagged")
	})
}

func TestTransactionRepository_TransferIDsIn_EmptyInput(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTransactionRepository(dbi.DB)
		hits, err := repo.TransferIDsIn(nil)
		require.NoError(t, err)
		assert.Empty(t, hits)
	})
}

func TestTransactionRepository_GetRelatedID_NoneWhenStandalone(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTransactionRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "standalone@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")
		acct := testutil.CreateTestAccount(t, dbi.DB, space.ID, "A")
		txn := testutil.CreateTestTransaction(t, dbi.DB, acct.ID, "x", model.TransactionTypeDeposit, decimal.NewFromInt(1))

		other, err := repo.GetRelatedID(txn.ID)
		require.NoError(t, err)
		assert.Nil(t, other)
	})
}
