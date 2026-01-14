package repository

import (
	"database/sql"
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrShoppingListNotFound = errors.New("shopping list not found")
)

type ShoppingListRepository interface {
	Create(list *model.ShoppingList) error
	GetByID(id string) (*model.ShoppingList, error)
	GetBySpaceID(spaceID string) ([]*model.ShoppingList, error)
	Update(list *model.ShoppingList) error
	Delete(id string) error
}

type shoppingListRepository struct {
	db *sqlx.DB
}

func NewShoppingListRepository(db *sqlx.DB) ShoppingListRepository {
	return &shoppingListRepository{db: db}
}

func (r *shoppingListRepository) Create(list *model.ShoppingList) error {
	query := `INSERT INTO shopping_lists (id, space_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5);`
	_, err := r.db.Exec(query, list.ID, list.SpaceID, list.Name, list.CreatedAt, list.UpdatedAt)
	return err
}

func (r *shoppingListRepository) GetByID(id string) (*model.ShoppingList, error) {
	list := &model.ShoppingList{}
	query := `SELECT * FROM shopping_lists WHERE id = $1;`
	err := r.db.Get(list, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrShoppingListNotFound
	}
	return list, err
}

func (r *shoppingListRepository) GetBySpaceID(spaceID string) ([]*model.ShoppingList, error) {
	var lists []*model.ShoppingList
	query := `SELECT * FROM shopping_lists WHERE space_id = $1 ORDER BY created_at DESC;`
	err := r.db.Select(&lists, query, spaceID)
	if err != nil {
		return nil, err
	}
	return lists, nil
}

func (r *shoppingListRepository) Update(list *model.ShoppingList) error {
	list.UpdatedAt = time.Now()
	query := `UPDATE shopping_lists SET name = $1, updated_at = $2 WHERE id = $3;`
	result, err := r.db.Exec(query, list.Name, list.UpdatedAt, list.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrShoppingListNotFound
	}
	return err
}

func (r *shoppingListRepository) Delete(id string) error {
	query := `DELETE FROM shopping_lists WHERE id = $1;`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrShoppingListNotFound
	}
	return err
}
