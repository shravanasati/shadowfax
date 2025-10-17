// Test cases copied from https://cs.opensource.google/go/go/+/refs/tags/go1.25.3:src/net/http/csrf_test.go

package middleware

import (
	"io"
	"strings"
	"testing"

	"github.com/shravanasati/shadowfax/headers"
	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/server"
)

var okHandler server.Handler = func(_ *request.Request) response.Response {
	return response.NewBaseResponse().WithStatusCode(response.StatusOK)
}

func newReq(method, target string) *request.Request {
	return &request.Request{
		RequestLine: request.RequestLine{Method: method, Target: target, HTTPVersion: "1.1"},
		Headers:     *headers.NewHeaders(),
	}
}

func TestCORFSecFetchSite(t *testing.T) {
	middleware, err := NewCORF()
	if err != nil {
		t.Fatalf("NewCORFMiddleware: %v", err)
	}
	handler := middleware.Handler(okHandler)

	tests := []struct {
		name           string
		method         string
		secFetchSite   string
		origin         string
		expectedStatus response.StatusCode
	}{
		{"same-origin allowed", "POST", "same-origin", "", response.StatusOK},
		{"none allowed", "POST", "none", "", response.StatusOK},
		{"cross-site blocked", "POST", "cross-site", "", response.StatusForbidden},
		{"same-site blocked", "POST", "same-site", "", response.StatusForbidden},

		// No Sec-Fetch-Site header cases
		{"no header with no origin", "POST", "", "", response.StatusOK},
		{"no header with matching origin", "POST", "", "https://example.com", response.StatusOK},
		{"no header with mismatched origin", "POST", "", "https://attacker.example", response.StatusForbidden},
		{"no header with null origin", "POST", "", "null", response.StatusForbidden},

		// Safe methods allowed even when cross-site
		{"GET allowed", "GET", "cross-site", "", response.StatusOK},
		{"HEAD allowed", "HEAD", "cross-site", "", response.StatusOK},
		{"OPTIONS allowed", "OPTIONS", "cross-site", "", response.StatusOK},
		{"PUT blocked", "PUT", "cross-site", "", response.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := newReq(tc.method, "/")
			req.Headers.Add("Host", "example.com")
			if tc.secFetchSite != "" {
				req.Headers.Add("Sec-Fetch-Site", tc.secFetchSite)
			}
			if tc.origin != "" {
				req.Headers.Add("Origin", tc.origin)
			}

			resp := handler(req)
			if resp.GetStatusCode() != tc.expectedStatus {
				t.Errorf("got status %d, want %d", resp.GetStatusCode(), tc.expectedStatus)
			}
		})
	}
}

func TestCORFSetDenyHandler(t *testing.T) {
	// Ensure default behavior is 403 for cross-site POST
	middleware, err := NewCORF()
	if err != nil {
		t.Fatalf("NewCORFMiddleware: %v", err)
	}
	handler := middleware.Handler(okHandler)

	req := newReq("POST", "/")
	req.Headers.Add("Sec-Fetch-Site", "cross-site")

	resp := handler(req)
	if resp.GetStatusCode() != response.StatusForbidden {
		t.Fatalf("got status %d, want %d", resp.GetStatusCode(), response.StatusForbidden)
	}

	// Set a custom deny handler
	custom := func(_ *request.Request) response.Response {
		return response.NewTextResponse("custom error").WithStatusCode(response.StatusImATeapot)
	}
	middleware.SetDenyHandler(custom)
	t.Cleanup(func() { middleware.SetDenyHandler(nil) })

	resp = handler(req)
	if resp.GetStatusCode() != response.StatusImATeapot {
		t.Fatalf("got status %d, want %d", resp.GetStatusCode(), response.StatusImATeapot)
	}
	// Read body for message verification
	body := resp.GetBody()
	if body == nil {
		t.Fatalf("expected body from custom handler")
	}
	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(b), "custom error") {
		t.Fatalf("expected custom error message, got %q", string(b))
	}

	// Reset to default and check again
	middleware.SetDenyHandler(nil)
	resp = handler(req)
	if resp.GetStatusCode() != response.StatusForbidden {
		t.Fatalf("after reset got status %d, want %d", resp.GetStatusCode(), response.StatusForbidden)
	}
}

