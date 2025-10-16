package response

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedirectResponse(t *testing.T) {
	tests := []struct {
		name     string
		location string
	}{
		{
			name:     "simple URL",
			location: "https://example.com",
		},
		{
			name:     "relative path",
			location: "/dashboard",
		},
		{
			name:     "path with query parameters",
			location: "/search?q=test&page=1",
		},
		{
			name:     "external URL with path",
			location: "https://api.example.com/v1/users",
		},
		{
			name:     "URL with fragment",
			location: "https://example.com/page#section",
		},
		{
			name:     "empty location",
			location: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewRedirectResponse(tt.location)
			require.NotNil(t, resp)

			// Check that it returns a RedirectResponse type
			redirectResp, ok := resp.(*RedirectResponse)
			require.True(t, ok, "Expected RedirectResponse type")
			require.NotNil(t, redirectResp)

			// Check default status code (302 Found)
			assert.Equal(t, StatusFound, resp.GetStatusCode())

			// Check headers
			headers := resp.GetHeaders()
			require.NotNil(t, headers)

			// Check location header
			assert.Equal(t, tt.location, headers.Get("location"))

			// Check content-length header is set to 0
			assert.Equal(t, "0", headers.Get("content-length"))

			// Check that body is nil for redirect responses
			assert.Nil(t, resp.GetBody())
		})
	}
}

func TestRedirectResponseDefaults(t *testing.T) {
	location := "https://example.com/redirect"
	resp := NewRedirectResponse(location)

	// Verify it's properly structured as a redirect
	assert.Equal(t, StatusFound, resp.GetStatusCode()) // 302
	assert.Equal(t, location, resp.GetHeaders().Get("location"))
	assert.Equal(t, "0", resp.GetHeaders().Get("content-length"))
	assert.Nil(t, resp.GetBody())
}

func TestRedirectResponseModification(t *testing.T) {
	location := "https://example.com/original"
	resp := NewRedirectResponse(location)

	// Test changing status code to permanent redirect
	modifiedResp := resp.WithStatusCode(StatusMovedPermanently) // 301
	assert.Equal(t, StatusMovedPermanently, modifiedResp.GetStatusCode())

	// Test adding additional headers
	modifiedResp = resp.WithHeader("Cache-Control", "no-cache")
	assert.Equal(t, "no-cache", modifiedResp.GetHeaders().Get("Cache-Control"))

	// Location should still be preserved
	assert.Equal(t, location, modifiedResp.GetHeaders().Get("location"))

	// Test adding multiple headers
	headers := map[string]string{
		"X-Redirect-Reason": "maintenance",
		"Retry-After":       "3600",
	}
	modifiedResp = resp.WithHeaders(headers)
	assert.Equal(t, "maintenance", modifiedResp.GetHeaders().Get("X-Redirect-Reason"))
	assert.Equal(t, "3600", modifiedResp.GetHeaders().Get("Retry-After"))
}

func TestRedirectResponseChaining(t *testing.T) {
	location := "/new-location"

	// Test method chaining
	resp := NewRedirectResponse(location).
		WithStatusCode(StatusTemporaryRedirect). // 307
		WithHeader("Cache-Control", "no-store").
		WithHeader("X-Custom-Header", "redirect-test")

	assert.Equal(t, StatusTemporaryRedirect, resp.GetStatusCode())
	assert.Equal(t, location, resp.GetHeaders().Get("location"))
	assert.Equal(t, "no-store", resp.GetHeaders().Get("Cache-Control"))
	assert.Equal(t, "redirect-test", resp.GetHeaders().Get("X-Custom-Header"))
	assert.Equal(t, "0", resp.GetHeaders().Get("content-length"))
}

