package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

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

// GetMembers returns all members of a space with their profile info.
func (s *SpaceService) GetMembers(spaceID string) ([]*model.SpaceMemberWithProfile, error) {
	members, err := s.spaceRepo.GetMembers(spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	return members, nil
}

// RemoveMember removes a member from a space.
func (s *SpaceService) RemoveMember(spaceID, userID string) error {
	return s.spaceRepo.RemoveMember(spaceID, userID)
}

// UpdateSpaceName updates the name of a space.
func (s *SpaceService) UpdateSpaceName(spaceID, name string) error {
	if name == "" {
		return fmt.Errorf("space name cannot be empty")
	}
	return s.spaceRepo.UpdateName(spaceID, name)
}

// DeleteSpace permanently deletes a space and all its associated data.
func (s *SpaceService) DeleteSpace(spaceID string) error {
	return s.spaceRepo.Delete(spaceID)
}

func (s *SpaceService) GetMemberCount(spaceID string) (int, error) {
	count, err := s.spaceRepo.GetMemberCount(spaceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get member count: %w", err)
	}
	return count, nil
}

func (s *SpaceService) IsNameAvailable(name string, userID string) (bool, error) {
	spaces, err := s.GetSpacesForUser(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get spaces to check name availability: %w", err)
	}

	for _, sp := range spaces {
		if sp.Name == name {
			return false, nil
		}
	}

	return true, nil
}
