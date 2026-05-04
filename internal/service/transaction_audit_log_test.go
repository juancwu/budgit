package service

import (
	"encoding/json"
	"errors"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTxAuditRepo struct {
	created  []*model.TransactionAuditLog
	failNext error
}

func (f *fakeTxAuditRepo) Create(log *model.TransactionAuditLog) error {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	f.created = append(f.created, log)
	return nil
}
func (f *fakeTxAuditRepo) ListByTransaction(string, int, int) ([]*model.TransactionAuditLogWithActor, error) {
	return nil, nil
}
func (f *fakeTxAuditRepo) CountByTransaction(string) (int, error) { return 0, nil }
func (f *fakeTxAuditRepo) ListByAccount(string, int, int) ([]*model.TransactionAuditLogWithActor, error) {
	return nil, nil
}
func (f *fakeTxAuditRepo) CountByAccount(string) (int, error) { return 0, nil }
func (f *fakeTxAuditRepo) ListBySpace(string, int, int) ([]*model.TransactionAuditLogWithActor, error) {
	return nil, nil
}
func (f *fakeTxAuditRepo) CountBySpace(string) (int, error) { return 0, nil }

func TestTransactionAuditLogService_Record_PersistsEntry(t *testing.T) {
	repo := &fakeTxAuditRepo{}
	svc := NewTransactionAuditLogService(repo)

	svc.Record(TransactionRecordOptions{
		TransactionID: "txn-1",
		ActorID:       "actor-1",
		Action:        model.TransactionAuditActionEdited,
		Metadata:      map[string]any{"changes": map[string]any{"title": "x"}},
	})

	require.Len(t, repo.created, 1)
	got := repo.created[0]
	assert.Equal(t, "txn-1", got.TransactionID)
	require.NotNil(t, got.ActorID)
	assert.Equal(t, "actor-1", *got.ActorID)
	assert.Equal(t, model.TransactionAuditActionEdited, got.Action)

	var meta map[string]any
	require.NoError(t, json.Unmarshal(got.Metadata, &meta))
	assert.Contains(t, meta, "changes")
}

func TestTransactionAuditLogService_Record_SwallowsRepoError(t *testing.T) {
	repo := &fakeTxAuditRepo{failNext: errors.New("boom")}
	svc := NewTransactionAuditLogService(repo)
	assert.NotPanics(t, func() {
		svc.Record(TransactionRecordOptions{TransactionID: "x", Action: model.TransactionAuditActionEdited})
	})
}

func TestTransactionAuditLogService_Record_NilReceiverIsNoOp(t *testing.T) {
	var svc *TransactionAuditLogService
	assert.NotPanics(t, func() {
		svc.Record(TransactionRecordOptions{TransactionID: "x", Action: model.TransactionAuditActionEdited})
	})
}