func TestRedirectResponseWrite(t *testing.T) {
	tests := []struct {
		name       string
		location   string
		statusCode StatusCode
	}{
		{
			name:       "302 Found redirect",
			location:   "https://example.com/found",
			statusCode: StatusFound,
		},
		{
			name:       "301 Moved Permanently",
			location:   "/permanent-redirect",
			statusCode: StatusMovedPermanently,
		},
		{
			name:       "303 See Other",
			location:   "/see-other",
			statusCode: StatusSeeOther,
		},
		{
			name:       "307 Temporary Redirect",
			location:   "https://api.example.com/v2/endpoint",
			statusCode: StatusTemporaryRedirect,
		},
		{
			name:       "308 Permanent Redirect",
			location:   "/v2/permanent",
			statusCode: StatusPermanentRedirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewRedirectResponse(tt.location)
			if tt.statusCode != StatusFound {
				resp = resp.WithStatusCode(tt.statusCode)
			}

			var buf strings.Builder
			err := resp.Write(&buf)
			require.NoError(t, err)

			output := buf.String()

			// Check status line
			expectedStatusLine := "HTTP/1.1 " + fmt.Sprintf("%d", tt.statusCode) + " " + GetStatusReason(tt.statusCode)
			assert.Contains(t, output, expectedStatusLine)

			// Check headers
			assert.Contains(t, output, "location: "+tt.location)
			assert.Contains(t, output, "content-length: 0")

			// Check that response ends properly (no body)
			assert.True(t, strings.HasSuffix(output, "\r\n\r\n"))

			// Verify no body content after headers
			lines := strings.Split(output, "\r\n")
			var foundEmptyLine bool
			for i, line := range lines {
				if line == "" && !foundEmptyLine {
					foundEmptyLine = true
					// After empty line, there should be no more content (or just one more empty line)
					assert.True(t, i >= len(lines)-2, "Should not have body content after header terminator")
				}
			}
		})
	}
}

func TestRedirectResponseWithCustomHeaders(t *testing.T) {
	location := "https://example.com/secure-redirect"
	resp := NewRedirectResponse(location).
		WithHeader("Strict-Transport-Security", "max-age=31536000").
		WithHeader("X-Frame-Options", "DENY").
		WithHeader("Cache-Control", "no-cache, no-store, must-revalidate")

	var buf strings.Builder
	err := resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Check all headers are present
	assert.Contains(t, output, "location: "+location)
	assert.Contains(t, output, "content-length: 0")
	assert.Contains(t, output, "strict-transport-security: max-age=31536000")
	assert.Contains(t, output, "x-frame-options: DENY")
	assert.Contains(t, output, "cache-control: no-cache, no-store, must-revalidate")
}

func TestRedirectResponseInterface(t *testing.T) {
	location := "/test-redirect"
	resp := NewRedirectResponse(location)

	// Ensure it implements the Response interface
	var r Response = resp
	require.NotNil(t, r)

	// Test interface methods
	assert.Equal(t, StatusFound, r.GetStatusCode())
	assert.NotNil(t, r.GetHeaders())
	assert.Nil(t, r.GetBody())

	// Test fluent interface methods
	modified := r.WithStatusCode(StatusMovedPermanently)
	assert.Equal(t, StatusMovedPermanently, modified.GetStatusCode())

	modified = r.WithHeader("Test", "Value")
	assert.Equal(t, "Value", modified.GetHeaders().Get("Test"))
}

func TestRedirectResponseEdgeCases(t *testing.T) {
	t.Run("very long URL", func(t *testing.T) {
		longURL := "https://example.com/" + strings.Repeat("very-long-path-segment/", 100) + "final"
		resp := NewRedirectResponse(longURL)

		assert.Equal(t, longURL, resp.GetHeaders().Get("location"))

		var buf strings.Builder
		err := resp.Write(&buf)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), longURL)
	})

	t.Run("URL with special characters", func(t *testing.T) {
		specialURL := "https://example.com/path?query=hello%20world&other=%3Cscript%3E"
		resp := NewRedirectResponse(specialURL)

		assert.Equal(t, specialURL, resp.GetHeaders().Get("location"))
	})

	t.Run("unicode in URL", func(t *testing.T) {
		unicodeURL := "https://example.com/测试/路径"
		resp := NewRedirectResponse(unicodeURL)

		assert.Equal(t, unicodeURL, resp.GetHeaders().Get("location"))
	})
}
