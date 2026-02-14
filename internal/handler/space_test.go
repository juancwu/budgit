package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestSpaceHandler(t *testing.T, dbi testutil.DBInfo) *SpaceHandler {
	t.Helper()
	spaceRepo := repository.NewSpaceRepository(dbi.DB)
	tagRepo := repository.NewTagRepository(dbi.DB)
	listRepo := repository.NewShoppingListRepository(dbi.DB)
	itemRepo := repository.NewListItemRepository(dbi.DB)
	expenseRepo := repository.NewExpenseRepository(dbi.DB)
	inviteRepo := repository.NewInvitationRepository(dbi.DB)
	accountRepo := repository.NewMoneyAccountRepository(dbi.DB)
	methodRepo := repository.NewPaymentMethodRepository(dbi.DB)
	userRepo := repository.NewUserRepository(dbi.DB)
	emailSvc := service.NewEmailService(nil, "test@example.com", "http://localhost:9999", "Budgit Test", false)
	return NewSpaceHandler(
		service.NewSpaceService(spaceRepo),
		service.NewTagService(tagRepo),
		service.NewShoppingListService(listRepo, itemRepo),
		service.NewExpenseService(expenseRepo),
		service.NewInviteService(inviteRepo, spaceRepo, userRepo, emailSvc),
		service.NewMoneyAccountService(accountRepo),
		service.NewPaymentMethodService(methodRepo),
	)
}

func TestSpaceHandler_CreateList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists", user, profile, url.Values{"name": {"Groceries"}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateList(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSpaceHandler_CreateList_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists", user, profile, url.Values{"name": {""}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateList(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSpaceHandler_DeleteList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Groceries")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists/"+list.ID+"?from=card", user, profile, nil)
		req.SetPathValue("spaceID", space.ID)
		req.SetPathValue("listID", list.ID)

		w := httptest.NewRecorder()
		h.DeleteList(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSpaceHandler_AddItemToList(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		list := testutil.CreateTestShoppingList(t, dbi.DB, space.ID, "Groceries")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/lists/"+list.ID+"/items", user, profile, url.Values{"name": {"Milk"}})
		req.SetPathValue("spaceID", space.ID)
		req.SetPathValue("listID", list.ID)

		w := httptest.NewRecorder()
		h.AddItemToList(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSpaceHandler_CreateTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/tags", user, profile, url.Values{"name": {"food"}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateTag(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSpaceHandler_DeleteTag(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")
		tag := testutil.CreateTestTag(t, dbi.DB, space.ID, "food", nil)

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/tags/"+tag.ID, user, profile, nil)
		req.SetPathValue("spaceID", space.ID)
		req.SetPathValue("tagID", tag.ID)

		w := httptest.NewRecorder()
		h.DeleteTag(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSpaceHandler_CreateAccount(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/accounts", user, profile, url.Values{"name": {"Savings"}})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreateAccount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSpaceHandler_CreatePaymentMethod(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestSpaceHandler(t, dbi)
		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test")
		space := testutil.CreateTestSpace(t, dbi.DB, user.ID, "Test Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/spaces/"+space.ID+"/payment-methods", user, profile, url.Values{
			"name":      {"Visa"},
			"type":      {"credit"},
			"last_four": {"4242"},
		})
		req.SetPathValue("spaceID", space.ID)

		w := httptest.NewRecorder()
		h.CreatePaymentMethod(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
