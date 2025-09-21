package response

import (
	"strconv"
	"strings"
)

type TextResponse struct {
	Response
}

func NewTextResponse(body string) Response {
	br := NewBaseResponse().
		WithHeader("content-type", "text/plain").
		WithHeader("content-length", strconv.Itoa(len(body))).
		WithBody(strings.NewReader(body))

	return &TextResponse{
		Response: br,
	}
}
