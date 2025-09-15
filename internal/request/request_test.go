package request

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

// Read reads up to len(p) or numBytesPerRead bytes from the string per call
// its useful for simulating reading a variable number of bytes per chunk from a network connection
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := min(cr.pos+cr.numBytesPerRead, len(cr.data))
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n

	return n, nil
}

func TestRequestLineParse(t *testing.T) {
	// Test: Good GET Request line
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.Target)
	assert.Equal(t, "1.1", r.RequestLine.HTTPVersion)

	// Test: Good GET Request line with path
	reader = &chunkReader{
		data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.Target)
	assert.Equal(t, "1.1", r.RequestLine.HTTPVersion)

	// Test: Invalid number of parts in request line
	reader = &chunkReader{
		data:            "/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 5,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Good POST Request with path
	reader = &chunkReader{
		data:            "POST /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 4,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.Target)
	assert.Equal(t, "1.1", r.RequestLine.HTTPVersion)

	// Test: Invalid method (out of order) Request line
	reader = &chunkReader{
		data:            "HTTP/1.1 GET /\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 2,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Invalid version in Request line
	reader = &chunkReader{
		data:            "GET / HTTP/1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 6,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)
}

func TestHeadersParse(t *testing.T) {
	// Test: Standard Headers
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	host := r.Headers.Get("host")
	assert.Equal(t, "localhost:42069", host)
	userAgent := r.Headers.Get("user-agent")
	assert.Equal(t, "curl/7.81.0", userAgent)
	accept := r.Headers.Get("accept")
	assert.Equal(t, "*/*", accept)

	// Test: Empty Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\n\r\n",
		numBytesPerRead: 2,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	host = r.Headers.Get("host")
	assert.Equal(t, "", host) // Should return empty string for missing header

	// Test: Malformed Header
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Duplicate Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nAccept: text/html\r\nAccept: application/json\r\n\r\n",
		numBytesPerRead: 5,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	accept = r.Headers.Get("accept")
	assert.Equal(t, "text/html, application/json", accept)

	// Test: Case Insensitive Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHOST: localhost:42069\r\nuser-agent: curl/7.81.0\r\n\r\n",
		numBytesPerRead: 4,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	host = r.Headers.Get("host")
	assert.Equal(t, "localhost:42069", host)
	userAgent = r.Headers.Get("USER-AGENT")
	assert.Equal(t, "curl/7.81.0", userAgent)

	// Test: Missing End of Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n",
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)
}

// func TestBodyParse(t *testing.T) {
// 	// Test: Standard Body
// 	reader := &chunkReader{
// 		data: "POST /submit HTTP/1.1\r\n" +
// 			"Host: localhost:42069\r\n" +
// 			"Content-Length: 13\r\n" +
// 			"\r\n" +
// 			"hello world!\n",
// 		numBytesPerRead: 3,
// 	}
// 	r, err := RequestFromReader(reader)
// 	require.NoError(t, err)
// 	require.NotNil(t, r)
// 	assert.Equal(t, "hello world!\n", string(r.Body))

// 	// Test: Empty Body
// 	reader = &chunkReader{
// 		data: "POST /submit HTTP/1.1\r\n" +
// 			"Host: localhost:42069\r\n" +
// 			"Content-Length: 0\r\n" +
// 			"\r\n",
// 		numBytesPerRead: 2,
// 	}
// 	r, err = RequestFromReader(reader)
// 	require.NoError(t, err)
// 	require.NotNil(t, r)
// 	assert.Equal(t, "", string(r.Body))

// 	// Test: Body shorter than reported content length
// 	reader = &chunkReader{
// 		data: "POST /submit HTTP/1.1\r\n" +
// 			"Host: localhost:42069\r\n" +
// 			"Content-Length: 20\r\n" +
// 			"\r\n" +
// 			"partial content",
// 		numBytesPerRead: 3,
// 	}
// 	r, err = RequestFromReader(reader)
// 	require.Error(t, err)

// 	// Test: Body longer than reported content length
// 	reader = &chunkReader{
// 		data: "POST /submit HTTP/1.1\r\n" +
// 			"Host: localhost:42069\r\n" +
// 			"Content-Length: 5\r\n" +
// 			"\r\n" +
// 			"hello world extra data",
// 		numBytesPerRead: 4,
// 	}
// 	r, err = RequestFromReader(reader)
// 	require.Error(t, err)

// 	// Test: No Content-Length header
// 	reader = &chunkReader{
// 		data: "POST /submit HTTP/1.1\r\n" +
// 			"Host: localhost:42069\r\n" +
// 			"\r\n" +
// 			"body without content length",
// 		numBytesPerRead: 5,
// 	}
// 	r, err = RequestFromReader(reader)
// 	require.NoError(t, err)
// 	require.NotNil(t, r)
// 	assert.Equal(t, "", string(r.Body)) // Should have empty body when no Content-Length

// 	// Test: Invalid Content-Length header
// 	reader = &chunkReader{
// 		data: "POST /submit HTTP/1.1\r\n" +
// 			"Host: localhost:42069\r\n" +
// 			"Content-Length: invalid\r\n" +
// 			"\r\n" +
// 			"some body",
// 		numBytesPerRead: 6,
// 	}
// 	r, err = RequestFromReader(reader)
// 	require.Error(t, err)
// }
