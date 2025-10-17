package middleware

import (
	"encoding/base64"
	"testing"

	"github.com/shravanasati/shadowfax/headers"
	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
)

func newReqNoBody(method, target string) *request.Request {
	return &request.Request{
		RequestLine: request.RequestLine{Method: method, Target: target, HTTPVersion: "1.1"},
		Headers:     *headers.NewHeaders(),
	}
}

func TestBasicAuth_NoHeader(t *testing.T) {
	mw := BasicAuthMiddleware([]Account{{Username: "user", Password: "pass"}})
	handler := mw(func(_ *request.Request) response.Response { return response.NewBaseResponse() })

	req := newReqNoBody("GET", "/")
	resp := handler(req)

	if resp.GetStatusCode() != response.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.GetStatusCode())
	}
	if got := resp.GetHeaders().Get("www-authenticate"); got == "" {
		t.Fatalf("expected www-authenticate header to be set")
	}
}

func TestBasicAuth_Malformed_NoColon(t *testing.T) {
	mw := BasicAuthMiddleware([]Account{{Username: "user", Password: "pass"}})
	handler := mw(func(_ *request.Request) response.Response { return response.NewBaseResponse() })

	req := newReqNoBody("GET", "/")
	payload := base64.StdEncoding.EncodeToString([]byte("useronly"))
	req.Headers.Add("Authorization", "Basic "+payload)

	resp := handler(req)

	if resp.GetStatusCode() != response.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.GetStatusCode())
	}
}

func TestBasicAuth_InvalidBase64(t *testing.T) {
	mw := BasicAuthMiddleware([]Account{{Username: "user", Password: "pass"}})
	handler := mw(func(_ *request.Request) response.Response { return response.NewBaseResponse() })

	req := newReqNoBody("GET", "/")
	req.Headers.Add("Authorization", "Basic not-base64!!")

	resp := handler(req)

	if resp.GetStatusCode() != response.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.GetStatusCode())
	}
}

func TestBasicAuth_WrongCredentials(t *testing.T) {
	mw := BasicAuthMiddleware([]Account{{Username: "user", Password: "pass"}})
	handler := mw(func(_ *request.Request) response.Response { return response.NewBaseResponse() })

	req := newReqNoBody("GET", "/")
	payload := base64.StdEncoding.EncodeToString([]byte("user:wrong"))
	req.Headers.Add("Authorization", "Basic "+payload)

	resp := handler(req)

	if resp.GetStatusCode() != response.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.GetStatusCode())
	}
	if got := resp.GetHeaders().Get("www-authenticate"); got == "" {
		t.Fatalf("expected www-authenticate header to be set")
	}
}

func TestBasicAuth_Success(t *testing.T) {
	mw := BasicAuthMiddleware([]Account{{Username: "user", Password: "pass"}})
	called := false
	handler := mw(func(_ *request.Request) response.Response { called = true; return response.NewBaseResponse() })

	req := newReqNoBody("GET", "/")
	payload := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	req.Headers.Add("Authorization", "Basic "+payload)

	resp := handler(req)

	if resp.GetStatusCode() != response.StatusOK {
		t.Fatalf("expected 200, got %d", resp.GetStatusCode())
	}
	if !called {
		t.Fatalf("expected next handler to be called")
	}
}
