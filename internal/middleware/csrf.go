package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
)

const (
	csrfCookieName = "csrf_token"
	csrfFormField  = "csrf_token"
	csrfHeader     = "X-CSRF-Token"
	csrfTokenLen   = 32
)

// CSRFProtection validates CSRF tokens on all state-changing requests
func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods (GET, HEAD, OPTIONS)
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			token, err := getOrGenerateCSRFToken(w, r)
			if err != nil {
				slog.Error("failed to generate CSRF token", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			ctx := ctxkeys.WithCSRFToken(r.Context(), token)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Skip CSRF check for webhooks (external services)
		if strings.HasPrefix(r.URL.Path, "/webhooks/") {
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token for state-changing methods (POST, PUT, PATCH, DELETE)
		token, err := getOrGenerateCSRFToken(w, r)
		if err != nil {
			slog.Error("failed to generate CSRF token", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		ctx := ctxkeys.WithCSRFToken(r.Context(), token)

		// Get submitted token - try multiple sources in priority order
		// 1. Header (HTMX automatic via meta tag)
		// 2. Form field (both application/x-www-form-urlencoded and multipart/form-data)
		// PostFormValue() automatically parses the request based on Content-Type
		submittedToken := r.Header.Get(csrfHeader)
		if submittedToken == "" {
			submittedToken = r.PostFormValue(csrfFormField)
		}

		// Validate token using constant-time comparison
		if !validCSRFToken(token, submittedToken) {
			slog.Warn("csrf validation failed",
				"path", r.URL.Path,
				"method", r.Method,
				"ip", getClientIP(r),
			)
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getOrGenerateCSRFToken retrieves existing token or generates new one
func getOrGenerateCSRFToken(w http.ResponseWriter, r *http.Request) (string, error) {
	cookie, err := r.Cookie(csrfCookieName)
	if err == nil && cookie.Value != "" && len(cookie.Value) == base64.RawURLEncoding.EncodedLen(csrfTokenLen) {
		return cookie.Value, nil
	}

	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}

	cfg := ctxkeys.Config(r.Context())
	isProduction := cfg != nil && cfg.IsProduction()

	// Set cookie with SameSite=Lax for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction, // Secure flag based on APP_ENV (safer than r.TLS behind load balancers)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	})

	return token, nil
}

// generateCSRFToken creates cryptographically secure random token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenLen)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate csrf token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// validCSRFToken performs constant-time comparison of tokens
func validCSRFToken(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
