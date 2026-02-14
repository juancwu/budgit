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

func TestTagRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTagRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-create@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Space")

		color := "#ff0000"
		now := time.Now()
		tag := &model.Tag{
			ID:        uuid.NewString(),
			SpaceID:   space.ID,
			Name:      "Groceries",
			Color:     &color,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(tag)
		require.NoError(t, err)

		fetched, err := repo.GetByID(tag.ID)
		require.NoError(t, err)
		assert.Equal(t, "Groceries", fetched.Name)
		assert.Equal(t, &color, fetched.Color)
		assert.Equal(t, space.ID, fetched.SpaceID)
	})
}

func TestTagRepository_GetBySpaceID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTagRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-list@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag List Space")

		// Create tags with names that sort alphabetically: "Alpha" < "Beta".
		testutil.CreateTestTag(t, dbi.DB, space.ID, "Beta", nil)
		testutil.CreateTestTag(t, dbi.DB, space.ID, "Alpha", nil)

		tags, err := repo.GetBySpaceID(space.ID)
		require.NoError(t, err)
		require.Len(t, tags, 2)
		assert.Equal(t, "Alpha", tags[0].Name)
		assert.Equal(t, "Beta", tags[1].Name)
	})
}

func TestTagRepository_Update(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTagRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-update@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Update Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Old Tag", nil)

		newColor := "#00ff00"
		tag.Name = "New Tag"
		tag.Color = &newColor

		err := repo.Update(tag)
		require.NoError(t, err)

		fetched, err := repo.GetByID(tag.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Tag", fetched.Name)
		assert.Equal(t, &newColor, fetched.Color)
	})
}

func TestTagRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTagRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-delete@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Delete Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "Doomed Tag", nil)

		err := repo.Delete(tag.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(tag.ID)
		assert.ErrorIs(t, err, ErrTagNotFound)
	})
}

func TestTagRepository_DuplicateTagName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTagRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "tag-dup@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Tag Dup Space")
		testutil.CreateTestTag(t, dbi.DB, space.ID, "Duplicate", nil)

		now := time.Now()
		duplicate := &model.Tag{
			ID:        uuid.NewString(),
			SpaceID:   space.ID,
			Name:      "Duplicate",
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(duplicate)
		assert.ErrorIs(t, err, ErrDuplicateTagName)
	})
}
