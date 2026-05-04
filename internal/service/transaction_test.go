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

func TestTransactionService_Transfer_HappyPath(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		dest := testutil.CreateTestAccount(t, dbi.DB, f.account.SpaceID, "Savings")

		result, err := f.svc.Transfer(TransferInput{
			SourceAccountID: f.account.ID,
			DestAccountID:   dest.ID,
			Title:           "Move to savings",
			Amount:          decimal.NewFromInt(50),
			OccurredAt:      time.Now(),
			ActorID:         f.user.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, result.Withdrawal)
		require.NotNil(t, result.Deposit)
		assert.Equal(t, model.TransactionTypeWithdrawal, result.Withdrawal.Type)
		assert.Equal(t, model.TransactionTypeDeposit, result.Deposit.Type)
		assert.Equal(t, f.account.ID, result.Withdrawal.AccountID)
		assert.Equal(t, dest.ID, result.Deposit.AccountID)

		// Source went from 0 → -50 (overdraft allowed); dest 0 → +50.
		src, err := f.accounts.ByID(f.account.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(-50).Equal(src.Balance))
		dst, err := f.accounts.ByID(dest.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(50).Equal(dst.Balance))

		// Transactions are linked.
		relatedID, err := f.svc.GetRelatedTransactionID(result.Withdrawal.ID)
		require.NoError(t, err)
		assert.Equal(t, result.Deposit.ID, relatedID)

		// Audit recorded both sides with the right transfer_role and other-account name.
		wlogs, err := f.txAudit.ListByTransaction(result.Withdrawal.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, wlogs, 1)
		var wmeta map[string]any
		require.NoError(t, json.Unmarshal(wlogs[0].Metadata, &wmeta))
		assert.Equal(t, "source", wmeta["transfer_role"])
		assert.Equal(t, result.Deposit.ID, wmeta["transfer_pair_id"])
		assert.Equal(t, "Savings", wmeta["transfer_other_name"])

		dlogs, err := f.txAudit.ListByTransaction(result.Deposit.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, dlogs, 1)
		var dmeta map[string]any
		require.NoError(t, json.Unmarshal(dlogs[0].Metadata, &dmeta))
		assert.Equal(t, "destination", dmeta["transfer_role"])
		assert.Equal(t, result.Withdrawal.ID, dmeta["transfer_pair_id"])
		assert.Equal(t, "Acct", dmeta["transfer_other_name"])
	})
}

func TestTransactionService_Transfer_AllowsOverdraft(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		dest := testutil.CreateTestAccount(t, dbi.DB, f.account.SpaceID, "B")

		// Seed source to 100, then transfer 200 → -100.
		_, err := f.svc.Deposit(DepositInput{
			AccountID: f.account.ID, Title: "seed", Amount: decimal.NewFromInt(100),
			OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)
		_, err = f.svc.Transfer(TransferInput{
			SourceAccountID: f.account.ID, DestAccountID: dest.ID,
			Title: "T1", Amount: decimal.NewFromInt(200), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)

		src, err := f.accounts.ByID(f.account.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(-100).Equal(src.Balance), "expected -100, got %s", src.Balance.String())

		// Transfer another 200 from -100 → -300.
		_, err = f.svc.Transfer(TransferInput{
			SourceAccountID: f.account.ID, DestAccountID: dest.ID,
			Title: "T2", Amount: decimal.NewFromInt(200), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		require.NoError(t, err)

		src, err = f.accounts.ByID(f.account.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(-300).Equal(src.Balance), "expected -300, got %s", src.Balance.String())
	})
}

// TestTransactionService_Transfer_AppearsInAccountActivityFeeds is the regression
// test for "make sure activity logs are also created". The activity views on each
// account page and the space-level page merge transaction_audit_logs into a unified
// feed; this test exercises the full path end-to-end so silent gaps in audit
// recording or merging would fail.
func TestTransactionService_Transfer_AppearsInActivityFeeds(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		dest := testutil.CreateTestAccount(t, dbi.DB, f.account.SpaceID, "Savings")

		spaceAuditRepo := repository.NewSpaceAuditLogRepository(dbi.DB)
		activitySvc := NewAccountActivityService(
			NewSpaceAuditLogService(spaceAuditRepo),
			NewTransactionAuditLogService(f.txAudit),
		)

		result, err := f.svc.Transfer(TransferInput{
			SourceAccountID: f.account.ID,
			DestAccountID:   dest.ID,
			Title:           "Move to savings",
			Amount:          decimal.NewFromInt(75),
			OccurredAt:      time.Now(),
			ActorID:         f.user.ID,
		})
		require.NoError(t, err)

		// Source account activity sees the withdrawal half.
		srcRows, err := activitySvc.List(f.account.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, srcRows, 1, "source account feed should include the withdrawal half")
		require.NotNil(t, srcRows[0].TxLog)
		assert.Equal(t, result.Withdrawal.ID, srcRows[0].TxLog.TransactionID)
		assertTransferRole(t, srcRows[0].TxLog, "source", dest.ID)

		// Destination account activity sees the deposit half.
		dstRows, err := activitySvc.List(dest.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, dstRows, 1, "destination account feed should include the deposit half")
		require.NotNil(t, dstRows[0].TxLog)
		assert.Equal(t, result.Deposit.ID, dstRows[0].TxLog.TransactionID)
		assertTransferRole(t, dstRows[0].TxLog, "destination", f.account.ID)

		// Space-level activity feed sees both halves.
		spaceRows, err := activitySvc.ListSpace(f.account.SpaceID, 10, 0)
		require.NoError(t, err)
		require.Len(t, spaceRows, 2, "space feed should include both halves of the transfer")
		ids := []string{}
		for _, r := range spaceRows {
			require.NotNil(t, r.TxLog)
			ids = append(ids, r.TxLog.TransactionID)
		}
		assert.Contains(t, ids, result.Withdrawal.ID)
		assert.Contains(t, ids, result.Deposit.ID)

		// Counts agree with what the feed returns (pagination relies on this).
		srcCount, err := activitySvc.Count(f.account.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, srcCount)
		dstCount, err := activitySvc.Count(dest.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, dstCount)
		spaceCount, err := activitySvc.CountSpace(f.account.SpaceID)
		require.NoError(t, err)
		// Source account had no activity before the transfer, dest is brand-new;
		// the only activity in the space is the two transfer halves.
		assert.Equal(t, 2, spaceCount)
	})
}

