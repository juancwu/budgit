package repository

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitationRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewInvitationRepository(dbi.DB)

		owner := testutil.CreateTestUser(t, dbi.DB, "inv-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Invite Space")

		now := time.Now()
		invitation := &model.SpaceInvitation{
			Token:     uuid.NewString(),
			SpaceID:   space.ID,
			InviterID: owner.ID,
			Email:     "invitee@example.com",
			Status:    model.InvitationStatusPending,
			ExpiresAt: now.Add(48 * time.Hour),
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(invitation)
		require.NoError(t, err)

		fetched, err := repo.GetByToken(invitation.Token)
		require.NoError(t, err)
		assert.Equal(t, invitation.Token, fetched.Token)
		assert.Equal(t, space.ID, fetched.SpaceID)
		assert.Equal(t, "invitee@example.com", fetched.Email)
		assert.Equal(t, model.InvitationStatusPending, fetched.Status)
	})
}

func TestInvitationRepository_GetBySpaceID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewInvitationRepository(dbi.DB)

		owner := testutil.CreateTestUser(t, dbi.DB, "inv-space-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Space Invites")
		testutil.CreateTestInvitation(t, dbi.DB, space.ID, owner.ID, "one@example.com")

		invitations, err := repo.GetBySpaceID(space.ID)
		require.NoError(t, err)
		require.Len(t, invitations, 1)
		assert.Equal(t, "one@example.com", invitations[0].Email)
	})
}

func TestInvitationRepository_UpdateStatus(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewInvitationRepository(dbi.DB)

		owner := testutil.CreateTestUser(t, dbi.DB, "inv-status-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Status Space")
		invitation := testutil.CreateTestInvitation(t, dbi.DB, space.ID, owner.ID, "status@example.com")

		err := repo.UpdateStatus(invitation.Token, model.InvitationStatusAccepted)
		require.NoError(t, err)

		fetched, err := repo.GetByToken(invitation.Token)
		require.NoError(t, err)
		assert.Equal(t, model.InvitationStatusAccepted, fetched.Status)
	})
}

func TestInvitationRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewInvitationRepository(dbi.DB)

		owner := testutil.CreateTestUser(t, dbi.DB, "inv-delete-owner@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Delete Space")
		invitation := testutil.CreateTestInvitation(t, dbi.DB, space.ID, owner.ID, "delete@example.com")

		err := repo.Delete(invitation.Token)
		require.NoError(t, err)

		_, err = repo.GetByToken(invitation.Token)
		assert.ErrorIs(t, err, ErrInvitationNotFound)
	})
}

func TestInvitationRepository_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewInvitationRepository(dbi.DB)

		_, err := repo.GetByToken("nonexistent-token")
		assert.ErrorIs(t, err, ErrInvitationNotFound)
	})
}
