package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/config"
	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
)

// TestConfig returns a minimal config for tests. No env vars needed.
func TestConfig() *config.Config {
	return &config.Config{
		AppName:              "Budgit Test",
		AppTagline:           "Test tagline",
		AppEnv:               "test",
		AppURL:               "http://localhost:9999",
		Host:                 "127.0.0.1",
		Port:                 "9999",
		DBDriver:             "sqlite",
		DBConnection:         ":memory:",
		JWTSecret:            "test-secret-key-for-testing-only",
		JWTExpiry:            24 * time.Hour,
		TokenMagicLinkExpiry: 10 * time.Minute,
		Version:              "test",
	}
}

// AuthenticatedContext returns a context with user, profile, config, and CSRF token injected.
func AuthenticatedContext(user *model.User, profile *model.Profile) context.Context {
	ctx := context.Background()
	ctx = ctxkeys.WithUser(ctx, user)
	ctx = ctxkeys.WithProfile(ctx, profile)
	ctx = ctxkeys.WithConfig(ctx, TestConfig().Sanitized())
	ctx = ctxkeys.WithCSRFToken(ctx, "test-csrf-token")
	return ctx
}

// NewAuthenticatedRequest creates an HTTP request with auth context and optional form values.
// CSRF token is automatically added to form values for POST requests.
func NewAuthenticatedRequest(t *testing.T, method, target string, user *model.User, profile *model.Profile, formValues url.Values) *http.Request {
	t.Helper()

	var req *http.Request

	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		if formValues == nil {
			formValues = url.Values{}
		}
		formValues.Set("csrf_token", "test-csrf-token")
		body := strings.NewReader(formValues.Encode())
		req = httptest.NewRequest(method, target, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}

	ctx := AuthenticatedContext(user, profile)
	req = req.WithContext(ctx)

	return req
}

// NewHTMXRequest adds HX-Request header to a request.
func NewHTMXRequest(req *http.Request) *http.Request {
	req.Header.Set("HX-Request", "true")
	return req
}
