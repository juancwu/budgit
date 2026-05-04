package repository

import (
	"encoding/json"
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSpaceAuditLog(t *testing.T, repo SpaceAuditLogRepository, spaceID string, action model.SpaceAuditAction, actorID *string, metadata map[string]any, ts time.Time) *model.SpaceAuditLog {
	t.Helper()
	var meta []byte
	if metadata != nil {
		var err error
		meta, err = json.Marshal(metadata)
		require.NoError(t, err)
	}
	entry := &model.SpaceAuditLog{
		ID:        uuid.NewString(),
		SpaceID:   spaceID,
		ActorID:   actorID,
		Action:    action,
		Metadata:  meta,
		CreatedAt: ts,
	}
	require.NoError(t, repo.Create(entry))
	return entry
}

func TestSpaceAuditLogRepository_CreateAndList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceAuditLogRepository(dbi.DB)

		actor := testutil.CreateTestUserWithName(t, dbi.DB, "audit-actor@example.com", strPtr("Actor Name"))
		space := testutil.CreateTestSpace(t, dbi.DB, actor.ID, "Audit Space")

		base := time.Now().Add(-time.Hour)
		writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionRenamed, &actor.ID, map[string]any{"old_name": "A", "new_name": "B"}, base)
		writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionMemberInvited, &actor.ID, nil, base.Add(10*time.Minute))
		writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionDeleted, &actor.ID, map[string]any{"space_name": "Audit Space"}, base.Add(20*time.Minute))

		count, err := repo.CountBySpace(space.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		logs, err := repo.ListBySpace(space.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, logs, 3)

		// Newest first.
		assert.Equal(t, model.SpaceAuditActionDeleted, logs[0].Action)
		assert.Equal(t, model.SpaceAuditActionMemberInvited, logs[1].Action)
		assert.Equal(t, model.SpaceAuditActionRenamed, logs[2].Action)

		// Actor join populated.
		require.NotNil(t, logs[0].ActorName)
		assert.Equal(t, "Actor Name", *logs[0].ActorName)
	})
}

func TestSpaceAuditLogRepository_ListBySpace_Pagination(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceAuditLogRepository(dbi.DB)

		actor := testutil.CreateTestUser(t, dbi.DB, "page@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, actor.ID, "Paged Space")

		base := time.Now().Add(-time.Hour)
		for i := 0; i < 5; i++ {
			writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionRenamed, &actor.ID, nil, base.Add(time.Duration(i)*time.Minute))
		}

		page1, err := repo.ListBySpace(space.ID, 2, 0)
		require.NoError(t, err)
		require.Len(t, page1, 2)

		page2, err := repo.ListBySpace(space.ID, 2, 2)
		require.NoError(t, err)
		require.Len(t, page2, 2)

		// No overlap between pages.
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
		assert.NotEqual(t, page1[1].ID, page2[0].ID)
	})
}

func TestSpaceAuditLogRepository_ListAccountEvents_FiltersByMetadata(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceAuditLogRepository(dbi.DB)

		actor := testutil.CreateTestUser(t, dbi.DB, "acct-filter@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, actor.ID, "Filter Space")
		acct1 := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Account 1")
		acct2 := testutil.CreateTestAccount(t, dbi.DB, space.ID, "Account 2")

		base := time.Now().Add(-time.Hour)
		// Account 1 events
		writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionAccountCreated, &actor.ID, map[string]any{"account_id": acct1.ID, "account_name": "Account 1"}, base)
		writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionAccountRenamed, &actor.ID, map[string]any{"account_id": acct1.ID, "old_name": "Account 1", "new_name": "Renamed"}, base.Add(time.Minute))
		// Account 2 event
		writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionAccountCreated, &actor.ID, map[string]any{"account_id": acct2.ID, "account_name": "Account 2"}, base.Add(2*time.Minute))
		// Non-account event in same space — must NOT appear in account-scoped query
		writeSpaceAuditLog(t, repo, space.ID, model.SpaceAuditActionRenamed, &actor.ID, map[string]any{"old_name": "x", "new_name": "y"}, base.Add(3*time.Minute))

		acct1Logs, err := repo.ListAccountEvents(acct1.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, acct1Logs, 2)

		acct1Count, err := repo.CountAccountEvents(acct1.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, acct1Count)

		acct2Count, err := repo.CountAccountEvents(acct2.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, acct2Count)
	})
}

func strPtr(s string) *string { return &s }
