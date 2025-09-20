package router

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/stretchr/testify/assert"
)

func parseResponse(w *httptest.ResponseRecorder) (int, string) {
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestRouter(t *testing.T) {
	router := NewRouter()

	router.Get("/home", func(r *request.Request) response.Response {
		return response.NewTextResponse("get home")
	})

	router.Post("/home", func(r *request.Request) response.Response {
		return response.NewTextResponse("post home")
	})

	router.Put("/home", func(r *request.Request) response.Response {
		return response.NewTextResponse("put home")
	})

	router.Patch("/home", func(r *request.Request) response.Response {
		return response.NewTextResponse("patch home")
	})

	router.Delete("/home", func(r *request.Request) response.Response {
		return response.NewTextResponse("delete home")
	})

	router.Get("/users/:id", func(r *request.Request) response.Response {
		id := r.Params["id"]
		return response.NewTextResponse("user " + id)
	})

	router.Handle("/any", func(r *request.Request) response.Response {
		return response.NewTextResponse("any method")
	})

	handler := router.Handler()

	testCases := []struct {
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"GET", "/home", http.StatusOK, "get home"},
		{"POST", "/home", http.StatusOK, "post home"},
		{"PUT", "/home", http.StatusOK, "put home"},
		{"PATCH", "/home", http.StatusOK, "patch home"},
		{"DELETE", "/home", http.StatusOK, "delete home"},
		{"GET", "/users/123", http.StatusOK, "user 123"},
		{"GET", "/any", http.StatusOK, "any method"},
		{"POST", "/any", http.StatusOK, "any method"},
		{"DELETE", "/any", http.StatusOK, "any method"},
		{"GET", "/notfound", http.StatusNotFound, "Not Found"},
		{"OPTIONS", "/home", http.StatusNotFound, "Not Found"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			// Create a mock request
			httpReq := httptest.NewRequest(tc.method, tc.path, nil)

			// Create a buffer and write the request to it
			var buf bytes.Buffer
			err := httpReq.Write(&buf)
			assert.NoError(t, err)

			// Create a request object from the buffer
			req, err := request.RequestFromReader(&buf)
			assert.NoError(t, err)

			// Call the handler
			resp := handler(req)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Write the response to the recorder
			err = resp.Write(w)
			assert.NoError(t, err)

			// Parse the response
			statusCode, body := parseResponse(w)

			// Check the status code
			assert.Equal(t, tc.expectedStatus, statusCode)

			// Check the body
			assert.Equal(t, tc.expectedBody, body)
		})
	}
}
