package service

import (
	"fmt"
	"sort"

	"git.juancwu.dev/juancwu/budgit/internal/model"
)

type AccountActivityService struct {
	spaceAudit *SpaceAuditLogService
	txAudit    *TransactionAuditLogService
}

func NewAccountActivityService(spaceAudit *SpaceAuditLogService, txAudit *TransactionAuditLogService) *AccountActivityService {
	return &AccountActivityService{
		spaceAudit: spaceAudit,
		txAudit:    txAudit,
	}
}

// List returns a merged feed of account-scoped events (account.created/renamed/deleted)
// and transaction events (created/edited/deleted) for transactions in this account.
//
// Rather than a SQL UNION across two heterogeneous tables, we fetch up to (offset+limit)
// from each side and merge in Go. Audit volume per account is low, so the simplicity
// outweighs the slight overfetch.
func (s *AccountActivityService) List(accountID string, limit, offset int) ([]model.AccountActivityRow, error) {
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	fetchN := offset + limit

	spaceLogs, err := s.spaceAudit.repo.ListAccountEvents(accountID, fetchN, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list account events: %w", err)
	}
	txLogs, err := s.txAudit.repo.ListByAccount(accountID, fetchN, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list transaction events: %w", err)
	}

	rows := make([]model.AccountActivityRow, 0, len(spaceLogs)+len(txLogs))
	for _, l := range spaceLogs {
		rows = append(rows, model.AccountActivityRow{SpaceLog: l})
	}
	for _, l := range txLogs {
		rows = append(rows, model.AccountActivityRow{TxLog: l})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Timestamp().After(rows[j].Timestamp())
	})

	if offset >= len(rows) {
		return nil, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], nil
}

func (s *AccountActivityService) Count(accountID string) (int, error) {
	spaceCount, err := s.spaceAudit.repo.CountAccountEvents(accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to count account events: %w", err)
	}
	txCount, err := s.txAudit.repo.CountByAccount(accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to count transaction events: %w", err)
	}
	return spaceCount + txCount, nil
}
