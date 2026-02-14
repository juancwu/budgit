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

func TestPaymentMethodRepository_Create(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewPaymentMethodRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		lastFour := "4242"
		now := time.Now()
		method := &model.PaymentMethod{
			ID:        uuid.NewString(),
			SpaceID:   space.ID,
			Name:      "Visa Gold",
			Type:      model.PaymentMethodTypeCredit,
			LastFour:  &lastFour,
			CreatedBy: user.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := repo.Create(method)
		require.NoError(t, err)

		fetched, err := repo.GetByID(method.ID)
		require.NoError(t, err)
		assert.Equal(t, method.ID, fetched.ID)
		assert.Equal(t, space.ID, fetched.SpaceID)
		assert.Equal(t, "Visa Gold", fetched.Name)
		assert.Equal(t, model.PaymentMethodTypeCredit, fetched.Type)
		require.NotNil(t, fetched.LastFour)
		assert.Equal(t, "4242", *fetched.LastFour)
		assert.Equal(t, user.ID, fetched.CreatedBy)
	})
}

func TestPaymentMethodRepository_GetBySpaceID(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewPaymentMethodRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Visa", model.PaymentMethodTypeCredit, user.ID)
		testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Debit Card", model.PaymentMethodTypeDebit, user.ID)

		methods, err := repo.GetBySpaceID(space.ID)
		require.NoError(t, err)
		assert.Len(t, methods, 2)
	})
}

func TestPaymentMethodRepository_Update(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewPaymentMethodRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		method := testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Old Card", model.PaymentMethodTypeCredit, user.ID)

		method.Name = "New Card"
		method.Type = model.PaymentMethodTypeDebit
		err := repo.Update(method)
		require.NoError(t, err)

		fetched, err := repo.GetByID(method.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Card", fetched.Name)
		assert.Equal(t, model.PaymentMethodTypeDebit, fetched.Type)
	})
}

func TestPaymentMethodRepository_Delete(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		repo := NewPaymentMethodRepository(dbi.DB)
		user := testutil.CreateTestUser(t, dbi.DB, "test@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		method := testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "To Delete", model.PaymentMethodTypeCredit, user.ID)

		err := repo.Delete(method.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(method.ID)
		assert.ErrorIs(t, err, ErrPaymentMethodNotFound)
	})
}
