package repository

import (
	"encoding/json"
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTxAuditLog(t *testing.T, repo TransactionAuditLogRepository, transactionID string, action model.TransactionAuditAction, actorID *string, metadata map[string]any, ts time.Time) *model.TransactionAuditLog {
	t.Helper()
	var meta []byte
	if metadata != nil {
		var err error
		meta, err = json.Marshal(metadata)
		require.NoError(t, err)
	}
	entry := &model.TransactionAuditLog{
		ID:            uuid.NewString(),
		TransactionID: transactionID,
		ActorID:       actorID,
		Action:        action,
		Metadata:      meta,
		CreatedAt:     ts,
	}
	require.NoError(t, repo.Create(entry))
	return entry
}

func TestTransactionAuditLogRepository_CreateAndListByTransaction(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTransactionAuditLogRepository(dbi.DB)

		actor := testutil.CreateTestUserWithName(t, dbi.DB, "tx-audit@example.com", strPtr("Tx Actor"))
		space := testutil.CreateTestSpace(t, dbi.DB, actor.ID, "Tx Audit Space")
		account := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Acct")
		txn := testutil.CreateTestTransaction(t, dbi.DB, account.ID, "Coffee", model.TransactionTypeWithdrawal, decimal.NewFromInt(5))

		base := time.Now().Add(-time.Hour)
		writeTxAuditLog(t, repo, txn.ID, model.TransactionAuditActionCreated, &actor.ID,
			map[string]any{"account_id": account.ID, "transaction_type": "withdrawal", "title": "Coffee", "amount": "5.00"}, base)
		writeTxAuditLog(t, repo, txn.ID, model.TransactionAuditActionEdited, &actor.ID,
			map[string]any{"account_id": account.ID, "changes": map[string]any{"title": map[string]any{"old": "Coffee", "new": "Latte"}}}, base.Add(time.Minute))

		count, err := repo.CountByTransaction(txn.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		logs, err := repo.ListByTransaction(txn.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 2)
		// Newest first.
		assert.Equal(t, model.TransactionAuditActionEdited, logs[0].Action)
		require.NotNil(t, logs[0].ActorName)
		assert.Equal(t, "Tx Actor", *logs[0].ActorName)
	})
}

func TestTransactionAuditLogRepository_ListByAccount_LiveAndDeletedFallback(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTransactionAuditLogRepository(dbi.DB)

		actor := testutil.CreateTestUser(t, dbi.DB, "acct-list@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, actor.ID, "Acct List Space")
		account := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Acct")

		// Live transaction with audit entry
		live := testutil.CreateTestTransaction(t, dbi.DB, account.ID, "Live", model.TransactionTypeDeposit, decimal.NewFromInt(10))
		writeTxAuditLog(t, repo, live.ID, model.TransactionAuditActionCreated, &actor.ID,
			map[string]any{"account_id": account.ID}, time.Now().Add(-2*time.Minute))

		// Audit entry referencing a transaction that no longer exists.
		// Resolution must fall back to metadata.account_id.
		ghostID := uuid.NewString()
		writeTxAuditLog(t, repo, ghostID, model.TransactionAuditActionDeleted, &actor.ID,
			map[string]any{"account_id": account.ID, "title": "Ghost"}, time.Now().Add(-time.Minute))

		// Audit entry for a different account — must not appear.
		other := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Other")
		otherTxn := testutil.CreateTestTransaction(t, dbi.DB, other.ID, "Other", model.TransactionTypeDeposit, decimal.NewFromInt(1))
		writeTxAuditLog(t, repo, otherTxn.ID, model.TransactionAuditActionCreated, &actor.ID,
			map[string]any{"account_id": other.ID}, time.Now())

		count, err := repo.CountByAccount(account.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "should count live + ghost-via-metadata")

		logs, err := repo.ListByAccount(account.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 2)
		// Confirm both kinds present (one live, one ghost).
		ids := []string{logs[0].TransactionID, logs[1].TransactionID}
		assert.Contains(t, ids, live.ID)
		assert.Contains(t, ids, ghostID)
	})
}

func TestTransactionAuditLogRepository_ListBySpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTransactionAuditLogRepository(dbi.DB)

		actor := testutil.CreateTestUser(t, dbi.DB, "space-list@example.com", nil)

		// Two spaces, each with an account and a transaction.
		spaceA := testutil.CreateTestSpace(t, dbi.DB, actor.ID, "Space A")
		acctA := testutil.CreateTestAccount(t, dbi.DB, spaceA.ID, "Acct A")
		txnA := testutil.CreateTestTransaction(t, dbi.DB, acctA.ID, "txnA", model.TransactionTypeDeposit, decimal.NewFromInt(1))
		writeTxAuditLog(t, repo, txnA.ID, model.TransactionAuditActionCreated, &actor.ID,
			map[string]any{"account_id": acctA.ID}, time.Now().Add(-time.Minute))

		spaceB := testutil.CreateTestSpace(t, dbi.DB, actor.ID, "Space B")
		acctB := testutil.CreateTestAccount(t, dbi.DB, spaceB.ID, "Acct B")
		txnB := testutil.CreateTestTransaction(t, dbi.DB, acctB.ID, "txnB", model.TransactionTypeDeposit, decimal.NewFromInt(1))
		writeTxAuditLog(t, repo, txnB.ID, model.TransactionAuditActionCreated, &actor.ID,
			map[string]any{"account_id": acctB.ID}, time.Now())

		// Ghost in space A (deleted txn).
		ghostID := uuid.NewString()
		writeTxAuditLog(t, repo, ghostID, model.TransactionAuditActionDeleted, &actor.ID,
			map[string]any{"account_id": acctA.ID}, time.Now().Add(-30*time.Second))

		countA, err := repo.CountBySpace(spaceA.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, countA)

		countB, err := repo.CountBySpace(spaceB.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, countB)

		logsA, err := repo.ListBySpace(spaceA.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logsA, 2)
	})
}
