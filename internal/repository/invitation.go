package repository

import (
	"database/sql"
	"errors"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
)

type InvitationRepository interface {
	Create(invitation *model.SpaceInvitation) error
	GetByToken(token string) (*model.SpaceInvitation, error)
	GetBySpaceID(spaceID string) ([]*model.SpaceInvitation, error)
	UpdateStatus(token string, status model.InvitationStatus) error
	Delete(token string) error
}

type invitationRepository struct {
	db *sqlx.DB
}

func NewInvitationRepository(db *sqlx.DB) InvitationRepository {
	return &invitationRepository{db: db}
}

func (r *invitationRepository) Create(invitation *model.SpaceInvitation) error {
	query := `
		INSERT INTO space_invitations (token, space_id, inviter_id, email, status, expires_at, created_at, updated_at)
		VALUES (:token, :space_id, :inviter_id, :email, :status, :expires_at, :created_at, :updated_at)
	`
	_, err := r.db.NamedExec(query, invitation)
	return err
}

func (r *invitationRepository) GetByToken(token string) (*model.SpaceInvitation, error) {
	var invitation model.SpaceInvitation
	query := `SELECT * FROM space_invitations WHERE token = $1`
	err := r.db.Get(&invitation, query, token)
	if err == sql.ErrNoRows {
		return nil, ErrInvitationNotFound
	}
	return &invitation, err
}

func (r *invitationRepository) GetBySpaceID(spaceID string) ([]*model.SpaceInvitation, error) {
	var invitations []*model.SpaceInvitation
	query := `SELECT * FROM space_invitations WHERE space_id = $1 ORDER BY created_at DESC`
	err := r.db.Select(&invitations, query, spaceID)
	return invitations, err
}

func (r *invitationRepository) UpdateStatus(token string, status model.InvitationStatus) error {
	query := `UPDATE space_invitations SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE token = $2`
	_, err := r.db.Exec(query, status, token)
	return err
}

func (r *invitationRepository) Delete(token string) error {
	query := `DELETE FROM space_invitations WHERE token = $1`
	_, err := r.db.Exec(query, token)
	return err
}
