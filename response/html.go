package response

import (
	"strconv"
	"strings"
)

// HTMLResponse is a response that sends HTML.
type HTMLResponse struct {
	Response
}

// NewHTMLResponse creates a new HTML response.
func NewHTMLResponse(body string) Response {
	br := NewBaseResponse().
		WithHeader("content-type", "text/html").
		WithHeader("content-length", strconv.Itoa(len(body))).
		WithBody(strings.NewReader(body))

	return &HTMLResponse{
		Response: br,
	}
}
