package service

import (
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpaceService_CreateSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "create-space@example.com", nil)

		space, err := svc.CreateSpace("My Space", user.ID)
		require.NoError(t, err)
		assert.Equal(t, "My Space", space.Name)
		assert.Equal(t, user.ID, space.OwnerID)
		assert.NotEmpty(t, space.ID)
	})
}

func TestSpaceService_CreateSpace_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "empty-name@example.com", nil)

		_, err := svc.CreateSpace("", user.ID)
		assert.Error(t, err)
	})
}

func TestSpaceService_EnsurePersonalSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "personal@example.com", nil)

		// First call creates the personal space
		space1, err := svc.EnsurePersonalSpace(user)
		require.NoError(t, err)
		assert.Equal(t, PersonalSpaceName, space1.Name)

		// Second call returns the same space (idempotent)
		space2, err := svc.EnsurePersonalSpace(user)
		require.NoError(t, err)
		assert.Equal(t, space1.ID, space2.ID)
	})
}

func TestSpaceService_GetSpacesForUser(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "getspaces@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		spaces, err := svc.GetSpacesForUser(user.ID)
		require.NoError(t, err)
		require.Len(t, spaces, 1)
		assert.Equal(t, space.ID, spaces[0].ID)
		assert.Equal(t, "Test Space", spaces[0].Name)
	})
}

func TestSpaceService_IsMember(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "ismember@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Member Check Space")

		// Owner should be a member
		isMember, err := svc.IsMember(user.ID, space.ID)
		require.NoError(t, err)
		assert.True(t, isMember)

		// Random ID should not be a member
		isMember, err = svc.IsMember("nonexistent-user-id", space.ID)
		require.NoError(t, err)
		assert.False(t, isMember)
	})
}

func TestSpaceService_GetMembers(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		owner, _ := testutil.CreateTestUserWithProfile(t, dbi.DB, "owner-members@example.com", "Owner")
		member, _ := testutil.CreateTestUserWithProfile(t, dbi.DB, "member-members@example.com", "Member")
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Members Space")

		// Add second user as a member
		_, err := dbi.DB.Exec(
			`INSERT INTO space_members (space_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)`,
			space.ID, member.ID, model.RoleMember, time.Now(),
		)
		require.NoError(t, err)

		members, err := svc.GetMembers(space.ID)
		require.NoError(t, err)
		require.Len(t, members, 2)

		// The query orders by role DESC (owner first), then joined_at ASC
		assert.Equal(t, model.RoleOwner, members[0].Role)
		assert.Equal(t, "Owner", members[0].Name)
		assert.Equal(t, model.RoleMember, members[1].Role)
		assert.Equal(t, "Member", members[1].Name)
	})
}

func TestSpaceService_RemoveMember(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		owner := testutil.CreateTestUser(t, dbi.DB, "remove-owner@example.com", nil)
		member := testutil.CreateTestUser(t, dbi.DB, "remove-member@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Remove Space")

		// Add member
		_, err := dbi.DB.Exec(
			`INSERT INTO space_members (space_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)`,
			space.ID, member.ID, model.RoleMember, time.Now(),
		)
		require.NoError(t, err)

		// Verify member was added
		isMember, err := svc.IsMember(member.ID, space.ID)
		require.NoError(t, err)
		assert.True(t, isMember)

		// Remove member
		err = svc.RemoveMember(space.ID, member.ID)
		require.NoError(t, err)

		// Verify member was removed
		isMember, err = svc.IsMember(member.ID, space.ID)
		require.NoError(t, err)
		assert.False(t, isMember)
	})
}

func TestSpaceService_UpdateSpaceName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		svc := NewSpaceService(spaceRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "rename@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Old Name")

		err := svc.UpdateSpaceName(space.ID, "New Name")
		require.NoError(t, err)

		// Verify name was updated by fetching the space
		fetched, err := svc.GetSpace(space.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Name", fetched.Name)
	})
}
