package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestDashboardHandler_DashboardPage(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		spaceSvc := service.NewSpaceService(spaceRepo)
		expenseSvc := service.NewExpenseService(expenseRepo)
		h := NewDashboardHandler(spaceSvc, expenseSvc)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test User")
		testutil.CreateTestSpace(t, dbi.DB, user.ID, "My Space")

		req := testutil.NewAuthenticatedRequest(t, http.MethodGet, "/app/dashboard", user, profile, nil)
		w := httptest.NewRecorder()
		h.DashboardPage(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestDashboardHandler_CreateSpace(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		spaceSvc := service.NewSpaceService(spaceRepo)
		expenseSvc := service.NewExpenseService(expenseRepo)
		h := NewDashboardHandler(spaceSvc, expenseSvc)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test User")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/dashboard/spaces", user, profile, url.Values{"name": {"New Space"}})
		w := httptest.NewRecorder()
		h.CreateSpace(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, strings.HasPrefix(w.Header().Get("HX-Redirect"), "/app/spaces/"))
	})
}

func TestDashboardHandler_CreateSpace_EmptyName(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		spaceRepo := repository.NewSpaceRepository(dbi.DB)
		expenseRepo := repository.NewExpenseRepository(dbi.DB)
		spaceSvc := service.NewSpaceService(spaceRepo)
		expenseSvc := service.NewExpenseService(expenseRepo)
		h := NewDashboardHandler(spaceSvc, expenseSvc)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test User")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/dashboard/spaces", user, profile, url.Values{"name": {""}})
		w := httptest.NewRecorder()
		h.CreateSpace(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}
