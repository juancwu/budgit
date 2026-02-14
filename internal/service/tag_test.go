package service

import (
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagService_CreateTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		tagRepo := repository.NewTagRepository(dbi.DB)
		svc := NewTagService(tagRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-svc-create@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Svc Space")

		color := "#ff0000"
		tag, err := svc.CreateTag(space.ID, "Groceries", &color)
		require.NoError(t, err)
		assert.NotEmpty(t, tag.ID)
		assert.Equal(t, "groceries", tag.Name)
		assert.Equal(t, &color, tag.Color)
		assert.Equal(t, space.ID, tag.SpaceID)
	})
}

func TestTagService_CreateTag_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		tagRepo := repository.NewTagRepository(dbi.DB)
		svc := NewTagService(tagRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-svc-empty@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Svc Empty Space")

		tag, err := svc.CreateTag(space.ID, "", nil)
		assert.Error(t, err)
		assert.Nil(t, tag)
	})
}

func TestTagService_GetTagsForSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		tagRepo := repository.NewTagRepository(dbi.DB)
		svc := NewTagService(tagRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-svc-list@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Svc List Space")

		testutil.CreateTestTag(t, dbi.DB, space.ID, "Alpha", nil)
		testutil.CreateTestTag(t, dbi.DB, space.ID, "Beta", nil)

		tags, err := svc.GetTagsForSpace(space.ID)
		require.NoError(t, err)
		require.Len(t, tags, 2)
	})
}

func TestTagService_UpdateTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		tagRepo := repository.NewTagRepository(dbi.DB)
		svc := NewTagService(tagRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-svc-update@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Svc Update Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Old Name", nil)

		newColor := "#00ff00"
		updated, err := svc.UpdateTag(tag.ID, "New Name", &newColor)
		require.NoError(t, err)
		assert.Equal(t, "new name", updated.Name)
		assert.Equal(t, &newColor, updated.Color)
	})
}

func TestTagService_DeleteTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		tagRepo := repository.NewTagRepository(dbi.DB)
		svc := NewTagService(tagRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-svc-delete@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Svc Delete Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Doomed Tag", nil)

		err := svc.DeleteTag(tag.ID)
		require.NoError(t, err)

		tags, err := svc.GetTagsForSpace(space.ID)
		require.NoError(t, err)
		assert.Empty(t, tags)
	})
}

func TestNormalizeTagName(t *testing.T) {
	result := NormalizeTagName(" Hello World ")
	assert.Equal(t, "hello world", result)
}
