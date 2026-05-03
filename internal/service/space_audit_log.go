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

type SpaceAuditLogService struct {
	repo repository.SpaceAuditLogRepository
}

func NewSpaceAuditLogService(repo repository.SpaceAuditLogRepository) *SpaceAuditLogService {
	return &SpaceAuditLogService{repo: repo}
}

type RecordOptions struct {
	SpaceID      string
	ActorID      string
	Action       model.SpaceAuditAction
	TargetUserID string
	TargetEmail  string
	Metadata     map[string]any
}

// Record persists an audit entry. Failures are logged but never bubble up — auditing
// must not break the user-facing action that triggered it. A nil receiver is a no-op
// so tests can omit the dependency.
func (s *SpaceAuditLogService) Record(opts RecordOptions) {
	if s == nil {
		return
	}
	entry := &model.SpaceAuditLog{
		ID:        uuid.NewString(),
		SpaceID:   opts.SpaceID,
		Action:    opts.Action,
		CreatedAt: time.Now(),
	}
	if opts.ActorID != "" {
		actor := opts.ActorID
		entry.ActorID = &actor
	}
	if opts.TargetUserID != "" {
		t := opts.TargetUserID
		entry.TargetUserID = &t
	}
	if opts.TargetEmail != "" {
		e := opts.TargetEmail
		entry.TargetEmail = &e
	}
	if len(opts.Metadata) > 0 {
		raw, err := json.Marshal(opts.Metadata)
		if err != nil {
			slog.Error("failed to marshal audit metadata", "error", err, "action", opts.Action)
		} else {
			entry.Metadata = raw
		}
	}

	if err := s.repo.Create(entry); err != nil {
		slog.Error("failed to record space audit log",
			"error", err,
			"space_id", opts.SpaceID,
			"action", opts.Action,
		)
	}
}

func (s *SpaceAuditLogService) List(spaceID string, limit, offset int) ([]*model.SpaceAuditLogWithActor, error) {
	logs, err := s.repo.ListBySpace(spaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	return logs, nil
}

func (s *SpaceAuditLogService) Count(spaceID string) (int, error) {
	count, err := s.repo.CountBySpace(spaceID)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return count, nil
}
