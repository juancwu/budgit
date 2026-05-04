package service

import (
	"encoding/json"
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// txnFixture builds a fully wired TransactionService against a real DB along with
// the helper repos the tests need to inspect post-state.
type txnFixture struct {
	svc       *TransactionService
	txAudit   repository.TransactionAuditLogRepository
	accounts  repository.AccountRepository
	user      *model.User
	account   *model.Account
}

func newTxnFixture(t *testing.T, dbi testutil.DBInfo) *txnFixture {
	t.Helper()

	txnRepo := repository.NewTransactionRepository(dbi.DB)
	categoryRepo := repository.NewCategoryRepository(dbi.DB)
	accountRepo := repository.NewAccountRepository(dbi.DB)
	auditRepo := repository.NewTransactionAuditLogRepository(dbi.DB)

	accountSvc := NewAccountService(accountRepo)
	auditSvc := NewTransactionAuditLogService(auditRepo)
	svc := NewTransactionService(txnRepo, categoryRepo, accountSvc)
	svc.SetAuditLogger(auditSvc)

	user := testutil.CreateTestUser(t, dbi.DB, t.Name()+"@example.com", nil)
	space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")
	account := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Acct")

	return &txnFixture{
		svc:      svc,
		txAudit:  auditRepo,
		accounts: accountRepo,
		user:     user,
		account:  account,
	}
}

func TestTransactionService_Deposit_RecordsAuditAndUpdatesBalance(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)

		txn, err := f.svc.Deposit(DepositInput{
			AccountID:  f.account.ID,
			Title:      "Paycheck",
			Amount:     decimal.NewFromInt(100),
			OccurredAt: time.Now(),
			ActorID:    f.user.ID,
		})
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(100).Equal(txn.Value))

		updated, err := f.accounts.ByID(f.account.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(100).Equal(updated.Balance))

		logs, err := f.txAudit.ListByTransaction(txn.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Equal(t, model.TransactionAuditActionCreated, logs[0].Action)

		var meta map[string]any
		require.NoError(t, json.Unmarshal(logs[0].Metadata, &meta))
		assert.Equal(t, "deposit", meta["transaction_type"])
		assert.Equal(t, f.account.ID, meta["account_id"])
		assert.Equal(t, "Paycheck", meta["title"])
		assert.Equal(t, "100.00", meta["amount"])
	})
}

func TestTransactionService_PayBill_RecordsAuditAndDebitsBalance(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)

		// Seed some balance via deposit.
		_, err := f.svc.Deposit(DepositInput{
			AccountID: f.account.ID, Title: "seed", Amount: decimal.NewFromInt(50), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)

		txn, err := f.svc.PayBill(PayBillInput{
			AccountID:  f.account.ID,
			Title:      "Rent",
			Amount:     decimal.NewFromInt(20),
			OccurredAt: time.Now(),
			ActorID:    f.user.ID,
		})
		require.NoError(t, err)

		updated, err := f.accounts.ByID(f.account.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(30).Equal(updated.Balance))

		logs, err := f.txAudit.ListByTransaction(txn.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		var meta map[string]any
		require.NoError(t, json.Unmarshal(logs[0].Metadata, &meta))
		assert.Equal(t, "withdrawal", meta["transaction_type"])
	})
}

func TestTransactionService_UpdateDeposit_RebalancesAndDiffs(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)

		original, err := f.svc.Deposit(DepositInput{
			AccountID: f.account.ID, Title: "Old", Amount: decimal.NewFromInt(40),
			OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ActorID:    f.user.ID,
		})
		require.NoError(t, err)

		_, err = f.svc.UpdateDeposit(UpdateDepositInput{
			TransactionID: original.ID,
			Title:         "New",
			Amount:        decimal.NewFromInt(60), // +20 net
			OccurredAt:    time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
			ActorID:       f.user.ID,
		})
		require.NoError(t, err)

		// Balance reflects the swap (40 → 60 means +20 from 40 baseline).
		updated, err := f.accounts.ByID(f.account.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(60).Equal(updated.Balance))

		// 2 audit rows: created + edited (newest first).
		logs, err := f.txAudit.ListByTransaction(original.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 2)
		assert.Equal(t, model.TransactionAuditActionEdited, logs[0].Action)

		var meta struct {
			AccountID string                              `json:"account_id"`
			Changes   map[string]map[string]any           `json:"changes"`
		}
		require.NoError(t, json.Unmarshal(logs[0].Metadata, &meta))
		assert.Equal(t, f.account.ID, meta.AccountID)
		assert.Contains(t, meta.Changes, "title")
		assert.Equal(t, "Old", meta.Changes["title"]["old"])
		assert.Equal(t, "New", meta.Changes["title"]["new"])
		assert.Contains(t, meta.Changes, "amount")
		assert.Equal(t, "40.00", meta.Changes["amount"]["old"])
		assert.Equal(t, "60.00", meta.Changes["amount"]["new"])
		assert.Contains(t, meta.Changes, "occurred_at")
	})
}

