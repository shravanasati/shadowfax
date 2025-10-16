package response

import (
	"html/template"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateResponse(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		data         any
		expectedBody string
		expectError  bool
	}{
		{
			name:         "simple template with string data",
			template:     "<h1>Hello, {{.}}!</h1>",
			data:         "World",
			expectedBody: "<h1>Hello, World!</h1>",
			expectError:  false,
		},
		{
			name:     "template with struct data",
			template: "<h1>{{.Title}}</h1><p>{{.Content}}</p>",
			data: struct {
				Title   string
				Content string
			}{
				Title:   "Welcome",
				Content: "This is a test page.",
			},
			expectedBody: "<h1>Welcome</h1><p>This is a test page.</p>",
			expectError:  false,
		},
		{
			name:     "template with map data",
			template: "<h1>{{.title}}</h1><ul>{{range .items}}<li>{{.}}</li>{{end}}</ul>",
			data: map[string]any{
				"title": "My List",
				"items": []string{"Item 1", "Item 2", "Item 3"},
			},
			expectedBody: "<h1>My List</h1><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>",
			expectError:  false,
		},
		{
			name:        "invalid template syntax",
			template:    "<h1>{{.InvalidSyntax",
			data:        "test",
			expectError: true,
		},
		{
			name:        "template with missing field",
			template:    "<h1>{{.NonExistentField}}</h1>",
			data:        struct{ Name string }{Name: "test"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewTemplateResponse(tt.template, tt.data)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			// Check headers
			headers := resp.GetHeaders()
			assert.Equal(t, "text/html; charset=utf-8", headers.Get("content-type"))
			assert.Equal(t, strconv.Itoa(len(tt.expectedBody)), headers.Get("content-length"))

			// Check body
			body := resp.GetBody()
			require.NotNil(t, body)

			bodyBytes, err := io.ReadAll(body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody, string(bodyBytes))

			// Check status code
			assert.Equal(t, StatusCode(200), resp.GetStatusCode())
		})
	}
}

func TestNewTemplateResponseWithFuncs(t *testing.T) {
	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
		"add": func(a, b int) int {
			return a + b
		},
	}

	tests := []struct {
		name         string
		template     string
		funcMap      template.FuncMap
		data         any
		expectedBody string
		expectError  bool
	}{
		{
			name:     "template with custom function",
			template: "<h1>{{upper .name}}</h1><p>Result: {{add .a .b}}</p>",
			funcMap:  funcMap,
			data: map[string]any{
				"name": "hello world",
				"a":    5,
				"b":    3,
			},
			expectedBody: "<h1>HELLO WORLD</h1><p>Result: 8</p>",
			expectError:  false,
		},
		{
			name:        "template with undefined function",
			template:    "<h1>{{undefined .name}}</h1>",
			funcMap:     funcMap,
			data:        map[string]any{"name": "test"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewTemplateResponseWithFuncs(tt.template, tt.funcMap, tt.data)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			// Check body
			body := resp.GetBody()
			require.NotNil(t, body)

			bodyBytes, err := io.ReadAll(body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody, string(bodyBytes))
		})
	}
}

func TestTemplateResponseWrite(t *testing.T) {
	template := "<h1>{{.title}}</h1>"
	data := map[string]any{"title": "Test Page"}

	resp, err := NewTemplateResponse(template, data)
	require.NoError(t, err)

	var buf strings.Builder
	err = resp.Write(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Check that it contains HTTP response parts
	assert.Contains(t, output, "HTTP/1.1 200 OK")
	assert.Contains(t, output, "content-type: text/html; charset=utf-8")
	assert.Contains(t, output, "<h1>Test Page</h1>")
}

func TestTemplateResponseMethods(t *testing.T) {
	template := "<h1>Test</h1>"
	data := "test"

	resp, err := NewTemplateResponse(template, data)
	require.NoError(t, err)

	// Test WithStatusCode
	modifiedResp := resp.WithStatusCode(404)
	assert.Equal(t, StatusCode(404), modifiedResp.GetStatusCode())

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
