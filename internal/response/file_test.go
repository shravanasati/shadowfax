package response

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileResponse(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Test case 1: Valid file with known size
	t.Run("valid file with size", func(t *testing.T) {
		testContent := "This is test file content for file response testing"
		testFilePath := filepath.Join(tempDir, "test1.txt")
		
		err := os.WriteFile(testFilePath, []byte(testContent), 0644)
		require.NoError(t, err)

		file, err := os.Open(testFilePath)
		require.NoError(t, err)
		defer file.Close()

		resp := NewFileResponse(file)
		require.NotNil(t, resp)

		// Check headers - should have Content-Length
		headers := resp.GetHeaders()
		contentLength := headers.Get("Content-Length")
		assert.NotEmpty(t, contentLength)
		assert.Equal(t, "51", contentLength) // Length of test content

		// Should not have Transfer-Encoding header for known size
		assert.Empty(t, headers.Get("Transfer-Encoding"))

		// Check body
		body := resp.GetBody()
		require.NotNil(t, body)
		
		bodyBytes, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, testContent, string(bodyBytes))

		// Check status code
		assert.Equal(t, StatusCode(200), resp.GetStatusCode())
	})

	// Test case 2: Empty file
	t.Run("empty file", func(t *testing.T) {
		testFilePath := filepath.Join(tempDir, "empty.txt")
		
		err := os.WriteFile(testFilePath, []byte(""), 0644)
		require.NoError(t, err)

		file, err := os.Open(testFilePath)
		require.NoError(t, err)
		defer file.Close()

		resp := NewFileResponse(file)
		require.NotNil(t, resp)

		headers := resp.GetHeaders()
		assert.Equal(t, "0", headers.Get("Content-Length"))
	})

	// Test case 3: Large file
	t.Run("large file", func(t *testing.T) {
		// Create a larger test file (1MB)
		largeContent := strings.Repeat("A", 1024*1024)
		testFilePath := filepath.Join(tempDir, "large.txt")
		
		err := os.WriteFile(testFilePath, []byte(largeContent), 0644)
		require.NoError(t, err)

		file, err := os.Open(testFilePath)
		require.NoError(t, err)
		defer file.Close()

		resp := NewFileResponse(file)
		require.NotNil(t, resp)

		headers := resp.GetHeaders()
		assert.Equal(t, "1048576", headers.Get("Content-Length")) // 1MB
	})
}

func TestFileResponseWrite(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testContent := "File response write test content"
	testFilePath := filepath.Join(tempDir, "write_test.txt")
	
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	require.NoError(t, err)

	file, err := os.Open(testFilePath)
	require.NoError(t, err)
	defer file.Close()

	resp := NewFileResponse(file)

	var buf strings.Builder
	err = resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()
	
	// Check that it contains HTTP response parts
	assert.Contains(t, output, "HTTP/1.1 200 OK")
	assert.Contains(t, output, "content-length: 32") // Length of test content
	assert.Contains(t, output, testContent)
}

func TestFileResponseMethods(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testContent := "Method test content"
	testFilePath := filepath.Join(tempDir, "method_test.txt")
	
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	require.NoError(t, err)

	file, err := os.Open(testFilePath)
	require.NoError(t, err)
	defer file.Close()

	resp := NewFileResponse(file)

	// Test WithStatusCode
	modifiedResp := resp.WithStatusCode(206) // Partial Content
	assert.Equal(t, StatusCode(206), modifiedResp.GetStatusCode())

	// Test WithHeader - common for file responses
	modifiedResp = resp.WithHeader("content-type", "text/plain")
	assert.Equal(t, "text/plain", modifiedResp.GetHeaders().Get("content-type"))

	// Test WithHeaders - typical file headers
	headers := map[string]string{
		"content-disposition": "attachment; filename=\"test.txt\"",
		"cache-control":       "public, max-age=3600",
		"etag":               "\"test-etag\"",
	}
	modifiedResp = resp.WithHeaders(headers)
	assert.Equal(t, "attachment; filename=\"test.txt\"", modifiedResp.GetHeaders().Get("content-disposition"))
	assert.Equal(t, "public, max-age=3600", modifiedResp.GetHeaders().Get("cache-control"))
	assert.Equal(t, "\"test-etag\"", modifiedResp.GetHeaders().Get("etag"))
}

// Note: Testing chunked encoding case would require creating a file descriptor
// that fails stat operations, which is complex to set up in a portable way.
// The chunked fallback path is tested indirectly through integration tests.

func TestFileResponseDifferentFileTypes(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name     string
		content  []byte
		filename string
	}{
		{
			name:     "text file",
			content:  []byte("Hello, World!"),
			filename: "hello.txt",
		},
		{
			name:     "binary data",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG header
			filename: "test.png",
		},
		{
			name:     "json file",
			content:  []byte(`{"key": "value", "number": 42}`),
			filename: "data.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFilePath := filepath.Join(tempDir, tc.filename)
			
			err := os.WriteFile(testFilePath, tc.content, 0644)
			require.NoError(t, err)

			file, err := os.Open(testFilePath)
			require.NoError(t, err)
			defer file.Close()

			resp := NewFileResponse(file)
			require.NotNil(t, resp)

			// Verify content
			body := resp.GetBody()
			bodyBytes, err := io.ReadAll(body)
			require.NoError(t, err)
			assert.Equal(t, tc.content, bodyBytes)

			// Verify content length
			headers := resp.GetHeaders()
			assert.Equal(t, strconv.Itoa(len(tc.content)), headers.Get("Content-Length"))
		})
	}
}