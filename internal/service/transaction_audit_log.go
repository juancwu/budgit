package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type TransactionAuditLogService struct {
	repo repository.TransactionAuditLogRepository
}

func NewTransactionAuditLogService(repo repository.TransactionAuditLogRepository) *TransactionAuditLogService {
	return &TransactionAuditLogService{repo: repo}
}

type TransactionRecordOptions struct {
	TransactionID string
	ActorID       string
	Action        model.TransactionAuditAction
	Metadata      map[string]any
}

// Record persists a transaction audit entry. Failures are logged but never bubble up —
// auditing must not break the user-facing action that triggered it. A nil receiver is
// a no-op so tests can omit the dependency.
func (s *TransactionAuditLogService) Record(opts TransactionRecordOptions) {
	if s == nil {
		return
	}
	entry := &model.TransactionAuditLog{
		ID:            uuid.NewString(),
		TransactionID: opts.TransactionID,
		Action:        opts.Action,
		CreatedAt:     time.Now(),
	}
	if opts.ActorID != "" {
		actor := opts.ActorID
		entry.ActorID = &actor
	}
	if len(opts.Metadata) > 0 {
		raw, err := json.Marshal(opts.Metadata)
		if err != nil {
			slog.Error("failed to marshal transaction audit metadata", "error", err, "action", opts.Action)
		} else {
			entry.Metadata = raw
		}
	}

	if err := s.repo.Create(entry); err != nil {
		slog.Error("failed to record transaction audit log",
			"error", err,
			"transaction_id", opts.TransactionID,
			"action", opts.Action,
		)
	}
}

func (s *TransactionAuditLogService) List(transactionID string, limit, offset int) ([]*model.TransactionAuditLogWithActor, error) {
	logs, err := s.repo.ListByTransaction(transactionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transaction audit logs: %w", err)
	}
	return logs, nil
}

func (s *TransactionAuditLogService) Count(transactionID string) (int, error) {
	count, err := s.repo.CountByTransaction(transactionID)
	if err != nil {
		return 0, fmt.Errorf("failed to count transaction audit logs: %w", err)
	}
	return count, nil
}
