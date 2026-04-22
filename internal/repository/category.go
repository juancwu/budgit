package repository

import (
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

type CategoryRepository interface {
	All() ([]*model.Category, error)
}

type categoryRepository struct {
	db *sqlx.DB
}

func NewCategoryRepository(db *sqlx.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) All() ([]*model.Category, error) {
	var categories []*model.Category
	query := `SELECT * FROM categories ORDER BY name ASC;`
	if err := r.db.Select(&categories, query); err != nil {
		return nil, err
	}
	return categories, nil
}
