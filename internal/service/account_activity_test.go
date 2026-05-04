package service

import (
	"errors"
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSpaceAuditRepo serves canned responses for the activity-merger tests so we can
// focus on the merge/sort/pagination logic without a real DB.
type stubSpaceAuditRepo struct {
	listAccount  []*model.SpaceAuditLogWithActor
	countAccount int
	listSpace    []*model.SpaceAuditLogWithActor
	countSpace   int
	err          error
}

func (s *stubSpaceAuditRepo) Create(*model.SpaceAuditLog) error { return nil }
func (s *stubSpaceAuditRepo) ListBySpace(_ string, limit, _ int) ([]*model.SpaceAuditLogWithActor, error) {
	if s.err != nil {
		return nil, s.err
	}
	return firstN(s.listSpace, limit), nil
}
func (s *stubSpaceAuditRepo) CountBySpace(string) (int, error)   { return s.countSpace, s.err }
func (s *stubSpaceAuditRepo) ListAccountEvents(_ string, limit, _ int) ([]*model.SpaceAuditLogWithActor, error) {
	if s.err != nil {
		return nil, s.err
	}
	return firstN(s.listAccount, limit), nil
}
func (s *stubSpaceAuditRepo) CountAccountEvents(string) (int, error) { return s.countAccount, s.err }

type stubTxAuditRepo struct {
	listAccount  []*model.TransactionAuditLogWithActor
	countAccount int
	listSpace    []*model.TransactionAuditLogWithActor
	countSpace   int
	err          error
}

func (s *stubTxAuditRepo) Create(*model.TransactionAuditLog) error { return nil }
func (s *stubTxAuditRepo) ListByTransaction(string, int, int) ([]*model.TransactionAuditLogWithActor, error) {
	return nil, nil
}
func (s *stubTxAuditRepo) CountByTransaction(string) (int, error) { return 0, nil }
func (s *stubTxAuditRepo) ListByAccount(_ string, limit, _ int) ([]*model.TransactionAuditLogWithActor, error) {
	if s.err != nil {
		return nil, s.err
	}
	return firstNTx(s.listAccount, limit), nil
}
func (s *stubTxAuditRepo) CountByAccount(string) (int, error) { return s.countAccount, s.err }
func (s *stubTxAuditRepo) ListBySpace(_ string, limit, _ int) ([]*model.TransactionAuditLogWithActor, error) {
	if s.err != nil {
		return nil, s.err
	}
	return firstNTx(s.listSpace, limit), nil
}
func (s *stubTxAuditRepo) CountBySpace(string) (int, error) { return s.countSpace, s.err }

func firstN(s []*model.SpaceAuditLogWithActor, n int) []*model.SpaceAuditLogWithActor {
	if n >= len(s) {
		return s
	}
	return s[:n]
}
func firstNTx(s []*model.TransactionAuditLogWithActor, n int) []*model.TransactionAuditLogWithActor {
	if n >= len(s) {
		return s
	}
	return s[:n]
}

func spaceLog(action model.SpaceAuditAction, ts time.Time) *model.SpaceAuditLogWithActor {
	return &model.SpaceAuditLogWithActor{
		SpaceAuditLog: model.SpaceAuditLog{Action: action, CreatedAt: ts},
	}
}
func txLog(action model.TransactionAuditAction, ts time.Time) *model.TransactionAuditLogWithActor {
	return &model.TransactionAuditLogWithActor{
		TransactionAuditLog: model.TransactionAuditLog{Action: action, CreatedAt: ts},
	}
}

func TestAccountActivityService_List_MergesAndSortsByTimestamp(t *testing.T) {
	now := time.Now()
	spaceRepo := &stubSpaceAuditRepo{
		listAccount: []*model.SpaceAuditLogWithActor{
			spaceLog(model.SpaceAuditActionAccountRenamed, now.Add(-1*time.Minute)),
			spaceLog(model.SpaceAuditActionAccountCreated, now.Add(-10*time.Minute)),
		},
		countAccount: 2,
	}
	txRepo := &stubTxAuditRepo{
		listAccount: []*model.TransactionAuditLogWithActor{
			txLog(model.TransactionAuditActionEdited, now),                            // newest overall
			txLog(model.TransactionAuditActionCreated, now.Add(-5*time.Minute)),
			txLog(model.TransactionAuditActionDeleted, now.Add(-15*time.Minute)),      // oldest overall
		},
		countAccount: 3,
	}
	svc := NewAccountActivityService(NewSpaceAuditLogService(spaceRepo), NewTransactionAuditLogService(txRepo))

	rows, err := svc.List("acct-1", 10, 0)
	require.NoError(t, err)
	require.Len(t, rows, 5)

	// Strictly descending by timestamp.
	for i := 1; i < len(rows); i++ {
		assert.False(t, rows[i].Timestamp().After(rows[i-1].Timestamp()),
			"row %d (%v) is newer than row %d (%v)", i, rows[i].Timestamp(), i-1, rows[i-1].Timestamp())
	}
	// Top row is the transaction edit at `now`.
	require.NotNil(t, rows[0].TxLog)
	assert.Equal(t, model.TransactionAuditActionEdited, rows[0].TxLog.Action)
}

func TestAccountActivityService_List_Pagination(t *testing.T) {
	now := time.Now()
	spaceRepo := &stubSpaceAuditRepo{
		listAccount: []*model.SpaceAuditLogWithActor{
			spaceLog(model.SpaceAuditActionAccountCreated, now.Add(-30*time.Minute)),
		},
	}
	txRepo := &stubTxAuditRepo{
		listAccount: []*model.TransactionAuditLogWithActor{
			txLog(model.TransactionAuditActionEdited, now.Add(-10*time.Minute)),
			txLog(model.TransactionAuditActionEdited, now.Add(-20*time.Minute)),
			txLog(model.TransactionAuditActionEdited, now.Add(-40*time.Minute)),
		},
	}
	svc := NewAccountActivityService(NewSpaceAuditLogService(spaceRepo), NewTransactionAuditLogService(txRepo))

	page1, err := svc.List("a", 2, 0)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	page2, err := svc.List("a", 2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2)

	// Total of 4 entries; page2[1] is the oldest.
	assert.Equal(t, now.Add(-40*time.Minute).Unix(), page2[1].Timestamp().Unix())
}

func TestAccountActivityService_List_OffsetPastEndReturnsEmpty(t *testing.T) {
	svc := NewAccountActivityService(
		NewSpaceAuditLogService(&stubSpaceAuditRepo{}),
		NewTransactionAuditLogService(&stubTxAuditRepo{}),
	)
	rows, err := svc.List("a", 10, 100)
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestAccountActivityService_Count_SumsBothSources(t *testing.T) {
	svc := NewAccountActivityService(
		NewSpaceAuditLogService(&stubSpaceAuditRepo{countAccount: 3}),
		NewTransactionAuditLogService(&stubTxAuditRepo{countAccount: 7}),
	)
	count, err := svc.Count("a")
	require.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestAccountActivityService_List_PropagatesError(t *testing.T) {
	svc := NewAccountActivityService(
		NewSpaceAuditLogService(&stubSpaceAuditRepo{err: errors.New("boom")}),
		NewTransactionAuditLogService(&stubTxAuditRepo{}),
	)
	_, err := svc.List("a", 10, 0)
	assert.Error(t, err)
}

func TestAccountActivityService_ListSpace_MergesSpaceAndTxFeeds(t *testing.T) {
	now := time.Now()
	spaceRepo := &stubSpaceAuditRepo{
		listSpace: []*model.SpaceAuditLogWithActor{
			spaceLog(model.SpaceAuditActionRenamed, now.Add(-3*time.Minute)),
			spaceLog(model.SpaceAuditActionMemberInvited, now.Add(-5*time.Minute)),
		},
		countSpace: 2,
	}
	txRepo := &stubTxAuditRepo{
		listSpace: []*model.TransactionAuditLogWithActor{
			txLog(model.TransactionAuditActionCreated, now),
			txLog(model.TransactionAuditActionEdited, now.Add(-4*time.Minute)),
		},
		countSpace: 2,
	}
	svc := NewAccountActivityService(NewSpaceAuditLogService(spaceRepo), NewTransactionAuditLogService(txRepo))

	rows, err := svc.ListSpace("space", 10, 0)
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.NotNil(t, rows[0].TxLog, "newest is the tx-created row at `now`")
}

func TestAccountActivityService_CountSpace_SumsBothSources(t *testing.T) {
	svc := NewAccountActivityService(
		NewSpaceAuditLogService(&stubSpaceAuditRepo{countSpace: 4}),
		NewTransactionAuditLogService(&stubTxAuditRepo{countSpace: 6}),
	)
	count, err := svc.CountSpace("s")
	require.NoError(t, err)
	assert.Equal(t, 10, count)
}
