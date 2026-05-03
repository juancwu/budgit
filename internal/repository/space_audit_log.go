package repository

import (
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

type SpaceAuditLogRepository interface {
	Create(log *model.SpaceAuditLog) error
	ListBySpace(spaceID string, limit, offset int) ([]*model.SpaceAuditLogWithActor, error)
	CountBySpace(spaceID string) (int, error)
	ListAccountEvents(accountID string, limit, offset int) ([]*model.SpaceAuditLogWithActor, error)
	CountAccountEvents(accountID string) (int, error)
}

type spaceAuditLogRepository struct {
	db *sqlx.DB
}

func NewSpaceAuditLogRepository(db *sqlx.DB) SpaceAuditLogRepository {
	return &spaceAuditLogRepository{db: db}
}

func (r *spaceAuditLogRepository) Create(log *model.SpaceAuditLog) error {
	query := `
		INSERT INTO space_audit_logs
			(id, space_id, actor_id, action, target_user_id, target_email, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`
	metadata := log.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	_, err := r.db.Exec(query,
		log.ID, log.SpaceID, log.ActorID, log.Action,
		log.TargetUserID, log.TargetEmail, metadata, log.CreatedAt,
	)
	return err
}

func (r *spaceAuditLogRepository) ListBySpace(spaceID string, limit, offset int) ([]*model.SpaceAuditLogWithActor, error) {
	query := `
		SELECT
			a.id, a.space_id, a.actor_id, a.action, a.target_user_id, a.target_email,
			a.metadata, a.created_at,
			actor.name AS actor_name, actor.email AS actor_email,
			target.name AS target_user_name, target.email AS target_user_email
		FROM space_audit_logs a
		LEFT JOIN users actor ON actor.id = a.actor_id
		LEFT JOIN users target ON target.id = a.target_user_id
		WHERE a.space_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3;`
	var logs []*model.SpaceAuditLogWithActor
	err := r.db.Select(&logs, query, spaceID, limit, offset)
	return logs, err
}

func (r *spaceAuditLogRepository) CountBySpace(spaceID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM space_audit_logs WHERE space_id = $1;`, spaceID)
	return count, err
}

func (r *spaceAuditLogRepository) ListAccountEvents(accountID string, limit, offset int) ([]*model.SpaceAuditLogWithActor, error) {
	query := `
		SELECT
			a.id, a.space_id, a.actor_id, a.action, a.target_user_id, a.target_email,
			a.metadata, a.created_at,
			actor.name AS actor_name, actor.email AS actor_email,
			target.name AS target_user_name, target.email AS target_user_email
		FROM space_audit_logs a
		LEFT JOIN users actor ON actor.id = a.actor_id
		LEFT JOIN users target ON target.id = a.target_user_id
		WHERE a.action LIKE 'account.%'
		  AND a.metadata->>'account_id' = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3;`
	var logs []*model.SpaceAuditLogWithActor
	err := r.db.Select(&logs, query, accountID, limit, offset)
	return logs, err
}

func (r *spaceAuditLogRepository) CountAccountEvents(accountID string) (int, error) {
	var count int
	err := r.db.Get(&count,
		`SELECT COUNT(*) FROM space_audit_logs
		 WHERE action LIKE 'account.%' AND metadata->>'account_id' = $1;`,
		accountID)
	return count, err
}
