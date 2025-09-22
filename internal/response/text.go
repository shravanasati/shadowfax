package response

import (
	"strconv"
	"strings"
)

// TextResponse is a response that sends plain text.
type TextResponse struct {
	Response
}

// NewTextResponse creates a new text response.
func NewTextResponse(body string) Response {
	br := NewBaseResponse().
		WithHeader("content-type", "text/plain").
		WithHeader("content-length", strconv.Itoa(len(body))).
		WithBody(strings.NewReader(body))

	return &TextResponse{
		Response: br,
	}
}
