package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrTagNotFound        = errors.New("tag not found")
	ErrDuplicateTagName   = errors.New("tag with that name already exists in this space")
)

type TagRepository interface {
	Create(tag *model.Tag) error
	GetByID(id string) (*model.Tag, error)
	GetBySpaceID(spaceID string) ([]*model.Tag, error)
	Update(tag *model.Tag) error
	Delete(id string) error
}

type tagRepository struct {
	db *sqlx.DB
}

func NewTagRepository(db *sqlx.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) Create(tag *model.Tag) error {
	query := `INSERT INTO tags (id, space_id, name, color, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6);`
	_, err := r.db.Exec(query, tag.ID, tag.SpaceID, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") || strings.Contains(errStr, "duplicate key value") {
			return ErrDuplicateTagName
		}
		return err
	}
	return nil
}

func (r *tagRepository) GetByID(id string) (*model.Tag, error) {
	tag := &model.Tag{}
	query := `SELECT * FROM tags WHERE id = $1;`
	err := r.db.Get(tag, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrTagNotFound
	}
	return tag, err
}

func (r *tagRepository) GetBySpaceID(spaceID string) ([]*model.Tag, error) {
	var tags []*model.Tag
	query := `SELECT * FROM tags WHERE space_id = $1 ORDER BY name ASC;`
	err := r.db.Select(&tags, query, spaceID)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *tagRepository) Update(tag *model.Tag) error {
	tag.UpdatedAt = time.Now()
	query := `UPDATE tags SET name = $1, color = $2, updated_at = $3 WHERE id = $4;`
	result, err := r.db.Exec(query, tag.Name, tag.Color, tag.UpdatedAt, tag.ID)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") || strings.Contains(errStr, "duplicate key value") {
			return ErrDuplicateTagName
		}
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrTagNotFound
	}
	return err
}

func (r *tagRepository) Delete(id string) error {
	query := `DELETE FROM tags WHERE id = $1;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrTagNotFound
	}
	return err
}