func TestTransactionService_UpdateDeposit_NoChanges_NoAudit(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)

		original, err := f.svc.Deposit(DepositInput{
			AccountID: f.account.ID, Title: "Same", Amount: decimal.NewFromInt(10),
			OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ActorID:    f.user.ID,
		})
		require.NoError(t, err)

		// Update with identical values.
		_, err = f.svc.UpdateDeposit(UpdateDepositInput{
			TransactionID: original.ID,
			Title:         "Same",
			Amount:        decimal.NewFromInt(10),
			OccurredAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ActorID:       f.user.ID,
		})
		require.NoError(t, err)

		// Only the original `created` audit row exists; no `edited` row.
		count, err := f.txAudit.CountByTransaction(original.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestTransactionService_UpdateBill_RebalancesAndDiffs(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)

		// Seed funds and then a bill.
		_, err := f.svc.Deposit(DepositInput{
			AccountID: f.account.ID, Title: "seed", Amount: decimal.NewFromInt(100), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)
		bill, err := f.svc.PayBill(PayBillInput{
			AccountID: f.account.ID, Title: "Cable", Amount: decimal.NewFromInt(30), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)

		_, err = f.svc.UpdateBill(UpdateBillInput{
			TransactionID: bill.ID,
			Title:         "Internet",
			Amount:        decimal.NewFromInt(40), // -10 vs original
			OccurredAt:    bill.OccurredAt,
			ActorID:       f.user.ID,
		})
		require.NoError(t, err)

		// 100 - 40 = 60.
		updated, err := f.accounts.ByID(f.account.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(60).Equal(updated.Balance))

		logs, err := f.txAudit.ListByTransaction(bill.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 2)
		assert.Equal(t, model.TransactionAuditActionEdited, logs[0].Action)
	})
}

func TestTransactionService_UpdateDeposit_RejectsBillTransaction(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		_, err := f.svc.Deposit(DepositInput{
			AccountID: f.account.ID, Title: "seed", Amount: decimal.NewFromInt(50), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)
		bill, err := f.svc.PayBill(PayBillInput{
			AccountID: f.account.ID, Title: "Bill", Amount: decimal.NewFromInt(10), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)

		_, err = f.svc.UpdateDeposit(UpdateDepositInput{
			TransactionID: bill.ID,
			Title:         "x",
			Amount:        decimal.NewFromInt(1),
			OccurredAt:    time.Now(),
			ActorID:       f.user.ID,
		})
		require.Error(t, err)
	})
}

func TestTransactionService_Validations(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)

		_, err := f.svc.Deposit(DepositInput{AccountID: f.account.ID, Amount: decimal.NewFromInt(1), OccurredAt: time.Now()})
		assert.Error(t, err, "blank title")

		_, err = f.svc.Deposit(DepositInput{AccountID: f.account.ID, Title: "x", Amount: decimal.NewFromInt(0), OccurredAt: time.Now()})
		assert.Error(t, err, "zero amount")

		_, err = f.svc.Deposit(DepositInput{AccountID: f.account.ID, Title: "x", Amount: decimal.NewFromInt(1)})
		assert.Error(t, err, "missing date")

		_, err = f.svc.PayBill(PayBillInput{Title: "x", Amount: decimal.NewFromInt(1), OccurredAt: time.Now()})
		assert.Error(t, err, "missing account id")
	})
}
