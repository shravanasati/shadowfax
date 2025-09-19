package response

import (
	"strconv"
	"strings"
)

type TextResponse struct {
	*BaseResponse
}

func NewTextResponse(body string) *TextResponse {
	br := NewBaseResponse().
		WithHeader("content-type", "text/plain").
		WithHeader("content-length", strconv.Itoa(len(body))).
		WithBody(strings.NewReader(body))

	return &TextResponse{
		BaseResponse: br,
	}
}
