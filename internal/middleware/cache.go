package middleware

import "net/http"

// CacheStatic wraps a handler to set long-lived cache headers for static assets.
// Assets use query-string cache busting (?v=<timestamp>), so it's safe to cache
// them indefinitely — the URL changes when the content changes.
func CacheStatic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		h.ServeHTTP(w, r)
	})
}

// NoCacheDynamic sets Cache-Control: no-cache on responses so browsers always
// revalidate with the server. This prevents stale HTML from being shown after
// navigation (e.g. back button) while still allowing conditional requests.
func NoCacheDynamic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip static assets — they're handled by CacheStatic.
		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/assets/" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}
