package service

import (
	"encoding/json"
	"errors"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSpaceAuditRepo struct {
	created []*model.SpaceAuditLog
	failNext error
}

func (f *fakeSpaceAuditRepo) Create(log *model.SpaceAuditLog) error {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	f.created = append(f.created, log)
	return nil
}
func (f *fakeSpaceAuditRepo) ListBySpace(string, int, int) ([]*model.SpaceAuditLogWithActor, error) {
	return nil, nil
}
func (f *fakeSpaceAuditRepo) CountBySpace(string) (int, error) { return 0, nil }
func (f *fakeSpaceAuditRepo) ListAccountEvents(string, int, int) ([]*model.SpaceAuditLogWithActor, error) {
	return nil, nil
}
func (f *fakeSpaceAuditRepo) CountAccountEvents(string) (int, error) { return 0, nil }

func TestSpaceAuditLogService_Record_PersistsEntry(t *testing.T) {
	repo := &fakeSpaceAuditRepo{}
	svc := NewSpaceAuditLogService(repo)

	svc.Record(RecordOptions{
		SpaceID:      "space-1",
		ActorID:      "actor-1",
		Action:       model.SpaceAuditActionRenamed,
		TargetUserID: "target-1",
		TargetEmail:  "target@example.com",
		Metadata:     map[string]any{"old_name": "A", "new_name": "B"},
	})

	require.Len(t, repo.created, 1)
	got := repo.created[0]
	assert.Equal(t, "space-1", got.SpaceID)
	require.NotNil(t, got.ActorID)
	assert.Equal(t, "actor-1", *got.ActorID)
	require.NotNil(t, got.TargetUserID)
	assert.Equal(t, "target-1", *got.TargetUserID)
	require.NotNil(t, got.TargetEmail)
	assert.Equal(t, "target@example.com", *got.TargetEmail)
	assert.Equal(t, model.SpaceAuditActionRenamed, got.Action)
	assert.NotEmpty(t, got.ID)
	assert.False(t, got.CreatedAt.IsZero())

	var meta map[string]any
	require.NoError(t, json.Unmarshal(got.Metadata, &meta))
	assert.Equal(t, "A", meta["old_name"])
	assert.Equal(t, "B", meta["new_name"])
}

func TestSpaceAuditLogService_Record_OmitsBlankOptionalFields(t *testing.T) {
	repo := &fakeSpaceAuditRepo{}
	svc := NewSpaceAuditLogService(repo)

	svc.Record(RecordOptions{
		SpaceID: "space-1",
		Action:  model.SpaceAuditActionDeleted,
	})

	require.Len(t, repo.created, 1)
	got := repo.created[0]
	assert.Nil(t, got.ActorID)
	assert.Nil(t, got.TargetUserID)
	assert.Nil(t, got.TargetEmail)
	assert.Empty(t, got.Metadata)
}

func TestSpaceAuditLogService_Record_SwallowsRepoError(t *testing.T) {
	// Audit failures must not bubble up to break the user's action.
	repo := &fakeSpaceAuditRepo{failNext: errors.New("boom")}
	svc := NewSpaceAuditLogService(repo)
	assert.NotPanics(t, func() {
		svc.Record(RecordOptions{SpaceID: "s", Action: model.SpaceAuditActionRenamed})
	})
	assert.Empty(t, repo.created)
}

func TestSpaceAuditLogService_Record_NilReceiverIsNoOp(t *testing.T) {
	var svc *SpaceAuditLogService
	assert.NotPanics(t, func() {
		svc.Record(RecordOptions{SpaceID: "s", Action: model.SpaceAuditActionRenamed})
	})
}
