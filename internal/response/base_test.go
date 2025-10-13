package response

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/shravanasati/shadowfax/internal/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseResponse(t *testing.T) {
	resp := NewBaseResponse()
	require.NotNil(t, resp)

	// Check default status code
	assert.Equal(t, StatusCode(200), resp.GetStatusCode())

	// Check that headers are initialized
	assert.NotNil(t, resp.GetHeaders())

	// Check that body is nil initially
	assert.Nil(t, resp.GetBody())
}

func TestBaseResponseGetters(t *testing.T) {
	resp := NewBaseResponse()

	// Test GetStatusCode
	assert.Equal(t, StatusCode(200), resp.GetStatusCode())

	// Test GetHeaders
	headers := resp.GetHeaders()
	require.NotNil(t, headers)
	assert.Equal(t, 0, headers.Size()) // No headers initially

	// Test GetBody
	assert.Nil(t, resp.GetBody())
}

func TestBaseResponseWithMethods(t *testing.T) {
	resp := NewBaseResponse()

	// Test WithStatusCode
	modifiedResp := resp.WithStatusCode(404)
	assert.Equal(t, StatusCode(404), modifiedResp.GetStatusCode())
	// Original response should be modified (not immutable)
	assert.Equal(t, StatusCode(404), resp.GetStatusCode())

	// Test WithHeader
	modifiedResp = resp.WithHeader("Content-Type", "application/json")
	assert.Equal(t, "application/json", modifiedResp.GetHeaders().Get("Content-Type"))

	// Test WithHeaders
	headersMap := map[string]string{
		"X-Custom-Header": "custom-value",
		"Cache-Control":   "no-cache",
	}
	modifiedResp = resp.WithHeaders(headersMap)
	assert.Equal(t, "custom-value", modifiedResp.GetHeaders().Get("X-Custom-Header"))
	assert.Equal(t, "no-cache", modifiedResp.GetHeaders().Get("Cache-Control"))

	// Test WithBody
	bodyContent := "Test body content"
	bodyReader := strings.NewReader(bodyContent)
	modifiedResp = resp.WithBody(bodyReader)
	assert.Equal(t, bodyReader, modifiedResp.GetBody())
}

func TestBaseResponseChaining(t *testing.T) {
	// Test method chaining
	bodyContent := "Chained response body"
	resp := NewBaseResponse().
		WithStatusCode(201).
		WithHeader("Content-Type", "text/plain").
		WithHeader("X-Test", "chaining").
		WithBody(strings.NewReader(bodyContent))

	assert.Equal(t, StatusCode(201), resp.GetStatusCode())
	assert.Equal(t, "text/plain", resp.GetHeaders().Get("Content-Type"))
	assert.Equal(t, "chaining", resp.GetHeaders().Get("X-Test"))

	body := resp.GetBody()
	require.NotNil(t, body)
	bodyBytes, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, bodyContent, string(bodyBytes))
}

func TestBaseResponseWrite(t *testing.T) {
	bodyContent := "Response body for write test"
	resp := NewBaseResponse().
		WithStatusCode(201).
		WithHeader("Content-Type", "text/plain").
		WithHeader("Content-Length", "28").
		WithBody(strings.NewReader(bodyContent))

	var buf strings.Builder
	err := resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Check status line
	assert.Contains(t, output, "HTTP/1.1 201 Created")

	// Check headers
	assert.Contains(t, output, "content-type: text/plain")
	assert.Contains(t, output, "content-length: 28")

	// Check body
	assert.Contains(t, output, bodyContent)
}

func TestBaseResponseWriteWithoutBody(t *testing.T) {
	resp := NewBaseResponse().
		WithStatusCode(204). // No Content
		WithHeader("X-Test", "no-body")

	var buf strings.Builder
	err := resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Check status line
	assert.Contains(t, output, "HTTP/1.1 204 No Content")

	// Check headers
	assert.Contains(t, output, "x-test: no-body")

	// Should end with just headers (no body)
	assert.True(t, strings.HasSuffix(output, "\r\n\r\n"))
}

func TestResponseWriter(t *testing.T) {
	var buf strings.Builder
	rw := NewResponseWriter(&buf)
	require.NotNil(t, rw)

	// Test WriteStatusLine
	err := rw.WriteStatusLine(200)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "HTTP/1.1 200 OK\r\n")

	// Test WriteStatusLine again (should fail)
	err = rw.WriteStatusLine(404)
	assert.Error(t, err)
	assert.Equal(t, errors.Unwrap(err), ErrInvalidWriterState)
}

func TestResponseWriterHeaders(t *testing.T) {
	var buf strings.Builder
	rw := NewResponseWriter(&buf)

	// Write status line first
	err := rw.WriteStatusLine(200)
	require.NoError(t, err)

	// Create and write headers
	h := headers.NewHeaders()
	h.Add("Content-Type", "text/html")
	h.Add("Content-Length", "13")

	err = rw.WriteHeaders(h)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "content-type: text/html\r\n")
	assert.Contains(t, output, "content-length: 13\r\n")
	assert.Contains(t, output, "\r\n\r\n") // Header terminator

	// Test WriteHeaders again (should fail)
	err = rw.WriteHeaders(h)
	assert.Error(t, err)
	assert.Equal(t, errors.Unwrap(err), ErrInvalidWriterState)
}

func TestResponseWriterBody(t *testing.T) {
	var buf strings.Builder
	rw := NewResponseWriter(&buf)

	// Write status line and headers first
	err := rw.WriteStatusLine(200)
	require.NoError(t, err)

	h := headers.NewHeaders()
	err = rw.WriteHeaders(h)
	require.NoError(t, err)

	// Write body
	bodyContent := "Hello, World!"
	bodyReader := strings.NewReader(bodyContent)

	err = rw.WriteBody(bodyReader)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, bodyContent)
}

func TestResponseWriterStateMachine(t *testing.T) {
	var buf strings.Builder
	rw := NewResponseWriter(&buf)

	// Test writing headers before status line (should fail)
	h := headers.NewHeaders()
	err := rw.WriteHeaders(h)
	assert.Error(t, err)
	assert.Equal(t, errors.Unwrap(err), ErrInvalidWriterState)

	// Reset with new writer
	buf.Reset()
	rw = NewResponseWriter(&buf)

	// Test writing body before headers (should fail)
	bodyReader := strings.NewReader("test")
	err = rw.WriteBody(bodyReader)
	assert.Error(t, err)
	assert.Equal(t, errors.Unwrap(err), ErrInvalidWriterState)
}

func TestResponseWriterNilWriter(t *testing.T) {
	rw := NewResponseWriter(nil)

	// All operations should fail with nil writer
	err := rw.WriteStatusLine(200)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writer is nil")

	h := headers.NewHeaders()
	err = rw.WriteHeaders(h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writer is nil")

	bodyReader := strings.NewReader("test")
	err = rw.WriteBody(bodyReader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writer is nil")
}
