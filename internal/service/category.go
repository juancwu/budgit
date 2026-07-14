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
// exists in the account.
var ErrCategoryNameTaken = errors.New("a category with this name already exists")

// ErrCategoryNotFound is returned when a category does not exist or does not
// belong to the requested account.
var ErrCategoryNotFound = errors.New("category not found")

const maxCategoryNameLen = 60

// CategoryService manages the per-account, user-created categories used to tag
// bills and deposits.
type CategoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// ListByAccount returns the account's categories ordered by name.
func (s *CategoryService) ListByAccount(accountID string) ([]*model.Category, error) {
	cats, err := s.repo.ListByAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return cats, nil
}

// Get returns a single category, verifying it belongs to the account. Returns
// ErrCategoryNotFound otherwise.
func (s *CategoryService) Get(accountID, categoryID string) (*model.Category, error) {
	cat, err := s.repo.ByID(categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to load category: %w", err)
	}
	if cat == nil || cat.AccountID != accountID {
		return nil, ErrCategoryNotFound
	}
	return cat, nil
}

// Create adds a category to the account. Names are trimmed and must be unique
// within the account (case-insensitive).
func (s *CategoryService) Create(accountID, name, description string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > maxCategoryNameLen {
		return nil, fmt.Errorf("name must be at most %d characters", maxCategoryNameLen)
	}

	existing, err := s.repo.ListByAccount(accountID)
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
		AccountID:   accountID,
		Name:        name,
		Description: desc,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(cat); err != nil {
		// The (account_id, name) unique index is the backstop against a race
		// between the check above and the insert.
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique") {
			return nil, ErrCategoryNameTaken
		}
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	return cat, nil
}

// Delete removes a category owned by the account. Its transaction links cascade.
func (s *CategoryService) Delete(accountID, categoryID string) error {
	if _, err := s.Get(accountID, categoryID); err != nil {
		return err
	}
	if err := s.repo.Delete(categoryID); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}
