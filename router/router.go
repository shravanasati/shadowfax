package router

import (
	"maps"

	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/server"
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
	corsEnabled     bool
	cors            *corsHandler
}

// Creates a new router.
func NewRouter(opts *RouterOptions) *Router {
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

	router := &Router{
		trees:           methodTreeMap,
		notFoundHandler: defaultNotFoundHandler,
		middlewares:     []Middleware{},
	}

	if opts != nil && opts.EnableCors {
		router.corsEnabled = true
		router.cors = newCorsHandler(opts.CorsOptions)
	}

	return router
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
//  1. Exact method and path match
//  2. For HEAD requests, attempts to use GET handler with body removed
//  3. Falls back to "ANY" method handler if available
//  4. Returns 405 Method Not Allowed if path exists for other methods
//  5. Returns 404 Not Found if no matching route exists
//
// Path parameters are extracted during route matching and added to the request
// context. The handler applies any configured middleware chain before executing
// the routing logic.
func (router *Router) Handler() server.Handler {
	routingHandler := func(r *request.Request) response.Response {
		reqMethod := r.Method
		path := r.Target

		if router.corsEnabled && reqMethod == "OPTIONS" {
			origin := r.Headers.Get("Origin")
			hasOriginHeader := len(origin) != 0

			if r.Headers.Get("Access-Control-Request-Method") != "" && hasOriginHeader {
				headers := router.cors.handlePreflight(r)
				resp := response.NewBaseResponse()

				if router.cors.optionPassthrough {
					if handler, params := router.trees["OPTIONS"].Match(path); handler != nil {
						r.PathParams = params
						resp = handler(r)
					} else if handler, params := router.trees["ANY"].Match(path); handler != nil {
						r.PathParams = params
						resp = handler(r)
					} else {
						resp.WithStatusCode(response.StatusNoContent)
					}
				} else {
					resp.WithStatusCode(response.StatusNoContent)
				}

				respHeaders := resp.GetHeaders()
				for k, v := range maps.Collect(headers.All()) {
					respHeaders.Set(k, v)
				}
				return resp
			}
		}

		handler, params := router.trees[reqMethod].Match(path)
		if handler != nil {
			r.PathParams = params
			resp := handler(r)
			if router.corsEnabled {
				corsHeaders := router.cors.handleActualRequest(r)
				resp.WithHeaders(maps.Collect(corsHeaders.All()))
			}
			return resp
		}

		if reqMethod == "HEAD" {
			getHandler, params := router.trees["GET"].Match(path)
			if getHandler != nil {
				r.PathParams = params
				resp := getHandler(r)
				if router.corsEnabled {
					corsHeaders := router.cors.handleActualRequest(r)
					resp.WithHeaders(maps.Collect(corsHeaders.All()))
				}
				return resp.WithBody(nil)
			}
		}

		handler, params = router.trees["ANY"].Match(path)
		if handler != nil {
			r.PathParams = params
			resp := handler(r)
			if router.corsEnabled {
				corsHeaders := router.cors.handleActualRequest(r)
				resp.WithHeaders(maps.Collect(corsHeaders.All()))
			}
			return resp
		}

		for method, tree := range router.trees {
			if method == reqMethod || method == "ANY" {
				continue
			}
			handler, _ := tree.Match(path)
			if handler != nil {
				return response.
					NewTextResponse(response.GetStatusReason(response.StatusMethodNotAllowed)).
					WithStatusCode(response.StatusMethodNotAllowed)
			}
		}

		return router.notFoundHandler(r)
	}

	return router.chain(routingHandler)
}
