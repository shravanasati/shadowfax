// used by the router to match on paths

package router

import (
	"strings"

	"github.com/shravanasati/shadowfax/internal/server"
)

type TrieNode struct {
	// static children
	children map[string]*TrieNode

	// parameter segment, eg. :id
	paramChild *TrieNode
	paramName  string

	// wildcard segment, eg. *file
	wildcardChild *TrieNode
	wildcardName  string

	method string
	// route handler to call
	handler server.Handler
}

func NewTrieNode() *TrieNode {
	return &TrieNode{children: make(map[string]*TrieNode)}
}

// AddRoute adds a new route with its handler to the trie
func (n *TrieNode) AddRoute(path string, handler server.Handler) {
	currentNode := n
	segments := strings.SplitSeq(strings.Trim(path, "/"), "/")

	for segment := range segments {
		if segment == "" {
			continue
		}

		// determine segment type
		switch {
		case strings.HasPrefix(segment, ":"):
			// parameter
			paramName := strings.TrimPrefix(segment, ":")
			if currentNode.paramChild == nil {
				currentNode.paramChild = NewTrieNode()
			}
			currentNode.paramName = paramName
			currentNode = currentNode.paramChild

		case strings.HasPrefix(segment, "*"):
			// wildcard
			wildcardName := strings.TrimPrefix(segment, "*")
			if currentNode.wildcardChild == nil {
				currentNode.wildcardChild = NewTrieNode()
			}
			currentNode.wildcardName = wildcardName
			currentNode = currentNode.wildcardChild

		default:
			// static
			if _, ok := currentNode.children[segment]; !ok {
				currentNode.children[segment] = NewTrieNode()
			}
			currentNode = currentNode.children[segment]
		}

	}

	currentNode.handler = handler
}

// Match finds a handler for a given path and extracts any parameters
func (n *TrieNode) Match(path string) (server.Handler, map[string]string,) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	currentNode := n
	params := make(map[string]string)

	for i, segment := range segments {
		if segment == "" {
			continue
		}

		// static paths first
		if child, ok := currentNode.children[segment]; ok {
			currentNode = child
			continue
		}

		// parameter paths next
		if currentNode.paramChild != nil {
			params[currentNode.paramName] = segment
			currentNode = currentNode.paramChild
			continue
		}

		// wildcard match final
		if currentNode.wildcardChild != nil {
			// matches the whole path
			params[currentNode.wildcardName] = strings.Join(segments[i:], "/")
			currentNode = currentNode.wildcardChild
			return currentNode.handler, params
		}

		// no match found
		return nil, nil
	}

	// final node
	if currentNode == nil {
		return nil, nil
	}
	return currentNode.handler, params
}
