package service

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

const DefaultSpaceName = "My Space"

type SpaceService struct {
	spaceRepo repository.SpaceRepository
	auditSvc  *SpaceAuditLogService
}

func NewSpaceService(spaceRepo repository.SpaceRepository) *SpaceService {
	return &SpaceService{
		spaceRepo: spaceRepo,
	}
}

// SetAuditLogger wires the audit log service after construction. Kept separate from
// the constructor to avoid disturbing existing callers (especially tests) that don't
// care about auditing.
func (s *SpaceService) SetAuditLogger(audit *SpaceAuditLogService) {
	s.auditSvc = audit
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

// GetOwnedSpaces returns spaces owned by the user.
func (s *SpaceService) GetOwnedSpaces(userID string) ([]*model.Space, error) {
	spaces, err := s.spaceRepo.ByOwnerID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get owned spaces: %w", err)
	}
	return spaces, nil
}

// GetSharedSpaces returns spaces shared with the user (not owned by them).
func (s *SpaceService) GetSharedSpaces(userID string) ([]*model.Space, error) {
	spaces, err := s.spaceRepo.SharedWithUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared spaces: %w", err)
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
func (s *SpaceService) RemoveMember(spaceID, userID, actorID string) error {
	if err := s.spaceRepo.RemoveMember(spaceID, userID); err != nil {
		return err
	}
	s.auditSvc.Record(RecordOptions{
		SpaceID:      spaceID,
		ActorID:      actorID,
		Action:       model.SpaceAuditActionMemberRemoved,
		TargetUserID: userID,
	})
	return nil
}

// UpdateSpaceName updates the name of a space.
func (s *SpaceService) UpdateSpaceName(spaceID, name, actorID string) error {
	if name == "" {
		return fmt.Errorf("space name cannot be empty")
	}
	current, err := s.spaceRepo.ByID(spaceID)
	if err != nil {
		return err
	}
	oldName := current.Name
	if err := s.spaceRepo.UpdateName(spaceID, name); err != nil {
		return err
	}
	if oldName != name {
		s.auditSvc.Record(RecordOptions{
			SpaceID: spaceID,
			ActorID: actorID,
			Action:  model.SpaceAuditActionRenamed,
			Metadata: map[string]any{
				"old_name": oldName,
				"new_name": name,
			},
		})
	}
	return nil
}

// DeleteSpace permanently deletes a space and all its associated data.
func (s *SpaceService) DeleteSpace(spaceID, actorID string) error {
	current, err := s.spaceRepo.ByID(spaceID)
	if err != nil {
		return err
	}
	// Record before deleting so the audit row is written while the space still exists.
	// The audit table intentionally does not foreign-key space_id, so the entry survives.
	s.auditSvc.Record(RecordOptions{
		SpaceID: spaceID,
		ActorID: actorID,
		Action:  model.SpaceAuditActionDeleted,
		Metadata: map[string]any{
			"space_name": current.Name,
		},
	})
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
