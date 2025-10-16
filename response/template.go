package response

import (
	"bytes"
	"html/template"
	"strconv"
)

// TemplateResponse is a response that renders HTML templates with data.
type TemplateResponse struct {
	Response
}

// NewTemplateResponse creates a new template response by rendering the given template with data.
// The templateContent should be a valid Go template string.
// The data parameter can be any struct, map, or value that the template can access.
func NewTemplateResponse(templateContent string, data any) (Response, error) {
	// Parse the template
	tmpl, err := template.New("response").Parse(templateContent)
	if err != nil {
		return nil, err
	}

	// Execute the template with the provided data
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	// Convert the rendered template to string
	renderedHTML := buf.String()

	// Create the base response with appropriate headers
	br := NewBaseResponse().
		WithHeader("content-type", "text/html; charset=utf-8").
		WithHeader("content-length", strconv.Itoa(len(renderedHTML))).
		WithBody(bytes.NewReader(buf.Bytes()))

	return &TemplateResponse{
		Response: br,
	}, nil
}

// NewTemplateResponseFromFile creates a new template response by loading and rendering a template file.
// The templatePath should be the path to a template file.
// The data parameter can be any struct, map, or value that the template can access.
func NewTemplateResponseFromFile(templatePath string, data any) (Response, error) {
	// Parse the template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, err
	}

	// Execute the template with the provided data
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	// Convert the rendered template to string
	renderedHTML := buf.String()

	// Create the base response with appropriate headers
	br := NewBaseResponse().
		WithHeader("content-type", "text/html; charset=utf-8").
		WithHeader("content-length", strconv.Itoa(len(renderedHTML))).
		WithBody(bytes.NewReader(buf.Bytes()))

	return &TemplateResponse{
		Response: br,
	}, nil
}

// NewTemplateResponseWithFuncs creates a new template response with custom template functions.
// The templateContent should be a valid Go template string.
// The funcMap contains custom functions that can be used in the template.
// The data parameter can be any struct, map, or value that the template can access.
func NewTemplateResponseWithFuncs(templateContent string, funcMap template.FuncMap, data any) (Response, error) {
	// Parse the template with custom functions
	tmpl, err := template.New("response").Funcs(funcMap).Parse(templateContent)
	if err != nil {
		return nil, err
	}

	// Execute the template with the provided data
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	// Convert the rendered template to string
	renderedHTML := buf.String()

	// Create the base response with appropriate headers
	br := NewBaseResponse().
		WithHeader("content-type", "text/html; charset=utf-8").
		WithHeader("content-length", strconv.Itoa(len(renderedHTML))).
		WithBody(bytes.NewReader(buf.Bytes()))

	return &TemplateResponse{
		Response: br,
	}, nil
}
