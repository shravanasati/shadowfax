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
	require.Error(t, err) // no host field
	require.Nil(t, r)

	// Test: Malformed Header
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Duplicate Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: smth\r\nAccept: text/html\r\nAccept: application/json\r\n\r\n",
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

func TestBodyParse(t *testing.T) {
	// Test: Standard Body
	reader := &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 13\r\n" +
			"\r\n" +
			"hello world!\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	bodyReader, err := r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err := io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "hello world!\n", string(bodyBytes))

	// Test: Empty Body
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
		numBytesPerRead: 2,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err = io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "", string(bodyBytes))

	// Test: Body shorter than reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 20\r\n" +
			"\r\n" +
			"partial content",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	_, err = io.ReadAll(bodyReader)
	require.Error(t, err) // Should error when body is shorter than content-length

	// Test: Body longer than reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 5\r\n" +
			"\r\n" +
			"hello world extra data",
		numBytesPerRead: 4,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err = io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(bodyBytes)) // Should only read up to content-length

	// Test: No Content-Length header
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n" +
			"body without content length",
		numBytesPerRead: 5,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err = io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "", string(bodyBytes)) // Should have empty body when no Content-Length

	// Test: Invalid Content-Length header - this now needs to be tested at body read time
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: invalid\r\n" +
			"\r\n" +
			"some body",
		numBytesPerRead: 6,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err) // Request parsing should succeed
	require.NotNil(t, r)
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err = io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "", string(bodyBytes)) // Should have empty body for invalid content-length
}

func TestChunkedTransferEncoding(t *testing.T) {
	// Test: Basic chunked transfer encoding
	reader := &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"7\r\n" +
			"Mozilla\r\n" +
			"9\r\n" +
			"Developer\r\n" +
			"7\r\n" +
			"Network\r\n" +
			"0\r\n" +
			"\r\n",
		numBytesPerRead: 4,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)

	// Verify body content is correctly reconstructed
	bodyReader, err := r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err := io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "MozillaDeveloperNetwork", string(bodyBytes))

	// Verify Transfer-Encoding header is removed
	transferEncoding := r.Headers.Get("transfer-encoding")
	assert.Equal(t, "", transferEncoding)

	// Verify Content-Length header is present and correct
	contentLength := r.Headers.Get("content-length")
	assert.Equal(t, "23", contentLength) // "MozillaDeveloperNetwork" = 23 bytes

	// Test: Chunked transfer encoding with extensions (should be ignored)
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"5;name=value\r\n" +
			"hello\r\n" +
			"6;another=ext\r\n" +
			" world\r\n" +
			"0\r\n" +
			"\r\n",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)

	// Verify body content is correctly reconstructed
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err = io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(bodyBytes))

	// Verify Transfer-Encoding header is removed
	transferEncoding = r.Headers.Get("transfer-encoding")
	assert.Equal(t, "", transferEncoding)

	// Verify Content-Length header is present and correct
	contentLength = r.Headers.Get("content-length")
	assert.Equal(t, "11", contentLength) // "hello world" = 11 bytes

	// Test: Chunked transfer encoding with trailer headers
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"Trailer: Expires, Signature\r\n" +
			"\r\n" +
			"4\r\n" +
			"test\r\n" +
			"0\r\n" +
			"Expires: Wed, 21 Oct 2015 07:28:00 GMT\r\n" +
			"Signature: abc123\r\n" +
			"\r\n",
		numBytesPerRead: 5,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)

	// Verify body content
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err = io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "test", string(bodyBytes))

	// Verify Transfer-Encoding header is removed
	transferEncoding = r.Headers.Get("transfer-encoding")
	assert.Equal(t, "", transferEncoding)

	// Verify Content-Length header is present
	contentLength = r.Headers.Get("content-length")
	assert.Equal(t, "4", contentLength)

	// Verify trailer headers are added to main headers
	expires := r.Headers.Get("expires")
	assert.Equal(t, "Wed, 21 Oct 2015 07:28:00 GMT", expires)
	signature := r.Headers.Get("signature")
	assert.Equal(t, "abc123", signature)

	// Test: Empty chunked body
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"0\r\n" +
			"\r\n",
		numBytesPerRead: 2,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)

	// Verify empty body
	bodyReader, err = r.Body()
	require.NoError(t, err)
	defer bodyReader.Close()
	bodyBytes, err = io.ReadAll(bodyReader)
	require.NoError(t, err)
	assert.Equal(t, "", string(bodyBytes))

	// Verify Transfer-Encoding header is removed
	transferEncoding = r.Headers.Get("transfer-encoding")
	assert.Equal(t, "", transferEncoding)

	// Verify Content-Length header is present and zero
	contentLength = r.Headers.Get("content-length")
	assert.Equal(t, "0", contentLength)

	// Test: Invalid chunk size (non-hex)
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"ZZ\r\n" +
			"hello\r\n" +
			"0\r\n" +
			"\r\n",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err) // Request parsing should succeed
	require.NotNil(t, r)

	// Error should occur when trying to read the body
	bodyReader, err = r.Body()
	require.NoError(t, err)
	_, err = io.ReadAll(bodyReader)
	require.Error(t, err)

	// Test: Missing final chunk
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"5\r\n" +
			"hello\r\n",
		numBytesPerRead: 4,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err) // Request parsing should succeed
	require.NotNil(t, r)

	// Error should occur when trying to read the body
	bodyReader, err = r.Body()
	require.NoError(t, err)
	_, err = io.ReadAll(bodyReader)
	require.Error(t, err)

	// Test: Chunk data shorter than declared size
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"10\r\n" +
			"short\r\n" +
			"0\r\n" +
			"\r\n",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err) // Request parsing should succeed
	require.NotNil(t, r)

	// Error should occur when trying to read the body
	bodyReader, err = r.Body()
	require.NoError(t, err)
	_, err = io.ReadAll(bodyReader)
	require.Error(t, err)
}

