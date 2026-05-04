package service

import (
	"encoding/json"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountService_CreateAccount_RecordsAudit(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewAccountRepository(dbi.DB)
		auditRepo := repository.NewSpaceAuditLogRepository(dbi.DB)
		auditSvc := NewSpaceAuditLogService(auditRepo)
		svc := NewAccountService(accountRepo)
		svc.SetAuditLogger(auditSvc)

		user := testutil.CreateTestUser(t, dbi.DB, "acct-create-audit@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")

		account, err := svc.CreateAccount(space.ID, "Checking", user.ID)
		require.NoError(t, err)

		logs, err := auditRepo.ListAccountEvents(account.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Equal(t, model.SpaceAuditActionAccountCreated, logs[0].Action)

		var meta map[string]any
		require.NoError(t, json.Unmarshal(logs[0].Metadata, &meta))
		assert.Equal(t, account.ID, meta["account_id"])
		assert.Equal(t, "Checking", meta["account_name"])
	})
}

func TestAccountService_RenameAccount_RecordsAuditOnlyWhenChanged(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewAccountRepository(dbi.DB)
		auditRepo := repository.NewSpaceAuditLogRepository(dbi.DB)
		svc := NewAccountService(accountRepo)
		svc.SetAuditLogger(NewSpaceAuditLogService(auditRepo))

		user := testutil.CreateTestUser(t, dbi.DB, "acct-rename-audit@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")
		account := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Old")

		// Rename to a new value records an audit row.
		require.NoError(t, svc.RenameAccount(account.ID, "New", user.ID))

		// Renaming to the same value does not.
		require.NoError(t, svc.RenameAccount(account.ID, "New", user.ID))

		count, err := auditRepo.CountAccountEvents(account.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		logs, err := auditRepo.ListAccountEvents(account.ID, 10, 0)
		require.NoError(t, err)
		var meta map[string]any
		require.NoError(t, json.Unmarshal(logs[0].Metadata, &meta))
		assert.Equal(t, "Old", meta["old_name"])
		assert.Equal(t, "New", meta["new_name"])
	})
}

func TestAccountService_DeleteAccount_RecordsAuditBeforeDelete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		accountRepo := repository.NewAccountRepository(dbi.DB)
		auditRepo := repository.NewSpaceAuditLogRepository(dbi.DB)
		svc := NewAccountService(accountRepo)
		svc.SetAuditLogger(NewSpaceAuditLogService(auditRepo))

		user := testutil.CreateTestUser(t, dbi.DB, "acct-delete-audit@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")
		account := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Target")

		require.NoError(t, svc.DeleteAccount(account.ID, user.ID))

		// Account is gone.
		_, err := accountRepo.ByID(account.ID)
		require.Error(t, err)

		// Audit row still exists and captured the pre-delete name (no FK on metadata).
		logs, err := auditRepo.ListAccountEvents(account.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Equal(t, model.SpaceAuditActionAccountDeleted, logs[0].Action)
		var meta map[string]any
		require.NoError(t, json.Unmarshal(logs[0].Metadata, &meta))
		assert.Equal(t, "Target", meta["account_name"])
	})
}

func TestAccountService_NoAuditLoggerSet_DoesNotPanic(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		// SetAuditLogger is intentionally optional so existing tests/callers that
		// don't care about audit don't have to wire it.
		svc := NewAccountService(repository.NewAccountRepository(dbi.DB))

		user := testutil.CreateTestUser(t, dbi.DB, "no-audit@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "S")

		account, err := svc.CreateAccount(space.ID, "x", user.ID)
		require.NoError(t, err)
		require.NoError(t, svc.RenameAccount(account.ID, "y", user.ID))
		require.NoError(t, svc.DeleteAccount(account.ID, user.ID))
	})
}
