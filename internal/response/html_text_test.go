package response

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTMLResponse(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		expectedBody string
	}{
		{
			name:         "simple HTML",
			body:         "<h1>Hello World</h1>",
			expectedBody: "<h1>Hello World</h1>",
		},
		{
			name:         "complex HTML document",
			body:         "<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Welcome</h1><p>This is a test page.</p></body></html>",
			expectedBody: "<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Welcome</h1><p>This is a test page.</p></body></html>",
		},
		{
			name:         "empty HTML",
			body:         "",
			expectedBody: "",
		},
		{
			name:         "HTML with special characters",
			body:         "<p>Special chars: &lt; &gt; &amp; &quot; &#39;</p>",
			expectedBody: "<p>Special chars: &lt; &gt; &amp; &quot; &#39;</p>",
		},
		{
			name:         "HTML with line breaks",
			body:         "<div>\n  <p>Line 1</p>\n  <p>Line 2</p>\n</div>",
			expectedBody: "<div>\n  <p>Line 1</p>\n  <p>Line 2</p>\n</div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewHTMLResponse(tt.body)
			require.NotNil(t, resp)

			// Check headers
			headers := resp.GetHeaders()
			assert.Equal(t, "text/html", headers.Get("content-type"))
			assert.Equal(t, strconv.Itoa(len(tt.expectedBody)), headers.Get("content-length"))

			// Check body
			body := resp.GetBody()
			require.NotNil(t, body)
			
			bodyBytes, err := io.ReadAll(body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody, string(bodyBytes))

			// Check status code
			assert.Equal(t, StatusCode(200), resp.GetStatusCode())
		})
	}
}

func TestHTMLResponseWrite(t *testing.T) {
	htmlContent := "<html><body><h1>Test Page</h1></body></html>"
	resp := NewHTMLResponse(htmlContent)

	var buf strings.Builder
	err := resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()
	
	// Check that it contains HTTP response parts
	assert.Contains(t, output, "HTTP/1.1 200 OK")
	assert.Contains(t, output, "content-type: text/html")
	assert.Contains(t, output, htmlContent)
}

func TestHTMLResponseMethods(t *testing.T) {
	htmlContent := "<p>Test content</p>"
	resp := NewHTMLResponse(htmlContent)

	// Test WithStatusCode
	modifiedResp := resp.WithStatusCode(404)
	assert.Equal(t, StatusCode(404), modifiedResp.GetStatusCode())

	// Test WithHeader
	modifiedResp = resp.WithHeader("Cache-Control", "no-cache")
	assert.Equal(t, "no-cache", modifiedResp.GetHeaders().Get("Cache-Control"))

	// Test WithHeaders
	headers := map[string]string{
		"X-Frame-Options": "DENY",
		"X-XSS-Protection": "1; mode=block",
	}
	modifiedResp = resp.WithHeaders(headers)
	assert.Equal(t, "DENY", modifiedResp.GetHeaders().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", modifiedResp.GetHeaders().Get("X-XSS-Protection"))
}

func TestNewTextResponse(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		expectedBody string
	}{
		{
			name:         "simple text",
			body:         "Hello World",
			expectedBody: "Hello World",
		},
		{
			name:         "multiline text",
			body:         "Line 1\nLine 2\nLine 3",
			expectedBody: "Line 1\nLine 2\nLine 3",
		},
		{
			name:         "empty text",
			body:         "",
			expectedBody: "",
		},
		{
			name:         "text with special characters",
			body:         "Special chars: @#$%^&*(){}[]|\\:;\"'<>?,./",
			expectedBody: "Special chars: @#$%^&*(){}[]|\\:;\"'<>?,./",
		},
		{
			name:         "Unicode text",
			body:         "Unicode: 你好世界 🌍 emoji test",
			expectedBody: "Unicode: 你好世界 🌍 emoji test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewTextResponse(tt.body)
			require.NotNil(t, resp)

			// Check headers
			headers := resp.GetHeaders()
			assert.Equal(t, "text/plain", headers.Get("content-type"))
			assert.Equal(t, strconv.Itoa(len(tt.expectedBody)), headers.Get("content-length"))

			// Check body
			body := resp.GetBody()
			require.NotNil(t, body)
			
			bodyBytes, err := io.ReadAll(body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody, string(bodyBytes))

			// Check status code
			assert.Equal(t, StatusCode(200), resp.GetStatusCode())
		})
	}
}

func TestTextResponseWrite(t *testing.T) {
	textContent := "This is a plain text response"
	resp := NewTextResponse(textContent)

	var buf strings.Builder
	err := resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()
	
	// Check that it contains HTTP response parts
	assert.Contains(t, output, "HTTP/1.1 200 OK")
	assert.Contains(t, output, "content-type: text/plain")
	assert.Contains(t, output, textContent)
}

func TestTextResponseMethods(t *testing.T) {
	textContent := "Error message"
	resp := NewTextResponse(textContent)

	// Test WithStatusCode
	modifiedResp := resp.WithStatusCode(500)
	assert.Equal(t, StatusCode(500), modifiedResp.GetStatusCode())

	// Test WithHeader
	modifiedResp = resp.WithHeader("X-Error-Code", "INTERNAL_ERROR")
	assert.Equal(t, "INTERNAL_ERROR", modifiedResp.GetHeaders().Get("X-Error-Code"))

	// Test WithHeaders
	headers := map[string]string{
		"Retry-After": "300",
		"X-Request-ID": "12345",
	}
	modifiedResp = resp.WithHeaders(headers)
	assert.Equal(t, "300", modifiedResp.GetHeaders().Get("Retry-After"))
	assert.Equal(t, "12345", modifiedResp.GetHeaders().Get("X-Request-ID"))
}