func TestUnsupportedTransferEncodings(t *testing.T) {
	// Test: Gzip transfer encoding should return not implemented error when body is read
	reader := &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: gzip\r\n" +
			"\r\n" +
			"some gzipped content here",
		numBytesPerRead: 4,
	}
	r, err := RequestFromReader(reader)
	assert.Equal(t, ErrNotImplemented, err)
	require.Nil(t, r)

	// Test: Deflate transfer encoding should return not implemented error when body is read
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: deflate\r\n" +
			"\r\n" +
			"some deflated content here",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	assert.Equal(t, ErrNotImplemented, err)
	require.Nil(t, r)

	// Test: Compress transfer encoding should return not implemented error when body is read
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: compress\r\n" +
			"\r\n" +
			"some compressed content here",
		numBytesPerRead: 5,
	}
	r, err = RequestFromReader(reader)
	assert.Equal(t, ErrNotImplemented, err)
	require.Nil(t, r)

	// Test: Multiple transfer encodings with unsupported encoding
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: chunked, gzip\r\n" +
			"\r\n" +
			"some content here",
		numBytesPerRead: 4,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)
	assert.Equal(t, ErrNotImplemented, err)

	// Test: Custom/unknown transfer encoding should return not implemented error when body is read
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: custom-encoding\r\n" +
			"\r\n" +
			"some custom encoded content here",
		numBytesPerRead: 6,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)
	assert.Equal(t, ErrNotImplemented, err)

	// Test: Case insensitive unsupported transfer encoding
	reader = &chunkReader{
		data: "POST /upload HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Transfer-Encoding: GZIP\r\n" +
			"\r\n" +
			"some gzipped content here",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)
	assert.Equal(t, ErrNotImplemented, err)
}

func TestInvalidFraming(t *testing.T) {
	// Test: Both Content-Length and Transfer-Encoding: chunked
	reader := &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 10\r\n" +
			"Transfer-Encoding: chunked\r\n" +
			"\r\n" +
			"0\r\n\r\n",
		numBytesPerRead: 5,
	}
	_, err := RequestFromReader(reader)
	require.Error(t, err)
	assert.Equal(t, ErrInvalidFraming, err)

	// Test: Duplicate Content-Length headers with same value (should be allowed or normalized, but RFC says reject if multiple)
	// Actually RFC 9112 Section 6.1 says "If a message is received that has multiple Content-Length header fields... then the message occurs in one of the following cases: (etc)"
	// In our implementation, headers.Add appends with comma.
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 10\r\n" +
			"Content-Length: 10\r\n" +
			"\r\n" +
			"1234567890",
		numBytesPerRead: 5,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err, "Should reject duplicate Content-Length headers")

	// Test: Duplicate Content-Length headers with different values
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 10\r\n" +
			"Content-Length: 20\r\n" +
			"\r\n" +
			"1234567890",
		numBytesPerRead: 5,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err, "Should reject duplicate Content-Length headers with different values")

	// Test: Multiple Host headers
	reader = &chunkReader{
		data: "GET / HTTP/1.1\r\n" +
			"Host: example.com\r\n" +
			"Host: example.org\r\n" +
			"\r\n",
		numBytesPerRead: 5,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err, "Should reject multiple Host headers")

	// Test: Transfer-Encoding last encoding is not chunked
	reader = &chunkReader{
		data: "POST / HTTP/1.1\r\n" +
			"Host: localhost\r\n" +
			"Transfer-Encoding: chunked, gzip\r\n" +
			"\r\n" +
			"body",
		numBytesPerRead: 5,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err, "Should reject if chunked is not the last Transfer-Encoding")
}

func TestInvalidTrailers(t *testing.T) {
	// Test: Restricted trailer headers (e.g., Transfer-Encoding, Content-Length, Host, etc.)
	// RFC 9110 Section 6.5.1: A sender MUST NOT generate a trailer section that contains any of the following fields:
	// - Fields used for message framing (e.g., Transfer-Encoding, Content-Length, etc.)
	// - Fields used for routing/reachability (e.g., Host, etc.)
	// - Authentication/credentials (e.g., Authorization, etc.)
	// - Control fields (e.g., Cache-Control, Max-Forwards, etc.)

	restrictedTrailers := []string{
		"Transfer-Encoding",
		"Authorization",
		"Cache-Control",
	}

	for _, header := range restrictedTrailers {
		t.Run("RestrictedTrailer_"+header, func(t *testing.T) {
			reader := &chunkReader{
				data: "POST /upload HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"Transfer-Encoding: chunked\r\n" +
					"\r\n" +
					"4\r\n" +
					"test\r\n" +
					"0\r\n" +
					header + ": some-value\r\n" +
					"\r\n",
				numBytesPerRead: 5,
			}
			r, err := RequestFromReader(reader)
			require.NoError(t, err)

			body, err := r.Body()
			require.NoError(t, err)
			_, err = io.ReadAll(body)
			require.NoError(t, err)

			assert.Empty(t, r.Headers.Get(header), "Header "+header+" should not be added from trailer")
		})
	}
}
