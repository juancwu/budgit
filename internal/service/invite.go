package service

import (
	"errors"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/utils"
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
	// Check if space exists
	space, err := s.spaceRepo.ByID(spaceID)
	if err != nil {
		return nil, err
	}

	// Check if inviter is member/owner of space? (Ideally yes, but for now assuming caller checks permissions)

	// Check if user is already a member
	// This would require a method on SpaceRepo or SpaceService.
	// For now, let's proceed.

	token := utils.RandomID() // Or a more secure token generator
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
