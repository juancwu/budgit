package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

// ErrCategoryNameTaken is returned when a category with the same name already
// exists in the space.
var ErrCategoryNameTaken = errors.New("a category with this name already exists")

// ErrCategoryNotFound is returned when a category does not exist or does not
// belong to the requested space.
var ErrCategoryNotFound = errors.New("category not found")

const maxCategoryNameLen = 60

// CategoryService manages the per-space, user-created categories used to tag
// bills and budget plan lines.
type CategoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// ListBySpace returns the space's categories ordered by name.
func (s *CategoryService) ListBySpace(spaceID string) ([]*model.Category, error) {
	cats, err := s.repo.ListBySpace(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return cats, nil
}

// Get returns a single category, verifying it belongs to the space. Returns
// ErrCategoryNotFound otherwise.
func (s *CategoryService) Get(spaceID, categoryID string) (*model.Category, error) {
	cat, err := s.repo.ByID(categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to load category: %w", err)
	}
	if cat == nil || cat.SpaceID != spaceID {
		return nil, ErrCategoryNotFound
	}
	return cat, nil
}

// Create adds a category to the space. Names are trimmed and must be unique
// within the space (case-insensitive).
func (s *CategoryService) Create(spaceID, name, description string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > maxCategoryNameLen {
		return nil, fmt.Errorf("name must be at most %d characters", maxCategoryNameLen)
	}

	existing, err := s.repo.ListBySpace(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}
	for _, c := range existing {
		if strings.EqualFold(strings.TrimSpace(c.Name), name) {
			return nil, ErrCategoryNameTaken
		}
	}

	var desc *string
	if d := strings.TrimSpace(description); d != "" {
		desc = &d
	}

	now := time.Now()
	cat := &model.Category{
		ID:          uuid.NewString(),
		SpaceID:     spaceID,
		Name:        name,
		Description: desc,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(cat); err != nil {
		// The (space_id, name) unique index is the backstop against a race
		// between the check above and the insert.
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique") {
			return nil, ErrCategoryNameTaken
		}
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	return cat, nil
}

// Delete removes a category owned by the space. Bills lose the link (FK
// cascade) and budget plan lines become uncategorized (FK set null).
func (s *CategoryService) Delete(spaceID, categoryID string) error {
	if _, err := s.Get(spaceID, categoryID); err != nil {
		return err
	}
	if err := s.repo.Delete(categoryID); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}
