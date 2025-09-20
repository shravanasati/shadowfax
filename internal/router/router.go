package router

import (
	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/shravanasati/shadowfax/internal/server"
)

type Router struct {
	trees map[string]*TrieNode
}

func NewRouter() *Router {
	methodTreeMap := map[string]*TrieNode{
		"GET":    NewTrieNode(),
		"POST":   NewTrieNode(),
		"PUT":    NewTrieNode(),
		"PATCH":  NewTrieNode(),
		"DELETE": NewTrieNode(),
		"ANY":    NewTrieNode(),
	}
	return &Router{trees: methodTreeMap}
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

func (router *Router) Handler() server.Handler {
	return func(r *request.Request) response.Response {
		method := r.Method
		path := r.Target

		// try exact method first
		handler, params := router.trees[method].Match(path)
		if handler != nil {
			r.Params = params
			return handler(r)
		}

		// general method handler
		handler, params = router.trees["ANY"].Match(path)
		if handler != nil {
			r.Params = params
			return handler(r)
		}

		// 404 not found
		return response.
			NewTextResponse(response.GetStatusReason(response.StatusNotFound)).
			WithStatusCode(response.StatusNotFound)
	}
}
