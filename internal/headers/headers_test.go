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
	hval, herr := headers.Get("Host")
	require.NoError(t, herr)
	assert.Equal(t, "localhost:42069", hval)

	// Test: Valid single header with extra whitespace
	headers = NewHeaders()
	data = []byte("Host:   localhost:42069   ")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hval, herr = headers.Get("Host")
	require.NoError(t, herr)
	assert.Equal(t, "localhost:42069", hval)

	// Test: Valid 2 headers with existing headers
	headers = NewHeaders()
	headers["user-agent"] = "curl/7.81.0"
	data = []byte("Host: localhost:42069")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	data = []byte("Accept: */*")
	err = headers.ParseLine(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hval, herr = headers.Get("Host")
	require.NoError(t, herr)
	assert.Equal(t, "localhost:42069", hval)
	hval, herr = headers.Get("Accept")
	require.NoError(t, herr)
	assert.Equal(t, "*/*", hval)
	hval, herr = headers.Get("User-Agent")
	require.NoError(t, herr)
	assert.Equal(t, "curl/7.81.0", hval)

	headers = NewHeaders()
	data = []byte("")
	err = headers.ParseLine(data)
	require.Error(t, err)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("       Host : localhost:42069       ")
	err = headers.ParseLine(data)
	require.Error(t, err)
}
