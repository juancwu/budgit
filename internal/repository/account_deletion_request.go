package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrAccountDeletionRequestNotFound = errors.New("account deletion request not found")

type AccountDeletionRequestRepository interface {
	CreateTx(tx *sqlx.Tx, req *model.AccountDeletionRequest) error
	ByID(id string) (*model.AccountDeletionRequest, error)
	HasPendingForUser(userID string) (bool, error)
	LatestForUser(userID string) (*model.AccountDeletionRequest, error)

	// ClaimNextPending atomically transitions the oldest pending request to
	// "processing" and returns it. Returns ErrAccountDeletionRequestNotFound
	// when no pending row exists. Uses SKIP LOCKED so multiple workers can
	// safely run in parallel without colliding.
	ClaimNextPending() (*model.AccountDeletionRequest, error)

	// MarkCompletedTx marks the request as completed within the same tx that
	// deletes the user's data, so the queue row and the data wipe always
	// agree.
	MarkCompletedTx(tx *sqlx.Tx, id string, spacesDeleted int) error

	// MarkFailedRetryable records the error and returns the request to the
	// pending state so the next worker tick will retry it.
	MarkFailedRetryable(id string, errMsg string) error

	// MarkFailedTerminal records the error and parks the request in the
	// failed state for human investigation. Called once attempts exceed the
	// retry budget.
	MarkFailedTerminal(id string, errMsg string) error
}

type accountDeletionRequestRepository struct {
	db *sqlx.DB
}

func NewAccountDeletionRequestRepository(db *sqlx.DB) AccountDeletionRequestRepository {
	return &accountDeletionRequestRepository{db: db}
}

func (r *accountDeletionRequestRepository) CreateTx(tx *sqlx.Tx, req *model.AccountDeletionRequest) error {
	_, err := tx.Exec(
		`INSERT INTO account_deletion_requests
			(id, user_id, email, name, reason, ip_address, status, attempts,
			 requested_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9);`,
		req.ID, req.UserID, req.Email, req.Name, req.Reason, req.IPAddress,
		req.Status, req.Attempts, req.RequestedAt,
	)
	return err
}

func (r *accountDeletionRequestRepository) ByID(id string) (*model.AccountDeletionRequest, error) {
	var req model.AccountDeletionRequest
	err := r.db.Get(&req, `SELECT * FROM account_deletion_requests WHERE id = $1;`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountDeletionRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *accountDeletionRequestRepository) LatestForUser(userID string) (*model.AccountDeletionRequest, error) {
	var req model.AccountDeletionRequest
	err := r.db.Get(&req,
		`SELECT * FROM account_deletion_requests
		 WHERE user_id = $1
		 ORDER BY requested_at DESC
		 LIMIT 1;`,
		userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountDeletionRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *accountDeletionRequestRepository) HasPendingForUser(userID string) (bool, error) {
	var n int
	err := r.db.Get(&n,
		`SELECT COUNT(*) FROM account_deletion_requests
		 WHERE user_id = $1 AND status IN ('pending', 'processing');`,
		userID,
	)
	return n > 0, err
}

func (r *accountDeletionRequestRepository) ClaimNextPending() (*model.AccountDeletionRequest, error) {
	var req model.AccountDeletionRequest
	err := r.db.Get(&req,
		`UPDATE account_deletion_requests
		 SET status = 'processing',
		     attempts = attempts + 1,
		     updated_at = $1
		 WHERE id = (
		     SELECT id FROM account_deletion_requests
		     WHERE status = 'pending'
		     ORDER BY requested_at
		     FOR UPDATE SKIP LOCKED
		     LIMIT 1
		 )
		 RETURNING *;`,
		time.Now(),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountDeletionRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *accountDeletionRequestRepository) MarkCompletedTx(tx *sqlx.Tx, id string, spacesDeleted int) error {
	now := time.Now()
	_, err := tx.Exec(
		`UPDATE account_deletion_requests
		 SET status = 'completed',
		     spaces_deleted = $2,
		     completed_at = $3,
		     updated_at = $3,
		     last_error = NULL
		 WHERE id = $1;`,
		id, spacesDeleted, now,
	)
	return err
}

func (r *accountDeletionRequestRepository) MarkFailedRetryable(id string, errMsg string) error {
	_, err := r.db.Exec(
		`UPDATE account_deletion_requests
		 SET status = 'pending',
		     last_error = $2,
		     updated_at = $3
		 WHERE id = $1;`,
		id, errMsg, time.Now(),
	)
	return err
}

func (r *accountDeletionRequestRepository) MarkFailedTerminal(id string, errMsg string) error {
	_, err := r.db.Exec(
		`UPDATE account_deletion_requests
		 SET status = 'failed',
		     last_error = $2,
		     updated_at = $3
		 WHERE id = $1;`,
		id, errMsg, time.Now(),
	)
	return err
}
