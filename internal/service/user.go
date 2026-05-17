package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrEmailConfirmationMismatch = errors.New("email confirmation does not match")
	ErrAccountAlreadyPending     = errors.New("account deletion already pending")
)

// maxAccountDeletionAttempts caps how many times a single deletion request
// gets retried before being parked in the failed state for human attention.
const maxAccountDeletionAttempts = 5

type UserService struct {
	db                  *sqlx.DB
	userRepository      repository.UserRepository
	deletionRequestRepo repository.AccountDeletionRequestRepository
	// triggerDeletion is set by the worker so that handlers can wake the
	// worker up immediately after enqueueing a new request, instead of
	// waiting for the next periodic tick.
	triggerDeletion chan<- struct{}
}

func NewUserService(
	db *sqlx.DB,
	userRepository repository.UserRepository,
	deletionRequestRepo repository.AccountDeletionRequestRepository,
) *UserService {
	return &UserService{
		db:                  db,
		userRepository:      userRepository,
		deletionRequestRepo: deletionRequestRepo,
	}
}

func (s *UserService) SetDeletionTrigger(ch chan<- struct{}) {
	s.triggerDeletion = ch
}

func (s *UserService) ByID(id string) (*model.User, error) {
	user, err := s.userRepository.ByID(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// RequestAccountDeletionInput captures the user's confirmed intent to
// permanently delete their account.
type RequestAccountDeletionInput struct {
	UserID            string
	ConfirmationEmail string
	Reason            string
	IPAddress         string
}

// RequestAccountDeletion validates the user's intent, flags the account as
// pending deletion (so middleware can lock out further activity), and
// enqueues a deletion job for the background worker to pick up. Both
// operations happen in a single transaction so we never end up with a
// flagged user without a queue entry, or vice versa.
func (s *UserService) RequestAccountDeletion(input RequestAccountDeletionInput) error {
	user, err := s.userRepository.ByID(input.UserID)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}

	if !strings.EqualFold(strings.TrimSpace(input.ConfirmationEmail), user.Email) {
		return ErrEmailConfirmationMismatch
	}

	if user.IsPendingDeletion() {
		return ErrAccountAlreadyPending
	}

	now := time.Now()
	req := &model.AccountDeletionRequest{
		ID:          uuid.NewString(),
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		Status:      model.AccountDeletionStatusPending,
		Attempts:    0,
		RequestedAt: now,
	}
	if reason := strings.TrimSpace(input.Reason); reason != "" {
		req.Reason = &reason
	}
	if ip := strings.TrimSpace(input.IPAddress); ip != "" {
		req.IPAddress = &ip
	}

	err = repository.WithTx(s.db, func(tx *sqlx.Tx) error {
		if err := s.userRepository.MarkPendingDeletionTx(tx, user.ID, now); err != nil {
			return fmt.Errorf("flag user pending deletion: %w", err)
		}
		if err := s.deletionRequestRepo.CreateTx(tx, req); err != nil {
			return fmt.Errorf("enqueue deletion request: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Wake the worker so it picks up immediately rather than waiting for the
	// next tick. Non-blocking so a busy/unbuffered channel never stalls the
	// HTTP request.
	if s.triggerDeletion != nil {
		select {
		case s.triggerDeletion <- struct{}{}:
		default:
		}
	}

	slog.Info("account deletion requested", "user_id", user.ID, "request_id", req.ID)
	return nil
}

// ProcessPendingDeletions drains all currently pending requests, processing
// each one in its own transaction. Safe to invoke from a ticker, on startup,
// and on demand right after enqueueing. Returns the number of requests
// completed in this call.
func (s *UserService) ProcessPendingDeletions() int {
	processed := 0
	for {
		req, err := s.deletionRequestRepo.ClaimNextPending()
		if errors.Is(err, repository.ErrAccountDeletionRequestNotFound) {
			return processed
		}
		if err != nil {
			slog.Error("failed to claim deletion request", "error", err)
			return processed
		}

		if err := s.executeDeletion(req); err != nil {
			s.handleDeletionFailure(req, err)
			continue
		}
		slog.Info("account deletion completed", "user_id", req.UserID, "request_id", req.ID, "attempt", req.Attempts)
		processed++
	}
}

func (s *UserService) executeDeletion(req *model.AccountDeletionRequest) error {
	return repository.WithTx(s.db, func(tx *sqlx.Tx) error {
		// space_audit_logs and transaction_audit_logs have no FK to their
		// parent rows, so we drop them explicitly before the spaces are gone
		// or they'd become orphans.
		if _, err := tx.Exec(
			`DELETE FROM transaction_audit_logs
			 WHERE transaction_id IN (
				 SELECT t.id FROM transactions t
				 JOIN accounts a ON a.id = t.account_id
				 JOIN spaces s ON s.id = a.space_id
				 WHERE s.owner_id = $1
			 );`,
			req.UserID,
		); err != nil {
			return fmt.Errorf("delete transaction audit logs: %w", err)
		}

		if _, err := tx.Exec(
			`DELETE FROM space_audit_logs
			 WHERE space_id IN (SELECT id FROM spaces WHERE owner_id = $1);`,
			req.UserID,
		); err != nil {
			return fmt.Errorf("delete space audit logs: %w", err)
		}

		// Cascades accounts, transactions, allocations, recurring events,
		// tags, members, and pending invitations on each space.
		result, err := tx.Exec(`DELETE FROM spaces WHERE owner_id = $1;`, req.UserID)
		if err != nil {
			return fmt.Errorf("delete owned spaces: %w", err)
		}
		spacesDeleted, _ := result.RowsAffected()

		// Remove the user. Cascades tokens, space memberships in spaces
		// owned by others, and invitations the user sent.
		if _, err := tx.Exec(`DELETE FROM users WHERE id = $1;`, req.UserID); err != nil {
			return fmt.Errorf("delete user: %w", err)
		}

		if err := s.deletionRequestRepo.MarkCompletedTx(tx, req.ID, int(spacesDeleted)); err != nil {
			return fmt.Errorf("mark request completed: %w", err)
		}
		return nil
	})
}

func (s *UserService) handleDeletionFailure(req *model.AccountDeletionRequest, deletionErr error) {
	msg := deletionErr.Error()
	if req.Attempts >= maxAccountDeletionAttempts {
		slog.Error("account deletion permanently failed", "request_id", req.ID, "user_id", req.UserID, "attempts", req.Attempts, "error", msg)
		if err := s.deletionRequestRepo.MarkFailedTerminal(req.ID, msg); err != nil {
			slog.Error("failed to mark request terminal", "error", err, "request_id", req.ID)
		}
		return
	}
	slog.Warn("account deletion attempt failed, will retry", "request_id", req.ID, "user_id", req.UserID, "attempt", req.Attempts, "error", msg)
	if err := s.deletionRequestRepo.MarkFailedRetryable(req.ID, msg); err != nil {
		slog.Error("failed to mark request retryable", "error", err, "request_id", req.ID)
	}
}
