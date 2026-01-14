package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrListItemNotFound = errors.New("list item not found")
)

type ListItemRepository interface {
	Create(item *model.ListItem) error
	GetByID(id string) (*model.ListItem, error)
	GetByListID(listID string) ([]*model.ListItem, error)
	Update(item *model.ListItem) error
	Delete(id string) error
	DeleteByListID(listID string) error
}

type listItemRepository struct {
	db *sqlx.DB
}

func NewListItemRepository(db *sqlx.DB) ListItemRepository {
	return &listItemRepository{db: db}
}

func (r *listItemRepository) Create(item *model.ListItem) error {
	query := `INSERT INTO list_items (id, list_id, name, is_checked, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7);`
	_, err := r.db.Exec(query, item.ID, item.ListID, item.Name, item.IsChecked, item.CreatedBy, item.CreatedAt, item.UpdatedAt)
	return err
}

func (r *listItemRepository) GetByID(id string) (*model.ListItem, error) {
	item := &model.ListItem{}
	query := `SELECT * FROM list_items WHERE id = $1;`
	err := r.db.Get(item, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrListItemNotFound
	}
	return item, err
}

func (r *listItemRepository) GetByListID(listID string) ([]*model.ListItem, error) {
	var items []*model.ListItem
	query := `SELECT * FROM list_items WHERE list_id = $1 ORDER BY created_at ASC;`
	err := r.db.Select(&items, query, listID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *listItemRepository) Update(item *model.ListItem) error {
	item.UpdatedAt = time.Now()
	query := `UPDATE list_items SET name = $1, is_checked = $2, updated_at = $3 WHERE id = $4;`
	result, err := r.db.Exec(query, item.Name, item.IsChecked, item.UpdatedAt, item.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrListItemNotFound
	}
	return err
}

func (r *listItemRepository) Delete(id string) error {
	query := `DELETE FROM list_items WHERE id = $1;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrListItemNotFound
	}
	return err
}

func (r *listItemRepository) DeleteByListID(listID string) error {
	query := `DELETE FROM list_items WHERE list_id = $1;`
	_, err := r.db.Exec(query, listID)
	return err
}