func TestCORFTrustedOriginBypass(t *testing.T) {
	middleware, err := NewCORF("https://trusted.example")
	if err != nil {
		t.Fatalf("NewCORFMiddleware: %v", err)
	}
	handler := middleware.Handler(okHandler)

	tests := []struct {
		name           string
		origin         string
		secFetchSite   string
		expectedStatus response.StatusCode
	}{
		{"trusted origin without sec-fetch-site", "https://trusted.example", "", response.StatusOK},
		{"trusted origin with cross-site", "https://trusted.example", "cross-site", response.StatusOK},
		{"untrusted origin without sec-fetch-site", "https://attacker.example", "", response.StatusForbidden},
		{"untrusted origin with cross-site", "https://attacker.example", "cross-site", response.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := newReq("POST", "/")
			req.Headers.Add("Host", "example.com")
			req.Headers.Add("Origin", tc.origin)
			if tc.secFetchSite != "" {
				req.Headers.Add("Sec-Fetch-Site", tc.secFetchSite)
			}

			resp := handler(req)
			if resp.GetStatusCode() != tc.expectedStatus {
				t.Errorf("got status %d, want %d", resp.GetStatusCode(), tc.expectedStatus)
			}
		})
	}
}

func TestCORFTrustedOriginValidation(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		wantErr bool
	}{
		{"valid origin", "https://example.com", false},
		{"valid origin with port", "https://example.com:8080", false},
		{"http origin", "http://example.com", false},
		{"missing scheme", "example.com", true},
		{"missing host", "https://", true},
		{"trailing slash", "https://example.com/", true},
		{"with path", "https://example.com/path", true},
		{"with query", "https://example.com?query=value", true},
		{"with fragment", "https://example.com#fragment", true},
		{"invalid url", "https://ex ample.com", true},
		{"empty string", "", true},
		{"null", "null", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCORF(tc.origin)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewCORFMiddleware(%q) error = %v, wantErr %v", tc.origin, err, tc.wantErr)
			}
		})
	}
}

func TestCrossOriginProtectionAddTrustedOriginErrors(t *testing.T) {
	protection, err := NewCORF()
	if err != nil {
		t.Fatalf("NewCORFMiddleware: %v", err)
	}

	tests := []struct {
		name    string
		origin  string
		wantErr bool
	}{
		{"valid origin", "https://example.com", false},
		{"valid origin with port", "https://example.com:8080", false},
		{"http origin", "http://example.com", false},
		{"missing scheme", "example.com", true},
		{"missing host", "https://", true},
		{"trailing slash", "https://example.com/", true},
		{"with path", "https://example.com/path", true},
		{"with query", "https://example.com?query=value", true},
		{"with fragment", "https://example.com#fragment", true},
		{"invalid url", "https://ex ample.com", true},
		{"empty string", "", true},
		{"null", "null", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := protection.AddTrustedOrigin(tc.origin)
			if (err != nil) != tc.wantErr {
				t.Errorf("AddTrustedOrigin(%q) error = %v, wantErr %v", tc.origin, err, tc.wantErr)
			}
		})
	}
}

func TestCrossOriginProtectionAddingBypassesConcurrently(t *testing.T) {
	protection, err := NewCORF()
	if err != nil {
		t.Fatalf("NewCORFMiddleware: %v", err)
	}
	handler := protection.Handler(okHandler)

	req := newReq("POST", "https://example.com/")
	req.Headers.Set("Origin", "https://concurrent.example")
	req.Headers.Set("Sec-Fetch-Site", "cross-site")

	resp := handler(req)
	if resp.GetStatusCode() != response.StatusForbidden {
		t.Errorf("got status %d, want %d", resp.GetStatusCode(), response.StatusForbidden)
	}

	start := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(start)
		defer close(done)
		for range 10 {
			handler(req)
		}
	}()

	// Add bypasses while the requests are in flight.
	<-start
	protection.AddTrustedOrigin("https://concurrent.example")
	<-done

	resp = handler(req)

	if resp.GetStatusCode() != response.StatusOK {
		t.Errorf("After concurrent bypass addition, got status %d, want %d", resp.GetStatusCode(), response.StatusOK)
	}
}
