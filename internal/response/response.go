package response

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/shravanasati/shadowfax/internal/headers"
)

type Response struct {
	Headers    *headers.Headers
	StatusCode StatusCode
	Body       []byte
}

func (r *Response) statusLine() string {
	return fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.StatusCode, GetStatusReason(r.StatusCode))
}

func (r *Response) allHeaders() string {
	var b strings.Builder
	for k, v := range r.Headers.All() {
		b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	b.WriteString("\r\n")
	return b.String()
}

func NewResponse() *Response {
	return &Response{
		Headers:    headers.NewHeaders(),
		StatusCode: 200,
	}
}

func (r *Response) WithStatusCode(code StatusCode) *Response {
	r.StatusCode = code
	return r
}

func (r *Response) WithBody(body []byte) *Response {
	r.Body = body
	return r
}

func (r *Response) WithBodyString(body string) *Response {
	r.Body = []byte(body)
	return r
}

func (r *Response) WithHeader(key, value string) *Response {
	r.Headers.Add(key, value)
	return r
}

func (r *Response) WithHeaders(headers map[string]string) *Response {
	for key, value := range headers {
		r.Headers.Add(key, value)
	}
	return r
}

func (r *Response) Bytes() []byte {
	var b bytes.Buffer
	b.WriteString(r.statusLine())
	if r.Headers.Size() == 0 {
		r.Headers.AddDefaultHeaders(len(r.Body))
	}
	b.WriteString(r.allHeaders())
	b.Write((r.Body))
	return b.Bytes()
}
