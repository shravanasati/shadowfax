package response

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONResponse(t *testing.T) {
	tests := []struct {
		name         string
		data         any
		expectedBody string
		expectError  bool
	}{
		{
			name:         "simple string",
			data:         "hello world",
			expectedBody: `"hello world"`,
			expectError:  false,
		},
		{
			name:         "integer",
			data:         42,
			expectedBody: "42",
			expectError:  false,
		},
		{
			name:         "boolean",
			data:         true,
			expectedBody: "true",
			expectError:  false,
		},
		{
			name: "struct",
			data: struct {
				Name  string `json:"name"`
				Age   int    `json:"age"`
				Email string `json:"email"`
			}{
				Name:  "John Doe",
				Age:   30,
				Email: "john@example.com",
			},
			expectedBody: `{"name":"John Doe","age":30,"email":"john@example.com"}`,
			expectError:  false,
		},
		{
			name: "map",
			data: map[string]any{
				"message": "success",
				"count":   5,
				"active":  true,
			},
			expectedBody: `{"active":true,"count":5,"message":"success"}`,
			expectError:  false,
		},
		{
			name:         "slice",
			data:         []string{"apple", "banana", "cherry"},
			expectedBody: `["apple","banana","cherry"]`,
			expectError:  false,
		},
		{
			name:         "nil",
			data:         nil,
			expectedBody: "null",
			expectError:  false,
		},
		{
			name:        "unmarshalable data",
			data:        make(chan int),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewJSONResponse(tt.data)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			// Check headers
			headers := resp.GetHeaders()
			assert.Equal(t, "application/json", headers.Get("content-type"))
			assert.Equal(t, strconv.Itoa(len(tt.expectedBody)), headers.Get("content-length"))

			// Check body
			body := resp.GetBody()
			require.NotNil(t, body)
			
			bodyBytes, err := io.ReadAll(body)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expectedBody, string(bodyBytes))

			// Check status code
			assert.Equal(t, StatusCode(200), resp.GetStatusCode())
		})
	}
}

func TestJSONResponseWrite(t *testing.T) {
	data := map[string]any{
		"message": "test",
		"status":  "ok",
	}
	
	resp, err := NewJSONResponse(data)
	require.NoError(t, err)

	var buf strings.Builder
	err = resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()
	
	// Check that it contains HTTP response parts
	assert.Contains(t, output, "HTTP/1.1 200 OK")
	assert.Contains(t, output, "content-type: application/json")
	assert.Contains(t, output, `"message":"test"`)
	assert.Contains(t, output, `"status":"ok"`)
}

func TestJSONResponseMethods(t *testing.T) {
	data := map[string]string{"test": "value"}
	
	resp, err := NewJSONResponse(data)
	require.NoError(t, err)

	// Test WithStatusCode
	modifiedResp := resp.WithStatusCode(201)
	assert.Equal(t, StatusCode(201), modifiedResp.GetStatusCode())

	// Test WithHeader
	modifiedResp = resp.WithHeader("X-Custom", "test-value")
	assert.Equal(t, "test-value", modifiedResp.GetHeaders().Get("X-Custom"))

	// Test WithHeaders
	headers := map[string]string{
		"X-Test-1": "value1",
		"X-Test-2": "value2",
	}
	modifiedResp = resp.WithHeaders(headers)
	assert.Equal(t, "value1", modifiedResp.GetHeaders().Get("X-Test-1"))
	assert.Equal(t, "value2", modifiedResp.GetHeaders().Get("X-Test-2"))
}

func TestJSONResponseComplexData(t *testing.T) {
	type User struct {
		ID       int      `json:"id"`
		Name     string   `json:"name"`
		Email    string   `json:"email"`
		Tags     []string `json:"tags"`
		Metadata map[string]any `json:"metadata"`
	}

	complexData := struct {
		Users  []User `json:"users"`
		Total  int    `json:"total"`
		Active bool   `json:"active"`
	}{
		Users: []User{
			{
				ID:    1,
				Name:  "Alice",
				Email: "alice@example.com",
				Tags:  []string{"admin", "active"},
				Metadata: map[string]any{
					"last_login": "2023-01-01",
					"preferences": map[string]bool{
						"notifications": true,
						"dark_mode":     false,
					},
				},
			},
			{
				ID:    2,
				Name:  "Bob",
				Email: "bob@example.com",
				Tags:  []string{"user"},
				Metadata: map[string]any{
					"last_login": "2023-01-02",
				},
			},
		},
		Total:  2,
		Active: true,
	}

	resp, err := NewJSONResponse(complexData)
	require.NoError(t, err)

	// Verify the JSON can be unmarshaled back
	body := resp.GetBody()
	bodyBytes, err := io.ReadAll(body)
	require.NoError(t, err)

	var unmarshaled map[string]any
	err = json.Unmarshal(bodyBytes, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, float64(2), unmarshaled["total"]) // JSON numbers are float64
	assert.Equal(t, true, unmarshaled["active"])
	
	users, ok := unmarshaled["users"].([]any)
	require.True(t, ok)
	assert.Len(t, users, 2)
}