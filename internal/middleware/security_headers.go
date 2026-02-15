package middleware

import (
	"net/http"
)

// SecurityHeaders sets common security response headers on every response.
// Note: HSTS is handled by Caddy at the reverse proxy layer.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline' https://www.googletagmanager.com; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; "+
					"connect-src 'self' https://www.google-analytics.com; "+
					"font-src 'self'; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'")

			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

			next.ServeHTTP(w, r)
		})
	}
}
