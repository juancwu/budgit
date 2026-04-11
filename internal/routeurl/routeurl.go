// Package routeurl resolves named routes to concrete URL paths.
// It is intentionally a dependency-free leaf package so templates in
// internal/ui/... can import it without creating a cycle through the
// routes/router/middleware/handler graph.
package routeurl

import (
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]string{}
)

// Register associates a name with a pattern path (e.g. "/join/{token}").
// Called by router.Route.Name at route-registration time.
func Register(name, path string) {
	mu.Lock()
	registry[name] = path
	mu.Unlock()
}

// Reset clears the registry. Intended for tests that want isolation
// between SetupRoutes calls.
func Reset() {
	mu.Lock()
	registry = map[string]string{}
	mu.Unlock()
}

// URL resolves a named route, substituting named wildcard segments from
// key/value pairs. Unknown names return "#" so templates degrade gracefully
// instead of panicking.
//
// Each key is replaced exactly once — net/http.ServeMux forbids duplicate
// wildcard names in a single pattern, so this is safe by construction.
//
//	routeurl.URL("page.public.join-space", "token", tok) // "/join/<tok>"
func URL(name string, kv ...string) string {
	mu.RLock()
	path, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return "#"
	}
	for i := 0; i+1 < len(kv); i += 2 {
		key, val := kv[i], kv[i+1]
		path = strings.Replace(path, "{"+key+"...}", val, 1)
		path = strings.Replace(path, "{"+key+"}", val, 1)
	}
	return path
}
