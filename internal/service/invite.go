package service

import (
	"errors"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrInviteAlreadyMember  = errors.New("user is already a member of this space")
	ErrInviteAlreadyPending = errors.New("an invitation is already pending for this email")
	ErrInviteSelf           = errors.New("you cannot invite yourself")
)

type InviteService struct {
	inviteRepo repository.InvitationRepository
	spaceRepo  repository.SpaceRepository
	userRepo   repository.UserRepository
	emailSvc   *EmailService
}

func NewInviteService(ir repository.InvitationRepository, sr repository.SpaceRepository, ur repository.UserRepository, es *EmailService) *InviteService {
	return &InviteService{
		inviteRepo: ir,
		spaceRepo:  sr,
		userRepo:   ur,
		emailSvc:   es,
	}
}

func (s *InviteService) CreateInvite(spaceID, inviterID, email string) (*model.SpaceInvitation, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	// Check if space exists
	space, err := s.spaceRepo.ByID(spaceID)
	if err != nil {
		return nil, err
	}

	// Block inviting an already-existing member
	if existingUser, err := s.userRepo.ByEmail(email); err == nil && existingUser != nil {
		if existingUser.ID == inviterID {
			return nil, ErrInviteSelf
		}
		isMember, err := s.spaceRepo.IsMember(spaceID, existingUser.ID)
		if err != nil {
			return nil, err
		}
		if isMember {
			return nil, ErrInviteAlreadyMember
		}
	}

	// Block duplicate pending invites for the same email
	existing, err := s.inviteRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for _, inv := range existing {
		if inv.Status == model.InvitationStatusPending && strings.EqualFold(inv.Email, email) && inv.ExpiresAt.After(now) {
			return nil, ErrInviteAlreadyPending
		}
	}

	token := uuid.NewString() // Or a more secure token generator
	expiresAt := time.Now().Add(48 * time.Hour)

	invitation := &model.SpaceInvitation{
		Token:     token,
		SpaceID:   spaceID,
		InviterID: inviterID,
		Email:     email,
		Status:    model.InvitationStatusPending,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.inviteRepo.Create(invitation); err != nil {
		return nil, err
	}

	// Get inviter name
	inviter, err := s.userRepo.ByID(inviterID)
	inviterName := "Someone"
	if err == nil {
		inviterName = inviter.Email // Or Name if available
		// Get profile for better name?
	}

	// Send Email
	go s.emailSvc.SendInvitationEmail(email, space.Name, inviterName, token)

	return invitation, nil
}

func (s *InviteService) AcceptInvite(token, userID string) (string, error) {
	invite, err := s.inviteRepo.GetByToken(token)
	if err != nil {
		return "", err
	}

	if invite.Status != model.InvitationStatusPending {
		return "", errors.New("invitation is not pending")
	}

	if time.Now().After(invite.ExpiresAt) {
		s.inviteRepo.UpdateStatus(token, model.InvitationStatusExpired)
		return "", errors.New("invitation expired")
	}

	// Add user to space
	err = s.spaceRepo.AddMember(invite.SpaceID, userID, model.RoleMember)
	if err != nil {
		return "", err
	}

	return invite.SpaceID, s.inviteRepo.UpdateStatus(token, model.InvitationStatusAccepted)
}

func (s *InviteService) CancelInvite(token string) error {
	invite, err := s.inviteRepo.GetByToken(token)
	if err != nil {
		return err
	}

	if invite.Status != model.InvitationStatusPending {
		return errors.New("invitation is not pending")
	}

	return s.inviteRepo.Delete(token)
}

type InviteContext struct {
	Invitation  *model.SpaceInvitation
	SpaceName   string
	InviterName string
}

func (s *InviteService) GetByToken(token string) (*InviteContext, error) {
	invite, err := s.inviteRepo.GetByToken(token)
	if err != nil {
		return nil, err
	}

	space, err := s.spaceRepo.ByID(invite.SpaceID)
	if err != nil {
		return nil, err
	}

	inviterName := "Someone"
	if inviter, err := s.userRepo.ByID(invite.InviterID); err == nil && inviter != nil {
		if inviter.Name != nil && *inviter.Name != "" {
			inviterName = *inviter.Name
		} else {
			inviterName = inviter.Email
		}
	}

	return &InviteContext{
		Invitation:  invite,
		SpaceName:   space.Name,
		InviterName: inviterName,
	}, nil
}

func (s *InviteService) GetPendingInvites(spaceID string) ([]*model.SpaceInvitation, error) {
	// Filter for pending only in memory or repo?
	// Repo returns all.
	all, err := s.inviteRepo.GetBySpaceID(spaceID)
	if err != nil {
		return nil, err
	}

	var pending []*model.SpaceInvitation
	for _, inv := range all {
		if inv.Status == model.InvitationStatusPending {
			pending = append(pending, inv)
		}
	}
	return pending, nil
}
