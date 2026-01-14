package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

const PersonalSpaceName = "Personal Space"

type SpaceService struct {
	spaceRepo repository.SpaceRepository
}

func NewSpaceService(spaceRepo repository.SpaceRepository) *SpaceService {
	return &SpaceService{
		spaceRepo: spaceRepo,
	}
}

// CreateSpace creates a new space and sets the owner.
func (s *SpaceService) CreateSpace(name string, ownerID string) (*model.Space, error) {
	if name == "" {
		return nil, fmt.Errorf("space name cannot be empty")
	}

	space := &model.Space{
		ID:        uuid.NewString(),
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.spaceRepo.Create(space)
	if err != nil {
		return nil, fmt.Errorf("failed to create space: %w", err)
	}

	return space, nil
}

// EnsurePersonalSpace creates a "Personal Space" for a user if one doesn't exist.
func (s *SpaceService) EnsurePersonalSpace(user *model.User) (*model.Space, error) {
	spaces, err := s.spaceRepo.ByUserID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user spaces: %w", err)
	}

	// Check if a personal space already exists.
	// We identify it by the user being the owner and the name being the default.
	for _, space := range spaces {
		if space.OwnerID == user.ID && space.Name == PersonalSpaceName {
			return space, nil // Personal space already exists
		}
	}

	// If no personal space, create one.
	return s.CreateSpace(PersonalSpaceName, user.ID)
}

// GetSpacesForUser returns all spaces a user is a member of.
func (s *SpaceService) GetSpacesForUser(userID string) ([]*model.Space, error) {
	spaces, err := s.spaceRepo.ByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get spaces for user: %w", err)
	}
	return spaces, nil
}

// GetSpace retrieves a single space by its ID.
func (s *SpaceService) GetSpace(spaceID string) (*model.Space, error) {
	space, err := s.spaceRepo.ByID(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get space: %w", err)
	}
	return space, nil
}

// IsMember checks if a user is a member of a given space.
func (s *SpaceService) IsMember(userID, spaceID string) (bool, error) {
	isMember, err := s.spaceRepo.IsMember(spaceID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return isMember, nil
}
