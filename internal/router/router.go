package router

import "net/http"

type Middleware func(http.Handler) http.Handler

type Group struct {
	prefix     string
	middleware []Middleware
	mux        *http.ServeMux
}

func newGroup(mux *http.ServeMux, prefix string, mw []Middleware) *Group {
	return &Group{prefix: prefix, middleware: mw, mux: mux}
}

func (g *Group) Use(mw ...Middleware) {
	g.middleware = append(g.middleware, mw...)
}

type Method string

const (
	MethodGet    Method = "GET"
	MethodPost   Method = "POST"
	MethodPut    Method = "PUT"
	MethodDelete Method = "DELETE"
	MethodPatch  Method = "PATCH"
)

func (g *Group) Handle(method Method, path string, handler http.HandlerFunc) {
	pattern := string(method) + " " + g.prefix + path
	wrapped := chain(handler, g.middleware)
	g.mux.Handle(pattern, wrapped)
}
func (g *Group) Get(path string, h http.HandlerFunc)    { g.Handle(MethodGet, path, h) }
func (g *Group) Post(path string, h http.HandlerFunc)   { g.Handle(MethodPost, path, h) }
func (g *Group) Put(path string, h http.HandlerFunc)    { g.Handle(MethodPut, path, h) }
func (g *Group) Patch(path string, h http.HandlerFunc)  { g.Handle(MethodPatch, path, h) }
func (g *Group) Delete(path string, h http.HandlerFunc) { g.Handle(MethodDelete, path, h) }

// SubGroup creates a nested group with accumulated prefix and middleware.
// Middleware added inside fn does not affect the parent group.
func (g *Group) SubGroup(prefix string, fn func(*Group)) {
	mw := make([]Middleware, len(g.middleware))
	copy(mw, g.middleware)
	sub := newGroup(g.mux, g.prefix+prefix, mw)
	fn(sub)
}

// RouteGroup is implemented by feature modules to register their routes.
type RouteGroup interface {
	Prefix() string
	Register(g *Group)
}

// MiddlewareProvider is optionally implemented by RouteGroups that need
// group-level middleware.
type MiddlewareProvider interface {
	Middlewares() []Middleware
}

type Router struct {
	mux        *http.ServeMux
	middleware []Middleware
}

func New() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (r *Router) Mux() *http.ServeMux {
	return r.mux
}

func (r *Router) Use(mw ...Middleware) {
	r.middleware = append(r.middleware, mw...)
}

// Mount registers one or more RouteGroups.
func (r *Router) Mount(groups ...RouteGroup) {
	for _, rg := range groups {
		var mw []Middleware
		if mp, ok := rg.(MiddlewareProvider); ok {
			mw = mp.Middlewares()
		}
		g := newGroup(r.mux, rg.Prefix(), mw)
		rg.Register(g)
	}
}

// Handler returns the final http.Handler with global middleware applied.
func (r *Router) Handler() http.Handler {
	if len(r.middleware) == 0 {
		return r.mux
	}
	return chain(r.mux, r.middleware)
}

func chain(base http.Handler, mws []Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		base = mws[i](base)
	}
	return base
}
