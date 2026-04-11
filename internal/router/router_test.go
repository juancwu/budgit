package router_test

import (
	"net/http"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/router"
	"github.com/stretchr/testify/assert"
)

// TestRouter_DuplicateWildcardPanics pins a property we rely on: net/http.ServeMux
// refuses to register a pattern that uses the same wildcard name twice. URL() in
// the routes package leans on this guarantee to replace each key exactly once, so
// if this ever regresses we want loud test failure, not silently wrong URLs.
func TestRouter_DuplicateWildcardPanics(t *testing.T) {
	r := router.New()
	assert.PanicsWithError(
		t,
		`parsing "GET /here/{token}/there/{token}": at offset 24: duplicate wildcard name "token"`,
		func() {
			r.Get("/here/{token}/there/{token}", func(http.ResponseWriter, *http.Request) {})
		},
	)
}
