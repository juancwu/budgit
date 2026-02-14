package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHomeHandler_HomePage_Guest(t *testing.T) {
	h := NewHomeHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.Background()
	ctx = ctxkeys.WithConfig(ctx, testutil.TestConfig().Sanitized())
	ctx = ctxkeys.WithCSRFToken(ctx, "test")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.HomePage(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/auth", w.Header().Get("Location"))
}

func TestHomeHandler_HomePage_Authenticated(t *testing.T) {
	h := NewHomeHandler()

	user := &model.User{ID: "user-1", Email: "test@example.com"}
	profile := &model.Profile{ID: "prof-1", UserID: "user-1", Name: "Test"}
	req := testutil.NewAuthenticatedRequest(t, http.MethodGet, "/", user, profile, nil)

	w := httptest.NewRecorder()
	h.HomePage(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/app/dashboard", w.Header().Get("Location"))
}