func assertTransferRole(t *testing.T, log *model.TransactionAuditLogWithActor, expectedRole, expectedOtherAcctID string) {
	t.Helper()
	var meta map[string]any
	require.NoError(t, json.Unmarshal(log.Metadata, &meta))
	assert.Equal(t, expectedRole, meta["transfer_role"])
	assert.Equal(t, expectedOtherAcctID, meta["transfer_other_acct"])
}

func TestTransactionService_Transfer_RejectsSameAccount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		_, err := f.svc.Transfer(TransferInput{
			SourceAccountID: f.account.ID,
			DestAccountID:   f.account.ID,
			Title:           "Self",
			Amount:          decimal.NewFromInt(10),
			OccurredAt:      time.Now(),
			ActorID:         f.user.ID,
		})
		assert.Error(t, err)
	})
}

func TestTransactionService_Transfer_RejectsNonPositiveAmount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		dest := testutil.CreateTestAccount(t, dbi.DB, f.account.SpaceID, "B")
		_, err := f.svc.Transfer(TransferInput{
			SourceAccountID: f.account.ID, DestAccountID: dest.ID,
			Title: "x", Amount: decimal.NewFromInt(0), OccurredAt: time.Now(), ActorID: f.user.ID,
		})
		assert.Error(t, err)
	})
}

func TestTransactionService_Update_RejectsTransferTransactions(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		f := newTxnFixture(t, dbi)
		dest := testutil.CreateTestAccount(t, dbi.DB, f.account.SpaceID, "Savings")

		result, err := f.svc.Transfer(TransferInput{
			SourceAccountID: f.account.ID,
			DestAccountID:   dest.ID,
			Title:           "Initial",
			Amount:          decimal.NewFromInt(20),
			OccurredAt:      time.Now(),
			ActorID:         f.user.ID,
		})
		require.NoError(t, err)

		// The withdrawal half cannot be edited via UpdateBill.
		_, err = f.svc.UpdateBill(UpdateBillInput{
			TransactionID: result.Withdrawal.ID,
			Title:         "tampered",
			Amount:        decimal.NewFromInt(99),
			OccurredAt:    time.Now(),
			ActorID:       f.user.ID,
		})
		require.ErrorIs(t, err, ErrTransactionPartOfTransfer)

		// The deposit half cannot be edited via UpdateDeposit.
		_, err = f.svc.UpdateDeposit(UpdateDepositInput{
			TransactionID: result.Deposit.ID,
			Title:         "tampered",
			Amount:        decimal.NewFromInt(99),
			OccurredAt:    time.Now(),
			ActorID:       f.user.ID,
		})
		require.ErrorIs(t, err, ErrTransactionPartOfTransfer)

		// Underlying transaction is untouched (no audit `edited` row added either).
		count, err := f.txAudit.CountByTransaction(result.Withdrawal.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "only the original `created` audit row should exist")
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
