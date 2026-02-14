package service

import (
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestInviteService(dbi testutil.DBInfo) *InviteService {
	inviteRepo := repository.NewInvitationRepository(dbi.DB)
	spaceRepo := repository.NewSpaceRepository(dbi.DB)
	userRepo := repository.NewUserRepository(dbi.DB)
	emailSvc := NewEmailService(nil, "test@example.com", "http://localhost:9999", "Budgit Test", false)
	return NewInviteService(inviteRepo, spaceRepo, userRepo, emailSvc)
}

func TestInviteService_CreateInvite(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestInviteService(dbi)

		owner := testutil.CreateTestUser(t, dbi.DB, "invite-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Invite Space")

		invitation, err := svc.CreateInvite(space.ID, owner.ID, "invitee@example.com")
		require.NoError(t, err)
		assert.Equal(t, space.ID, invitation.SpaceID)
		assert.Equal(t, owner.ID, invitation.InviterID)
		assert.Equal(t, "invitee@example.com", invitation.Email)
		assert.NotEmpty(t, invitation.Token)
	})
}

func TestInviteService_AcceptInvite(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestInviteService(dbi)

		owner := testutil.CreateTestUser(t, dbi.DB, "accept-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Accept Space")
		invitation := testutil.CreateTestInvitation(t, dbi.DB, space.ID, owner.ID, "acceptee@example.com")

		// Create the user who will accept
		accepter := testutil.CreateTestUser(t, dbi.DB, "acceptee@example.com", nil)

		spaceID, err := svc.AcceptInvite(invitation.Token, accepter.ID)
		require.NoError(t, err)
		assert.Equal(t, space.ID, spaceID)

		// Verify member was added to space
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		isMember, err := spaceRepo.IsMember(space.ID, accepter.ID)
		require.NoError(t, err)
		assert.True(t, isMember)
	})
}

func TestInviteService_CancelInvite(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestInviteService(dbi)

		owner := testutil.CreateTestUser(t, dbi.DB, "cancel-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Cancel Space")
		invitation := testutil.CreateTestInvitation(t, dbi.DB, space.ID, owner.ID, "cancelee@example.com")

		err := svc.CancelInvite(invitation.Token)
		require.NoError(t, err)

		// Verify invitation is gone
		inviteRepo := repository.NewInvitationRepository(dbi.DB)
		_, err = inviteRepo.GetByToken(invitation.Token)
		assert.Error(t, err)
	})
}

func TestInviteService_GetPendingInvites(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		svc := newTestInviteService(dbi)

		owner := testutil.CreateTestUser(t, dbi.DB, "pending-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Pending Space")
		testutil.CreateTestInvitation(t, dbi.DB, space.ID, owner.ID, "pending1@example.com")
		testutil.CreateTestInvitation(t, dbi.DB, space.ID, owner.ID, "pending2@example.com")

		pending, err := svc.GetPendingInvites(space.ID)
		require.NoError(t, err)
		assert.Len(t, pending, 2)
	})
}
