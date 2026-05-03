package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRedirectHandler_RederictToSpaces(t *testing.T) {
	h := NewRedirectHandler()

	req := testutil.NewRequest(t, http.MethodPost, "/app/dashboard", nil)
	w := httptest.NewRecorder()
	h.Spaces(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "/app/home", w.Header().Get("Location"))
}
