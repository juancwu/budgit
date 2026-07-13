package repository

import (
	"database/sql"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"github.com/jmoiron/sqlx"
)

type CategoryRepository interface {
	// ListBySpace returns the categories owned by a space, ordered by name.
	ListBySpace(spaceID string) ([]*model.Category, error)
	// ByID returns a single category, or (nil, nil) if it does not exist.
	ByID(id string) (*model.Category, error)
	// Create inserts a fully-populated category.
	Create(c *model.Category) error
	// Delete removes a category by ID. Its transaction links cascade; budget
	// plan lines referencing it must be cleared by the caller first.
	Delete(id string) error
}

type categoryRepository struct {
	db *sqlx.DB
}

func NewCategoryRepository(db *sqlx.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) ListBySpace(spaceID string) ([]*model.Category, error) {
	var categories []*model.Category
	query := `SELECT * FROM categories WHERE space_id = $1 ORDER BY name ASC;`
	if err := r.db.Select(&categories, query, spaceID); err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *categoryRepository) ByID(id string) (*model.Category, error) {
	c := &model.Category{}
	if err := r.db.Get(c, `SELECT * FROM categories WHERE id = $1;`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (r *categoryRepository) Create(c *model.Category) error {
	_, err := r.db.Exec(
		`INSERT INTO categories (id, space_id, name, description, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6);`,
		c.ID, c.SpaceID, c.Name, c.Description, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *categoryRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM categories WHERE id = $1;`, id)
	return err
}
