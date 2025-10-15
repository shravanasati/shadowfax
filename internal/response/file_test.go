package response

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	modifiedResp = resp.WithHeader("x-custom-header", "val")
	assert.Equal(t, "val", modifiedResp.GetHeaders().Get("x-custom-header"))

	// Test WithHeaders - typical file headers
	headers := map[string]string{
		"content-disposition": "attachment; filename=\"test.txt\"",
		"cache-control":       "public, max-age=3600",
	}
	modifiedResp = resp.WithHeaders(headers)
	assert.Equal(t, "attachment; filename=\"test.txt\"", modifiedResp.GetHeaders().Get("content-disposition"))
	assert.Equal(t, "public, max-age=3600", modifiedResp.GetHeaders().Get("cache-control"))
}

func TestFileResponseContentTypeDetection(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name         string
		filename     string
		content      []byte
		expectedType string
	}{
		{
			name:         "HTML file by extension",
			filename:     "index.html",
			content:      []byte("<html><body>Hello World</body></html>"),
			expectedType: "text/html; charset=utf-8",
		},
		{
			name:         "CSS file by extension",
			filename:     "styles.css",
			content:      []byte("body { color: red; }"),
			expectedType: "text/css; charset=utf-8",
		},
		{
			name:         "JavaScript file by extension",
			filename:     "script.js",
			content:      []byte("console.log('Hello World');"),
			expectedType: "text/javascript; charset=utf-8",
		},
		{
			name:         "JSON file by extension",
			filename:     "data.json",
			content:      []byte(`{"key": "value"}`),
			expectedType: "application/json",
		},
		{
			name:         "PNG file by extension",
			filename:     "image.png",
			content:      []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG signature
			expectedType: "image/png",
		},
		{
			name:         "PDF file by extension",
			filename:     "document.pdf",
			content:      []byte("%PDF-1.4"),
			expectedType: "application/pdf",
		},
		{
			name:         "XML file by extension",
			filename:     "config.xml",
			content:      []byte("<?xml version=\"1.0\"?><root></root>"),
			expectedType: "text/xml; charset=utf-8",
		},
		{
			name:         "plain text file by extension",
			filename:     "readme.txt",
			content:      []byte("This is plain text content"),
			expectedType: "text/plain; charset=utf-8",
		},
		{
			name:         "unknown extension - content sniffing HTML",
			filename:     "file.unknown",
			content:      []byte("<html><head><title>Test</title></head><body></body></html>"),
			expectedType: "text/html; charset=utf-8",
		},
		{
			name:         "unknown extension - content sniffing JSON",
			filename:     "file.xyz",
			content:      []byte(`{"test": "data", "numbers": [1, 2, 3]}`),
			expectedType: "text/plain; charset=utf-8", // JSON content is detected as plain text by http.DetectContentType
		},
		{
			name:         "unknown extension - content sniffing binary",
			filename:     "file.bin",
			content:      []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}, // PNG signature with extra bytes
			expectedType: "image/png",
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

			headers := resp.GetHeaders()
			contentType := headers.Get("Content-Type")
			assert.Equal(t, tc.expectedType, contentType, "Content-Type mismatch for %s", tc.filename)
		})
	}
}

func TestFileResponseETagHeader(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name        string
		filename    string
		content     []byte
		description string
	}{
		{
			name:        "small text file",
			filename:    "small.txt",
			content:     []byte("Small content"),
			description: "ETag should be generated for small files",
		},
		{
			name:        "larger file",
			filename:    "larger.txt",
			content:     []byte(strings.Repeat("Content line\n", 100)),
			description: "ETag should be generated for larger files",
		},
		{
			name:        "empty file",
			filename:    "empty.txt",
			content:     []byte(""),
			description: "ETag should be generated even for empty files",
		},
		{
			name:        "binary file",
			filename:    "binary.bin",
			content:     []byte{0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
			description: "ETag should be generated for binary files",
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

			headers := resp.GetHeaders()
			etag := headers.Get("ETag")

			// ETag should be present
			assert.NotEmpty(t, etag, "ETag header should be present")

			// ETag should be properly quoted
			assert.True(t, strings.HasPrefix(etag, `"`), "ETag should start with quote")
			assert.True(t, strings.HasSuffix(etag, `"`), "ETag should end with quote")

			// ETag should be a valid hex string (without quotes)
			etagValue := strings.Trim(etag, `"`)
			assert.Regexp(t, "^[a-f0-9]+$", etagValue, "ETag value should be valid hex")

			// ETag should be consistent for the same file
			file2, err := os.Open(testFilePath)
			require.NoError(t, err)
			defer file2.Close()

			resp2 := NewFileResponse(file2)
			headers2 := resp2.GetHeaders()
			etag2 := headers2.Get("ETag")

			assert.Equal(t, etag, etag2, "ETag should be consistent for the same file")
		})
	}
}

func TestFileResponseETagUniqueness(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple files with different content
	files := []struct {
		name    string
		content []byte
	}{
		{"file1.txt", []byte("content 1")},
		{"file2.txt", []byte("content 2")},
		{"file3.txt", []byte("different content")},
	}

	var etags []string

	for i, f := range files {
		testFilePath := filepath.Join(tempDir, f.name)

		err := os.WriteFile(testFilePath, f.content, 0644)
		require.NoError(t, err)

		// Sleep briefly between file creations to ensure different modification times
		if i > 0 {
			time.Sleep(10 * time.Millisecond)
			// Touch the file to update its modification time
			err = os.WriteFile(testFilePath, f.content, 0644)
			require.NoError(t, err)
		}

		file, err := os.Open(testFilePath)
		require.NoError(t, err)
		defer file.Close()

		resp := NewFileResponse(file)
		headers := resp.GetHeaders()
		etag := headers.Get("ETag")

		etags = append(etags, etag)
	}

	// All ETags should be different
	for i := 0; i < len(etags); i++ {
		for j := i + 1; j < len(etags); j++ {
			assert.NotEqual(t, etags[i], etags[j], "ETags for different files should be unique: etag[%d]=%s vs etag[%d]=%s", i, etags[i], j, etags[j])
		}
	}
}

func TestFileResponseETagBasedOnModTime(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "modtime_test.txt")
	content := []byte("Test content for modification time")

	// Create file
	err := os.WriteFile(testFilePath, content, 0644)
	require.NoError(t, err)

	// Get ETag for original file
	file1, err := os.Open(testFilePath)
	require.NoError(t, err)
	defer file1.Close()

	resp1 := NewFileResponse(file1)
	headers1 := resp1.GetHeaders()
	etag1 := headers1.Get("ETag")

	// Sleep briefly to ensure different modification time
	// Note: In real scenarios, modification times would differ by more than milliseconds
	time.Sleep(10 * time.Millisecond)

	// Modify the file (this changes the modification time)
	err = os.WriteFile(testFilePath, content, 0644) // Same content, but new mod time
	require.NoError(t, err)

	// Get ETag for modified file
	file2, err := os.Open(testFilePath)
	require.NoError(t, err)
	defer file2.Close()

	resp2 := NewFileResponse(file2)
	headers2 := resp2.GetHeaders()
	etag2 := headers2.Get("ETag")

	// ETags should be different because modification time changed
	assert.NotEqual(t, etag1, etag2, "ETags should differ when file modification time changes")
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
