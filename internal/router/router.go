package router

import (
	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/shravanasati/shadowfax/internal/server"
)

var defaultNotFoundHandler server.Handler = func(r *request.Request) response.Response {
	return response.
		NewTextResponse(response.GetStatusReason(response.StatusNotFound)).
		WithStatusCode(response.StatusNotFound)
}

type Middleware func(server.Handler) server.Handler

// Router is a simple HTTP router.
type Router struct {
	trees           map[string]*TrieNode
	notFoundHandler server.Handler
	middlewares     []Middleware
}

// Creates a new router.
func NewRouter() *Router {
	methodTreeMap := map[string]*TrieNode{
		"GET":     NewTrieNode(),
		"POST":    NewTrieNode(),
		"PUT":     NewTrieNode(),
		"PATCH":   NewTrieNode(),
		"DELETE":  NewTrieNode(),
		"OPTIONS": NewTrieNode(),
		"HEAD":    NewTrieNode(),
		"ANY":     NewTrieNode(),
	}
	return &Router{trees: methodTreeMap, notFoundHandler: defaultNotFoundHandler, middlewares: []Middleware{}}
}

// Get registers a new GET route.
func (r *Router) Get(path string, handler server.Handler) {
	r.trees["GET"].AddRoute(path, handler)
}

// Post registers a new POST route.
func (r *Router) Post(path string, handler server.Handler) {
	r.trees["POST"].AddRoute(path, handler)
}

// Put registers a new PUT route.
func (r *Router) Put(path string, handler server.Handler) {
	r.trees["PUT"].AddRoute(path, handler)
}

// Patch registers a new PATCH route.
func (r *Router) Patch(path string, handler server.Handler) {
	r.trees["PATCH"].AddRoute(path, handler)
}

// Delete registers a new DELETE route.
func (r *Router) Delete(path string, handler server.Handler) {
	r.trees["DELETE"].AddRoute(path, handler)
}

// Options registers a new OPTIONS route.
func (r *Router) Options(path string, handler server.Handler) {
	r.trees["OPTIONS"].AddRoute(path, handler)
}

// Head registers a new HEAD route.
func (r *Router) Head(path string, handler server.Handler) {
	r.trees["HEAD"].AddRoute(path, handler)
}

// Handle registers a new route for any HTTP method.
func (r *Router) Handle(path string, handler server.Handler) {
	r.trees["ANY"].AddRoute(path, handler)
}

// NotFound sets the handler for when no route is found.
func (r *Router) NotFound(handler server.Handler) {
	r.notFoundHandler = handler
}

// Use adds middleware to the router.
func (r *Router) Use(m ...Middleware) {
	r.middlewares = append(r.middlewares, m...)
}

func (r *Router) chain(h server.Handler) server.Handler {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return h
}

// Handler returns a server.Handler function that routes incoming requests to their
// corresponding handlers based on HTTP method and URL path.
//
// The routing logic follows this priority order:
//   1. Exact method and path match
//   2. For HEAD requests, attempts to use GET handler with body removed
//   3. Falls back to "ANY" method handler if available
//   4. Returns 405 Method Not Allowed if path exists for other methods
//   5. Returns 404 Not Found if no matching route exists
//
// Path parameters are extracted during route matching and added to the request
// context. The handler applies any configured middleware chain before executing
// the routing logic.
func (router *Router) Handler() server.Handler {
	routingHandler := func(r *request.Request) response.Response {
		reqMethod := r.Method
		path := r.Target

		// try exact method first
		handler, params := router.trees[reqMethod].Match(path)
		if handler != nil {
			r.PathParams = params
			return handler(r)
		}

		// auto handle head using get, if the specialised head handler doesnt exist
		if reqMethod == "HEAD" {
			getHandler, params := router.trees["GET"].Match(path)
			if getHandler != nil {
				r.PathParams = params
				resp := getHandler(r)
				return resp.WithBody(nil) // remove body
			}
		}

		// general method handler
		handler, params = router.trees["ANY"].Match(path)
		if handler != nil {
			r.PathParams = params
			return handler(r)
		}

		// check for method not allowed
		for method, tree := range router.trees {
			if method == reqMethod || method == "ANY" {
				// skip running trie search against already tried methods
				continue
			}
			handler, _ := tree.Match(path)
			if handler != nil {
				return response.
					NewTextResponse(response.GetStatusReason(response.StatusMethodNotAllowed)).
					WithStatusCode(response.StatusMethodNotAllowed)
			}
		}

		// 404 not found
		return router.notFoundHandler(r)
	}

	return router.chain(routingHandler)
}
