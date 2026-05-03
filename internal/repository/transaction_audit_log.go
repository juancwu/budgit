package repository

import (
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

type TransactionAuditLogRepository interface {
	Create(log *model.TransactionAuditLog) error
	ListByTransaction(transactionID string, limit, offset int) ([]*model.TransactionAuditLogWithActor, error)
	CountByTransaction(transactionID string) (int, error)
	ListByAccount(accountID string, limit, offset int) ([]*model.TransactionAuditLogWithActor, error)
	CountByAccount(accountID string) (int, error)
	ListBySpace(spaceID string, limit, offset int) ([]*model.TransactionAuditLogWithActor, error)
	CountBySpace(spaceID string) (int, error)
}

type transactionAuditLogRepository struct {
	db *sqlx.DB
}

func NewTransactionAuditLogRepository(db *sqlx.DB) TransactionAuditLogRepository {
	return &transactionAuditLogRepository{db: db}
}

func (r *transactionAuditLogRepository) Create(log *model.TransactionAuditLog) error {
	query := `
		INSERT INTO transaction_audit_logs
			(id, transaction_id, actor_id, action, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6);`
	metadata := log.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	_, err := r.db.Exec(query,
		log.ID, log.TransactionID, log.ActorID, log.Action, metadata, log.CreatedAt,
	)
	return err
}

func (r *transactionAuditLogRepository) ListByTransaction(transactionID string, limit, offset int) ([]*model.TransactionAuditLogWithActor, error) {
	query := `
		SELECT
			a.id, a.transaction_id, a.actor_id, a.action, a.metadata, a.created_at,
			actor.name AS actor_name, actor.email AS actor_email
		FROM transaction_audit_logs a
		LEFT JOIN users actor ON actor.id = a.actor_id
		WHERE a.transaction_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3;`
	var logs []*model.TransactionAuditLogWithActor
	err := r.db.Select(&logs, query, transactionID, limit, offset)
	return logs, err
}

func (r *transactionAuditLogRepository) CountByTransaction(transactionID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM transaction_audit_logs WHERE transaction_id = $1;`, transactionID)
	return count, err
}

// ListByAccount returns transaction audit entries whose transaction belongs to the given
// account. Uses the live transactions table when present and falls back to the metadata
// account_id (set on creation) so entries for deleted transactions are still surfaced.
func (r *transactionAuditLogRepository) ListByAccount(accountID string, limit, offset int) ([]*model.TransactionAuditLogWithActor, error) {
	query := `
		SELECT
			a.id, a.transaction_id, a.actor_id, a.action, a.metadata, a.created_at,
			actor.name AS actor_name, actor.email AS actor_email
		FROM transaction_audit_logs a
		LEFT JOIN users actor ON actor.id = a.actor_id
		LEFT JOIN transactions t ON t.id = a.transaction_id
		WHERE t.account_id = $1
		   OR (t.id IS NULL AND a.metadata->>'account_id' = $1)
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3;`
	var logs []*model.TransactionAuditLogWithActor
	err := r.db.Select(&logs, query, accountID, limit, offset)
	return logs, err
}

func (r *transactionAuditLogRepository) CountByAccount(accountID string) (int, error) {
	var count int
	err := r.db.Get(&count,
		`SELECT COUNT(*)
		 FROM transaction_audit_logs a
		 LEFT JOIN transactions t ON t.id = a.transaction_id
		 WHERE t.account_id = $1
		    OR (t.id IS NULL AND a.metadata->>'account_id' = $1);`,
		accountID)
	return count, err
}

// ListBySpace returns transaction audit entries for any transaction belonging to an
// account in this space. Resolves the account via the live transactions row, falling
// back to the metadata account_id (set on creation) so entries for deleted transactions
// still surface as long as the account itself exists.
func (r *transactionAuditLogRepository) ListBySpace(spaceID string, limit, offset int) ([]*model.TransactionAuditLogWithActor, error) {
	query := `
		SELECT
			a.id, a.transaction_id, a.actor_id, a.action, a.metadata, a.created_at,
			actor.name AS actor_name, actor.email AS actor_email
		FROM transaction_audit_logs a
		LEFT JOIN users actor ON actor.id = a.actor_id
		LEFT JOIN transactions t ON t.id = a.transaction_id
		LEFT JOIN accounts acc
		       ON acc.id = COALESCE(t.account_id, a.metadata->>'account_id')
		WHERE acc.space_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3;`
	var logs []*model.TransactionAuditLogWithActor
	err := r.db.Select(&logs, query, spaceID, limit, offset)
	return logs, err
}

func (r *transactionAuditLogRepository) CountBySpace(spaceID string) (int, error) {
	var count int
	err := r.db.Get(&count,
		`SELECT COUNT(*)
		 FROM transaction_audit_logs a
		 LEFT JOIN transactions t ON t.id = a.transaction_id
		 LEFT JOIN accounts acc
		        ON acc.id = COALESCE(t.account_id, a.metadata->>'account_id')
		 WHERE acc.space_id = $1;`,
		spaceID)
	return count, err
}
