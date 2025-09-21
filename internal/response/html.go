package response

import (
	"strconv"
	"strings"
)

type HTMLResponse struct {
	Response
}

func NewHTMLResponse(body string) Response {
	br := NewBaseResponse().
		WithHeader("content-type", "text/html").
		WithHeader("content-length", strconv.Itoa(len(body))).
		WithBody(strings.NewReader(body))

	return &HTMLResponse{
		Response: br,
	}
}