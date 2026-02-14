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

func TestTokenRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTokenRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "token-create@example.com", nil)

		token := &model.Token{
			ID:        uuid.NewString(),
			UserID:    user.ID,
			Type:      model.TokenTypeEmailVerify,
			Token:     uuid.NewString(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
			CreatedAt: time.Now(),
		}

		id, err := repo.Create(token)
		require.NoError(t, err)
		assert.Equal(t, token.ID, id)
	})
}

func TestTokenRepository_ConsumeToken(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTokenRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "token-consume@example.com", nil)
		tokenString := uuid.NewString()
		testutil.CreateTestToken(t, dbi.DB, user.ID, model.TokenTypeEmailVerify, tokenString, time.Now().Add(1*time.Hour))

		consumed, err := repo.ConsumeToken(tokenString)
		require.NoError(t, err)
		assert.NotNil(t, consumed.UsedAt)
		assert.Equal(t, user.ID, consumed.UserID)
	})
}

func TestTokenRepository_ConsumeExpiredToken(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTokenRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "token-expired@example.com", nil)
		tokenString := uuid.NewString()
		testutil.CreateTestToken(t, dbi.DB, user.ID, model.TokenTypeEmailVerify, tokenString, time.Now().Add(-1*time.Hour))

		_, err := repo.ConsumeToken(tokenString)
		assert.ErrorIs(t, err, ErrTokenNotFound)
	})
}

func TestTokenRepository_DeleteByUserAndType(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewTokenRepository(dbi.DB)

		user := testutil.CreateTestUser(t, dbi.DB, "token-delete@example.com", nil)
		tokenString := uuid.NewString()
		testutil.CreateTestToken(t, dbi.DB, user.ID, model.TokenTypeEmailVerify, tokenString, time.Now().Add(1*time.Hour))

		err := repo.DeleteByUserAndType(user.ID, model.TokenTypeEmailVerify)
		require.NoError(t, err)

		// The token should no longer be consumable since it was deleted.
		_, err = repo.ConsumeToken(tokenString)
		assert.ErrorIs(t, err, ErrTokenNotFound)
	})
}
