package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
)

// AuthMiddleware checks for JWT token and adds user to context if valid
func AuthMiddleware(authService *service.AuthService, userService *service.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get JWT from cookie
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				// No cookie, continue without auth
				next.ServeHTTP(w, r)
				return
			}

			// Verify token
			claims, err := authService.VerifyJWT(cookie.Value)
			if err != nil {
				// Invalid token, clear cookie and continue
				authService.ClearJWTCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			// Get user ID from claims
			userID, ok := claims["user_id"].(string)
			if !ok {
				authService.ClearJWTCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			// Fetch user from database
			user, err := userService.ByID(userID)
			if err != nil {
				authService.ClearJWTCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			// Security: Remove password hash from context
			user.PasswordHash = nil

			// Add user to context
			ctx := ctxkeys.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireGuest ensures request is not authenticated
func RequireGuest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := ctxkeys.User(r.Context())
		if user != nil {
			redirect(w, r, "/app/dashboard", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RequireAuth ensures the user is authenticated and has completed onboarding
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := ctxkeys.User(r.Context())
		if user == nil {
			redirect(w, r, "/auth", http.StatusSeeOther)
			return
		}

		// Check if user has completed onboarding (name set)
		if (user.Name == nil || *user.Name == "") && r.URL.Path != "/auth/onboarding" {
			redirect(w, r, "/auth/onboarding", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	}
}
