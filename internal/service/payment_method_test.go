package service

import (
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentMethodService_CreateMethod(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
		svc := NewPaymentMethodService(methodRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "pm-svc-create@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "PM Svc Space")

		method, err := svc.CreateMethod(CreatePaymentMethodDTO{
			SpaceID:   space.ID,
			Name:      "Visa Card",
			Type:      model.PaymentMethodTypeCredit,
			LastFour:  "4242",
			CreatedBy: user.ID,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, method.ID)
		assert.Equal(t, "Visa Card", method.Name)
		assert.Equal(t, model.PaymentMethodTypeCredit, method.Type)
		require.NotNil(t, method.LastFour)
		assert.Equal(t, "4242", *method.LastFour)
	})
}

func TestPaymentMethodService_CreateMethod_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
		svc := NewPaymentMethodService(methodRepo)

		method, err := svc.CreateMethod(CreatePaymentMethodDTO{
			SpaceID:   "some-space",
			Name:      "",
			Type:      model.PaymentMethodTypeCredit,
			LastFour:  "4242",
			CreatedBy: "some-user",
		})
		assert.Error(t, err)
		assert.Nil(t, method)
	})
}

func TestPaymentMethodService_CreateMethod_InvalidType(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
		svc := NewPaymentMethodService(methodRepo)

		method, err := svc.CreateMethod(CreatePaymentMethodDTO{
			SpaceID:   "some-space",
			Name:      "Bad Type Card",
			Type:      "invalid",
			LastFour:  "4242",
			CreatedBy: "some-user",
		})
		assert.Error(t, err)
		assert.Nil(t, method)
	})
}

func TestPaymentMethodService_CreateMethod_InvalidLastFour(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
		svc := NewPaymentMethodService(methodRepo)

		method, err := svc.CreateMethod(CreatePaymentMethodDTO{
			SpaceID:   "some-space",
			Name:      "Short Digits Card",
			Type:      model.PaymentMethodTypeDebit,
			LastFour:  "12",
			CreatedBy: "some-user",
		})
		assert.Error(t, err)
		assert.Nil(t, method)
	})
}

func TestPaymentMethodService_GetMethodsForSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
		svc := NewPaymentMethodService(methodRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "pm-svc-list@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "PM Svc List Space")

		testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Visa", model.PaymentMethodTypeCredit, user.ID)
		testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Debit", model.PaymentMethodTypeDebit, user.ID)

		methods, err := svc.GetMethodsForSpace(space.ID)
		require.NoError(t, err)
		assert.Len(t, methods, 2)
	})
}

func TestPaymentMethodService_UpdateMethod(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
		svc := NewPaymentMethodService(methodRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "pm-svc-update@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "PM Svc Update Space")
		method := testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Old Card", model.PaymentMethodTypeCredit, user.ID)

		updated, err := svc.UpdateMethod(UpdatePaymentMethodDTO{
			ID:       method.ID,
			Name:     "New Card",
			Type:     model.PaymentMethodTypeDebit,
			LastFour: "9999",
		})
		require.NoError(t, err)
		assert.Equal(t, "New Card", updated.Name)
		assert.Equal(t, model.PaymentMethodTypeDebit, updated.Type)
		require.NotNil(t, updated.LastFour)
		assert.Equal(t, "9999", *updated.LastFour)
	})
}

func TestPaymentMethodService_DeleteMethod(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
		svc := NewPaymentMethodService(methodRepo)

		user := testutil.CreateTestUser(t, dbi.DB, "pm-svc-delete@example.com", nil)
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "PM Svc Delete Space")
		method := testutil.CreateTestPaymentMethod(t, dbi.DB, space.ID, "Doomed Card", model.PaymentMethodTypeCredit, user.ID)

		err := svc.DeleteMethod(method.ID)
		require.NoError(t, err)

		methods, err := svc.GetMethodsForSpace(space.ID)
		require.NoError(t, err)
		assert.Empty(t, methods)
	})
}
