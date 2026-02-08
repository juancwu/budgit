package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrSpaceNotFound = errors.New("space not found")
)

type SpaceRepository interface {
	Create(space *model.Space) error
	ByID(id string) (*model.Space, error)
	ByUserID(userID string) ([]*model.Space, error)
	AddMember(spaceID, userID string, role model.Role) error
	RemoveMember(spaceID, userID string) error
	IsMember(spaceID, userID string) (bool, error)
	GetMembers(spaceID string) ([]*model.SpaceMemberWithProfile, error)
	UpdateName(spaceID, name string) error
}

type spaceRepository struct {
	db *sqlx.DB
}

func NewSpaceRepository(db *sqlx.DB) SpaceRepository {
	return &spaceRepository{db: db}
}

func (r *spaceRepository) Create(space *model.Space) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert Space
	querySpace := `INSERT INTO spaces (id, name, owner_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5);`
	_, err = tx.Exec(querySpace, space.ID, space.Name, space.OwnerID, space.CreatedAt, space.UpdatedAt)
	if err != nil {
		return err
	}

	// Insert Owner as Member
	queryMember := `INSERT INTO space_members (space_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4);`
	_, err = tx.Exec(queryMember, space.ID, space.OwnerID, model.RoleOwner, space.CreatedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *spaceRepository) ByID(id string) (*model.Space, error) {
	space := &model.Space{}
	query := `SELECT * FROM spaces WHERE id = $1;`

	err := r.db.Get(space, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrSpaceNotFound
	}

	return space, err
}

func (r *spaceRepository) ByUserID(userID string) ([]*model.Space, error) {
	var spaces []*model.Space
	// Select spaces where user is a member
	query := `
		SELECT s.* 
		FROM spaces s
		JOIN space_members sm ON s.id = sm.space_id
		WHERE sm.user_id = $1
		ORDER BY s.created_at DESC;
	`

	err := r.db.Select(&spaces, query, userID)
	if err != nil {
		return nil, err
	}

	return spaces, nil
}

func (r *spaceRepository) AddMember(spaceID, userID string, role model.Role) error {
	query := `INSERT INTO space_members (space_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4);`
	_, err := r.db.Exec(query, spaceID, userID, role, time.Now())
	return err
}

func (r *spaceRepository) RemoveMember(spaceID, userID string) error {
	query := `DELETE FROM space_members WHERE space_id = $1 AND user_id = $2;`
	_, err := r.db.Exec(query, spaceID, userID)
	return err
}

func (r *spaceRepository) IsMember(spaceID, userID string) (bool, error) {
	var count int
	query := `SELECT count(*) FROM space_members WHERE space_id = $1 AND user_id = $2;`
	err := r.db.Get(&count, query, spaceID, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *spaceRepository) GetMembers(spaceID string) ([]*model.SpaceMemberWithProfile, error) {
	var members []*model.SpaceMemberWithProfile
	query := `
		SELECT sm.space_id, sm.user_id, sm.role, sm.joined_at,
		       p.name, u.email
		FROM space_members sm
		JOIN users u ON sm.user_id = u.id
		JOIN profiles p ON sm.user_id = p.user_id
		WHERE sm.space_id = $1
		ORDER BY sm.role DESC, sm.joined_at ASC;`
	err := r.db.Select(&members, query, spaceID)
	return members, err
}

func (r *spaceRepository) UpdateName(spaceID, name string) error {
	query := `UPDATE spaces SET name = $1, updated_at = $2 WHERE id = $3;`
	_, err := r.db.Exec(query, name, time.Now(), spaceID)
	return err
}
