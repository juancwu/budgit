package router

import (
	"net/http"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/middleware"
)

type Group struct {
	prefix     string
	middleware []middleware.Middleware
	limiter    *middleware.RateLimiter
	parent     *Group
	mux        *http.ServeMux
}

func (g *Group) Use(mw ...middleware.Middleware) {
	g.middleware = append(g.middleware, mw...)
}

// RateLimit sets a rate limit on this group. It runs before any middleware
// in the chain, including inherited middleware from parent groups.
// Parent group rate limits are checked first (root → leaf order).
func (g *Group) RateLimit(limit int, window time.Duration) {
	g.limiter = middleware.NewRateLimiter(limit, window)
}

type Method string

const (
	MethodGet    Method = "GET"
	MethodPost   Method = "POST"
	MethodPut    Method = "PUT"
	MethodDelete Method = "DELETE"
	MethodPatch  Method = "PATCH"
)

func (g *Group) Handle(method Method, path string, handler http.HandlerFunc, mw ...middleware.Middleware) {
	// Build chain: [rate limiters root→self] → [middleware root→self] → [route mw] → handler
	rateLimiters := g.collectRateLimiters()
	middlewares := g.collectMiddleware()
	middlewares = append(middlewares, mw...)

	chain := append(rateLimiters, middlewares...)

	pattern := string(method) + " " + g.prefix + path
	wrapped := middleware.Chain(handler, chain...)
	g.mux.Handle(pattern, wrapped)
}

func (g *Group) Get(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	g.Handle(MethodGet, path, h, mw...)
}
func (g *Group) Post(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	g.Handle(MethodPost, path, h, mw...)
}
func (g *Group) Put(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	g.Handle(MethodPut, path, h, mw...)
}
func (g *Group) Patch(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	g.Handle(MethodPatch, path, h, mw...)
}
func (g *Group) Delete(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	g.Handle(MethodDelete, path, h, mw...)
}

// SubGroup creates a nested group. It inherits rate limits and middleware
// from the parent via the parent pointer (not by copying).
func (g *Group) SubGroup(prefix string, fn func(*Group)) {
	sub := &Group{
		prefix: g.prefix + prefix,
		parent: g,
		mux:    g.mux,
	}
	fn(sub)
}

// collectRateLimiters walks up the parent chain and returns rate limit
// middleware in root → leaf order.
func (g *Group) collectRateLimiters() []middleware.Middleware {
	var result []middleware.Middleware
	if g.parent != nil {
		result = g.parent.collectRateLimiters()
	}
	if g.limiter != nil {
		result = append(result, g.limiter.Middleware())
	}
	return result
}

// collectMiddleware walks up the parent chain and returns middleware
// in root → leaf order.
func (g *Group) collectMiddleware() []middleware.Middleware {
	var result []middleware.Middleware
	if g.parent != nil {
		result = g.parent.collectMiddleware()
	}
	result = append(result, g.middleware...)
	return result
}

type Router struct {
	root *Group
	mux  *http.ServeMux
}

func New() *Router {
	mux := http.NewServeMux()
	return &Router{
		mux: mux,
		root: &Group{
			mux: mux,
		},
	}
}

func (r *Router) Mux() *http.ServeMux {
	return r.mux
}

func (r *Router) Use(mw ...middleware.Middleware) {
	r.root.Use(mw...)
}

// Group creates a route group that inherits global middleware from the router.
func (r *Router) Group(prefix string, fn func(*Group)) {
	r.root.SubGroup(prefix, fn)
}

// Handler returns the final http.Handler. All middleware is already applied
// per-route through the group hierarchy, so this just returns the mux.
func (r *Router) Handler() http.Handler {
	return r.mux
}

func (r *Router) Handle(method Method, path string, handler http.HandlerFunc, mw ...middleware.Middleware) {
	r.root.Handle(method, path, handler, mw...)
}

func (r *Router) Get(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	r.root.Get(path, h, mw...)
}
func (r *Router) Post(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	r.root.Post(path, h, mw...)
}
func (r *Router) Put(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	r.root.Put(path, h, mw...)
}
func (r *Router) Patch(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	r.root.Patch(path, h, mw...)
}
func (r *Router) Delete(path string, h http.HandlerFunc, mw ...middleware.Middleware) {
	r.root.Delete(path, h, mw...)
}
