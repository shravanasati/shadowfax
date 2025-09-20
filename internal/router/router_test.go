package router

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseResponse(w *httptest.ResponseRecorder) (*http.Response, string, error) {
    res, err := http.ReadResponse(bufio.NewReader(w.Body), nil)
    if err != nil {
        return nil, "", err
    }
    defer res.Body.Close()
    b, _ := io.ReadAll(res.Body)
    return res, string(b), nil
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
			httpReq := httptest.NewRequest(tc.method, tc.path, nil)

			var buf bytes.Buffer
			err := httpReq.Write(&buf)
			assert.NoError(t, err)

			req, err := request.RequestFromReader(&buf)
			assert.NoError(t, err)

			resp := handler(req)

			w := httptest.NewRecorder()

			err = resp.Write(w)
			assert.NoError(t, err)

			res, body, err := parseResponse(w)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, res.StatusCode)
			assert.Equal(t, tc.expectedBody, body)
		})
	}
}
