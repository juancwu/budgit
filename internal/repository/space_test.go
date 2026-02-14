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

func TestSpaceRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "space-create@example.com", nil)

		now := time.Now()
		space := &model.Space{
			ID:        uuid.NewString(),
			Name:      "My Space",
			OwnerID:   user.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(space)
		require.NoError(t, err)

		fetched, err := repo.ByID(space.ID)
		require.NoError(t, err)
		assert.Equal(t, "My Space", fetched.Name)
		assert.Equal(t, user.ID, fetched.OwnerID)

		isMember, err := repo.IsMember(space.ID, user.ID)
		require.NoError(t, err)
		assert.True(t, isMember, "owner should be a member after Create")
	})
}

func TestSpaceRepository_ByUserID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "space-byuser@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "User Space")

		spaces, err := repo.ByUserID(user.ID)
		require.NoError(t, err)
		require.Len(t, spaces, 1)
		assert.Equal(t, space.ID, spaces[0].ID)
	})
}

func TestSpaceRepository_AddMember(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceRepository(dbi.DB)

		owner := testutil.CreateTestUser(t, dbi.DB, "space-owner@example.com", nil)
		member := testutil.CreateTestUser(t, dbi.DB, "space-member@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Shared Space")

		err := repo.AddMember(space.ID, member.ID, model.RoleMember)
		require.NoError(t, err)

		isMember, err := repo.IsMember(space.ID, member.ID)
		require.NoError(t, err)
		assert.True(t, isMember)
	})
}

func TestSpaceRepository_RemoveMember(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceRepository(dbi.DB)

		owner := testutil.CreateTestUser(t, dbi.DB, "remove-owner@example.com", nil)
		member := testutil.CreateTestUser(t, dbi.DB, "remove-member@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Remove Test")

		err := repo.AddMember(space.ID, member.ID, model.RoleMember)
		require.NoError(t, err)

		err = repo.RemoveMember(space.ID, member.ID)
		require.NoError(t, err)

		isMember, err := repo.IsMember(space.ID, member.ID)
		require.NoError(t, err)
		assert.False(t, isMember)
	})
}

func TestSpaceRepository_GetMembers(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceRepository(dbi.DB)

		owner, _ := testutil.CreateTestUserWithProfile(t, dbi.DB, "members-owner@example.com", "Owner")
		member, _ := testutil.CreateTestUserWithProfile(t, dbi.DB, "members-member@example.com", "Member")
		space := testutil.CreateTestSpace(t, dbi.DB, owner.ID, "Members Space")

		err := repo.AddMember(space.ID, member.ID, model.RoleMember)
		require.NoError(t, err)

		members, err := repo.GetMembers(space.ID)
		require.NoError(t, err)
		require.Len(t, members, 2)

		// The query orders by role DESC (owner first), then joined_at ASC.
		assert.Equal(t, model.RoleOwner, members[0].Role)
		assert.Equal(t, "Owner", members[0].Name)
		assert.Equal(t, model.RoleMember, members[1].Role)
		assert.Equal(t, "Member", members[1].Name)
	})
}

func TestSpaceRepository_UpdateName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "space-rename@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Old Name")

		err := repo.UpdateName(space.ID, "New Name")
		require.NoError(t, err)

		fetched, err := repo.ByID(space.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Name", fetched.Name)
	})
}

func TestSpaceRepository_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewSpaceRepository(dbi.DB)

		_, err := repo.ByID("nonexistent-id")
		assert.ErrorIs(t, err, ErrSpaceNotFound)
	})
}
