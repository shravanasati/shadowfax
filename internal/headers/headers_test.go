package headers

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParsing(t *testing.T) {

	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069")
	err := headers.ParseFieldLine(data)
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
	err = headers.ParseFieldLine(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hval = headers.Get("Host")
	assert.Equal(t, "localhost:42069", hval)

	// Test: Valid 2 headers with existing headers
	headers = NewHeaders()
	headers.Add("User-Agent", "curl/7.81.0")
	data = []byte("Host: localhost:42069")
	err = headers.ParseFieldLine(data)
	require.NoError(t, err)
	data = []byte("Accept: */*")
	err = headers.ParseFieldLine(data)
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
	err = headers.ParseFieldLine(data)
	require.Error(t, err)

	// Test: Invalid spacing header
	// https://datatracker.ietf.org/doc/html/rfc9112#section-5
	headers = NewHeaders()
	data = []byte("       Host : localhost:42069       ")
	err = headers.ParseFieldLine(data)
	require.Error(t, err)

	// Test: Invalid character in header key
	headers = NewHeaders()
	data = []byte("H©st: localhost:42069")
	err = headers.ParseFieldLine(data)
	require.Error(t, err)

	// Test: Multiple values of the same header
	headers = NewHeaders()
	data = []byte("Accept: text/html")
	err = headers.ParseFieldLine(data)
	require.NoError(t, err)
	data = []byte("Accept: application/json")
	err = headers.ParseFieldLine(data)
	require.NoError(t, err)
	hval = headers.Get("Accept")
	assert.Equal(t, "text/html, application/json", hval)

	// Test: Multiline header value (folded header)
	headers = NewHeaders()
	// Simulate a header value split across two lines (second line starts with a space)
	err = headers.ParseFieldLine([]byte("X-Long-Header: part1"))
	require.NoError(t, err)
	err = headers.ParseFieldLine([]byte(" part2"))
	require.Error(t, err)

	t.Run("Invalid Header Names", func(t *testing.T) {
		invalidNames := []string{
			"Invalid Name:",          // space in name
			"Invalid@Name:",          // @ in name
			"Invalid/Name:",          // / in name
			"Name\x00:",              // null character in name
			"Name\x7f:",              // delete character in name
			"Name with space: value", // space in name
		}

		for _, name := range invalidNames {
			headers := NewHeaders()
			err := headers.ParseFieldLine([]byte(name + " value"))
			assert.Error(t, err, "Expected error for header name: `%s`", name)
		}
	})

	t.Run("Invalid Header Values", func(t *testing.T) {
		invalidValues := []string{
			"Value with\x00null", // null character
			"Value with\x07bell", // bell character
			"Value with\x1funit separator",
		}

		for _, value := range invalidValues {
			headers := NewHeaders()
			line := "Valid-Name: " + value
			err := headers.ParseFieldLine([]byte(line))
			assert.Error(t, err, "Expected error for header value: %s", value)
		}
	})
}

func TestHeadersMethods(t *testing.T) {
	t.Run("Add and Get", func(t *testing.T) {
		headers := NewHeaders()

		// Simple add and get
		headers.Add("Content-Type", "application/json")
		assert.Equal(t, "application/json", headers.Get("Content-Type"))

		// Case-insensitivity of get
		assert.Equal(t, "application/json", headers.Get("content-type"))
		assert.Equal(t, "application/json", headers.Get("CONTENT-TYPE"))

		// Add multiple values to the same header
		headers.Add("Accept", "text/html")
		headers.Add("Accept", "application/xhtml+xml")
		assert.Equal(t, "text/html, application/xhtml+xml", headers.Get("Accept"))

		// Add with different cases for the same key
		headers.Add("X-Custom-Header", "value1")
		headers.Add("x-custom-header", "value2")
		assert.Equal(t, "value1, value2", headers.Get("X-Custom-Header"))

		// Get non-existent header
		assert.Equal(t, "", headers.Get("Non-Existent-Header"))

		// Add header with empty value
		headers.Add("Empty-Value", "")
		assert.Equal(t, "", headers.Get("Empty-Value"))

		// Add header with leading/trailing whitespace in value
		headers.Add("Whitespace-Value", "  some value  ")
		assert.Equal(t, "  some value  ", headers.Get("Whitespace-Value"))
	})

	t.Run("Remove", func(t *testing.T) {
		headers := NewHeaders()
		headers.Add("Content-Type", "application/json")
		headers.Add("X-Custom", "value")

		// Simple remove
		headers.Remove("X-Custom")
		assert.Equal(t, "", headers.Get("X-Custom"))
		assert.Equal(t, "application/json", headers.Get("Content-Type"))

		// Case-insensitivity of remove
		headers.Remove("CONTENT-TYPE")
		assert.Equal(t, "", headers.Get("Content-Type"))

		// Remove non-existent header
		headers.Remove("Non-Existent-Header") // Should not panic
	})

	t.Run("Size", func(t *testing.T) {
		headers := NewHeaders()
		assert.Equal(t, 0, headers.Size())

		headers.Add("A", "1")
		assert.Equal(t, 1, headers.Size())

		headers.Add("B", "2")
		assert.Equal(t, 2, headers.Size())

		// Adding same header again should not increase size
		headers.Add("A", "11")
		assert.Equal(t, 2, headers.Size())

		headers.Remove("A")
		assert.Equal(t, 1, headers.Size())

		headers.Remove("B")
		assert.Equal(t, 0, headers.Size())

		// Removing non-existent header should not change size
		headers.Remove("C")
		assert.Equal(t, 0, headers.Size())
	})

	t.Run("All", func(t *testing.T) {
		t.Run("empty headers", func(t *testing.T) {
			headers := NewHeaders()
			count := 0
			for range headers.All() {
				count++
			}
			assert.Equal(t, 0, count)
		})

		t.Run("with some headers", func(t *testing.T) {
			headers := NewHeaders()
			headers.Add("Content-Type", "application/json")
			headers.Add("Accept", "text/html")
			headers.Add("X-Custom", "value")

			expected := map[string]string{
				"content-type": "application/json",
				"accept":       "text/html",
				"x-custom":     "value",
			}

			actual := maps.Collect(headers.All())

			assert.Equal(t, expected, actual)
			assert.Equal(t, len(expected), headers.Size())
		})
	})
}
