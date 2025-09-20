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

type Router struct {
	trees           map[string]*TrieNode
	notFoundHandler server.Handler
	middlewares     []Middleware
}

func NewRouter() *Router {
	methodTreeMap := map[string]*TrieNode{
		"GET":     NewTrieNode(),
		"POST":    NewTrieNode(),
		"PUT":     NewTrieNode(),
		"PATCH":   NewTrieNode(),
		"DELETE":  NewTrieNode(),
		"OPTIONS": NewTrieNode(),
		"ANY":     NewTrieNode(),
	}
	return &Router{trees: methodTreeMap, notFoundHandler: defaultNotFoundHandler, middlewares: []Middleware{}}
}

func (r *Router) Get(path string, handler server.Handler) {
	r.trees["GET"].AddRoute(path, handler)
}

func (r *Router) Post(path string, handler server.Handler) {
	r.trees["POST"].AddRoute(path, handler)
}

func (r *Router) Put(path string, handler server.Handler) {
	r.trees["PUT"].AddRoute(path, handler)
}

func (r *Router) Patch(path string, handler server.Handler) {
	r.trees["PATCH"].AddRoute(path, handler)
}

func (r *Router) Delete(path string, handler server.Handler) {
	r.trees["DELETE"].AddRoute(path, handler)

}

func (r *Router) Handle(path string, handler server.Handler) {
	r.trees["ANY"].AddRoute(path, handler)
}

func (r *Router) NotFound(handler server.Handler) {
	r.notFoundHandler = handler
}

func (r *Router) Use(m ...Middleware) {
	r.middlewares = append(r.middlewares, m...)
}

func (r *Router) chain(h server.Handler) server.Handler {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return h
}

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
