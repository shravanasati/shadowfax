package response

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/shravanasati/shadowfax/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamResponse(t *testing.T) {
	t.Run("simple stream function", func(t *testing.T) {
		streamFunc := func(w io.Writer, setTrailer TrailerSetter) error {
			_, err := w.Write([]byte("Hello "))
			if err != nil {
				return err
			}
			_, err = w.Write([]byte("World"))
			return err
		}

		resp := NewStreamResponse(streamFunc, nil)
		require.NotNil(t, resp)

		// Check headers
		headers := resp.GetHeaders()
		assert.Equal(t, "chunked", headers.Get("transfer-encoding"))
		assert.Empty(t, headers.Get("Trailer")) // No trailers specified

		// Check status code
		assert.Equal(t, StatusCode(200), resp.GetStatusCode())
	})

	t.Run("stream with trailers", func(t *testing.T) {
		trailerNames := []string{"X-Content-Length", "X-Checksum"}

		streamFunc := func(w io.Writer, setTrailer TrailerSetter) error {
			content := "Test content for trailers"
			_, err := w.Write([]byte(content))
			if err != nil {
				return err
			}

			// Set trailers
			setTrailer("X-Content-Length", fmt.Sprintf("%d", len(content)))
			setTrailer("X-Checksum", "abc123")
			return nil
		}

		resp := NewStreamResponse(streamFunc, trailerNames)
		require.NotNil(t, resp)

		// Check headers
		headers := resp.GetHeaders()
		assert.Equal(t, "chunked", headers.Get("transfer-encoding"))
		assert.Equal(t, "X-Content-Length, X-Checksum", headers.Get("Trailer"))
	})

	t.Run("stream function with error", func(t *testing.T) {
		expectedError := errors.New("stream error")

		streamFunc := func(w io.Writer, setTrailer TrailerSetter) error {
			_, err := w.Write([]byte("Partial"))
			if err != nil {
				return err
			}
			return expectedError
		}

		resp := NewStreamResponse(streamFunc, nil)
		require.NotNil(t, resp)

		// Reading from the body should eventually return the error
		body := resp.GetBody()
		_, err := io.ReadAll(body)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedError.Error())
	})
}

func TestStreamResponseWrite(t *testing.T) {
	streamFunc := func(w io.Writer, setTrailer TrailerSetter) error {
		content := "Streaming response test content"
		_, err := w.Write([]byte(content))
		if err != nil {
			return err
		}
		setTrailer("X-Content-Length", fmt.Sprintf("%d", len(content)))
		return nil
	}

	resp := NewStreamResponse(streamFunc, []string{"X-Content-Length"})

	var buf strings.Builder
	err := resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Check that it contains HTTP response parts
	assert.Contains(t, output, "HTTP/1.1 200 OK")
	assert.Contains(t, output, "transfer-encoding: chunked")
	assert.Contains(t, output, "trailer: X-Content-Length")

	// The content should be in chunked format
	assert.Contains(t, output, "Streaming response test content")
}

func TestStreamResponseMethods(t *testing.T) {
	streamFunc := func(w io.Writer, setTrailer TrailerSetter) error {
		_, err := w.Write([]byte("test"))
		return err
	}

	resp := NewStreamResponse(streamFunc, nil)

	// Test WithStatusCode
	modifiedResp := resp.WithStatusCode(202)
	assert.Equal(t, StatusCode(202), modifiedResp.GetStatusCode())

	// Test WithHeader
	modifiedResp = resp.WithHeader("X-Stream-Type", "live")
	assert.Equal(t, "live", modifiedResp.GetHeaders().Get("X-Stream-Type"))

	// Test WithHeaders
	headers := map[string]string{
		"Cache-Control": "no-cache",
		"Connection":    "keep-alive",
	}
	modifiedResp = resp.WithHeaders(headers)
	assert.Equal(t, "no-cache", modifiedResp.GetHeaders().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", modifiedResp.GetHeaders().Get("Connection"))
}

func TestStreamResponseLargeContent(t *testing.T) {
	// Test streaming large content in chunks
	streamFunc := func(w io.Writer, setTrailer TrailerSetter) error {
		// Write content in multiple chunks
		for i := range 100 {
			content := fmt.Sprintf("Chunk %d: %s\n", i, strings.Repeat("x", 100))
			_, err := w.Write([]byte(content))
			if err != nil {
				return err
			}
		}
		setTrailer("X-Chunks-Written", "100")
		return nil
	}

	resp := NewStreamResponse(streamFunc, []string{"X-Chunks-Written"})

	// Read all content
	body := resp.GetBody()
	content, err := io.ReadAll(body)
	require.NoError(t, err)

	// Verify we got all chunks
	contentStr := string(content)
	assert.Contains(t, contentStr, "Chunk 0:")
	assert.Contains(t, contentStr, "Chunk 99:")

	// Each chunk is about 111 bytes, so 100 chunks should be around 11100 bytes
	assert.Greater(t, len(content), 10000)
}

func TestStreamResponseReader(t *testing.T) {
	streamFunc := func(w io.Writer, setTrailer TrailerSetter) error {
		// Simulate a time-based stream
		for i := range 3 {
			content := fmt.Sprintf("Event %d\n", i)
			_, err := w.Write([]byte(content))
			if err != nil {
				return err
			}
			// Small delay to simulate real-time streaming
			time.Sleep(10 * time.Millisecond)
		}
		return nil
	}

	resp := NewStreamResponse(streamFunc, nil)

	// Test that Reader() returns a proper io.Reader
	reader := resp.Reader()
	require.NotNil(t, reader)

	// Read content in chunks to test streaming behavior
	buf := make([]byte, 100)
	var allContent strings.Builder

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			allContent.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	content := allContent.String()
	assert.Contains(t, content, "Event 0")
	assert.Contains(t, content, "Event 1")
	assert.Contains(t, content, "Event 2")
}

func TestChunkedReader(t *testing.T) {
	// Test the chunked reader directly
	testContent := "Hello, World!"
	reader := strings.NewReader(testContent)

	chunkedReader := &chunkedReader{r: reader, trailers: headers.NewHeaders()}

	// Read all content
	result, err := io.ReadAll(chunkedReader)
	require.NoError(t, err)

	resultStr := string(result)

	// Should contain the original content in chunked format
	assert.Contains(t, resultStr, testContent)
	// Should contain hex length prefix
	assert.Contains(t, resultStr, "d\r\n") // 13 in hex (length of "Hello, World!")
	// Should contain final chunk marker
	assert.Contains(t, resultStr, "0\r\n")
}
