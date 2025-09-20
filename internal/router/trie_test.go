package router

import (
	"testing"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/shravanasati/shadowfax/internal/server"
	"github.com/stretchr/testify/assert"
)

// mock handler for testing
func mockHandler(req *request.Request) response.Response {
	resp := response.NewTextResponse("ok")
	resp.WithStatusCode(response.StatusOK)
	return resp
}

func TestTrie_AddAndMatch(t *testing.T) {
	trie := NewTrieNode()
	emptyMap := make(map[string]string)

	// Define test cases
	testCases := []struct {
		routePath      string
		requestPath    string
		expectedParams map[string]string
		shouldMatch    bool
	}{
		// Static routes
		{"/home", "/home", emptyMap, true},
		{"/about/team", "/about/team", emptyMap, true},
		{"/contact", "/contact-us", emptyMap, false},

		// Parameterized routes
		{"/users/:id", "/users/123", map[string]string{"id": "123"}, true},
		{"/posts/:year/:month", "/posts/2023/10", map[string]string{"year": "2023", "month": "10"}, true},
		{"/products/:category/:product_id", "/products/books/987", map[string]string{"category": "books", "product_id": "987"}, true},

		// Wildcard routes
		{"/static/*filepath", "/static/css/style.css", map[string]string{"filepath": "css/style.css"}, true},
		{"/files/*path", "/files/documents/report.pdf", map[string]string{"path": "documents/report.pdf"}, true},

		// Mixed routes
		{"/api/v1/:resource/data", "/api/v1/users/data", map[string]string{"resource": "users"}, true},
		{"/api/:version/status", "/api/v2/status", map[string]string{"version": "v2"}, true},

		// Edge cases
		{"/", "/", emptyMap, true},
		{"/trailing/", "/trailing", emptyMap, true},
		{"/double//slash", "/double/slash", emptyMap, true},
	}

	// Add routes to the trie
	for _, tc := range testCases {
		if tc.shouldMatch {
			trie.AddRoute(tc.routePath, server.Handler(mockHandler))
		}
	}

	// Run match tests
	for _, tc := range testCases {
		t.Run(tc.requestPath, func(t *testing.T) {
			handler, params := trie.Match(tc.requestPath)

			if tc.shouldMatch {
				assert.NotNil(t, handler, "Expected a handler for path %s", tc.requestPath)
				assert.Equal(t, tc.expectedParams, params, "Expected params %v, but got %v", tc.expectedParams, params)
			} else {
				assert.Nil(t, handler, "Expected no handler for path %s", tc.requestPath)
			}
		})
	}
}

func TestTrie_NoMatch(t *testing.T) {
	trie := NewTrieNode()
	trie.AddRoute("/home", server.Handler(mockHandler))

	handler, _ := trie.Match("/nonexistent")
	assert.Nil(t, handler, "Expected no handler for a nonexistent route, but got one")
}

func TestTrie_RootHandler(t *testing.T) {
	trie := NewTrieNode()
	trie.AddRoute("/", server.Handler(mockHandler))

	handler, _ := trie.Match("/")
	assert.NotNil(t, handler, "Expected a handler for the root path, but got nil")
}
