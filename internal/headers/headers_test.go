package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParsing(t *testing.T) {

	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069")
	err := headers.ParseLine(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hval := headers.Get("Host")
	assert.Equal(t, "localhost:42069", hval)
	// Test: Missing Headers
	hval2 := headers.Get("Missing")
	assert.Equal(t, hval2, "")

	// Test: Valid single header with extra whitespace
	headers = NewHeaders()
	data = []byte("Host:   localhost:42069   ")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hval = headers.Get("Host")
	assert.Equal(t, "localhost:42069", hval)

	// Test: Valid 2 headers with existing headers
	headers = NewHeaders()
	headers.Add("User-Agent", "curl/7.81.0")
	data = []byte("Host: localhost:42069")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	data = []byte("Accept: */*")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hval = headers.Get("Host")
	assert.Equal(t, "localhost:42069", hval)
	hval = headers.Get("Accept")
	assert.Equal(t, "*/*", hval)
	hval = headers.Get("User-Agent")
	assert.Equal(t, "curl/7.81.0", hval)

	headers = NewHeaders()
	data = []byte("")
	err = headers.ParseLine(data)
	require.Error(t, err)

	// Test: Invalid spacing header
	// https://datatracker.ietf.org/doc/html/rfc9112#section-5
	headers = NewHeaders()
	data = []byte("       Host : localhost:42069       ")
	err = headers.ParseLine(data)
	require.Error(t, err)

	// Test: Invalid character in header key
	headers = NewHeaders()
	data = []byte("H©st: localhost:42069")
	err = headers.ParseLine(data)
	require.Error(t, err)

	// Test: Multiple values of the same header
	headers = NewHeaders()
	data = []byte("Accept: text/html")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	data = []byte("Accept: application/json")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	hval = headers.Get("Accept")
	assert.Equal(t, "text/html, application/json", hval)

	// Test: Multiline header value (folded header)
	headers = NewHeaders()
	// Simulate a header value split across two lines (second line starts with a space)
	err = headers.ParseLine([]byte("X-Long-Header: part1"))
	require.NoError(t, err)
	err = headers.ParseLine([]byte(" part2"))
	require.Error(t, err)
}
