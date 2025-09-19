package response

import (
	"io"

	"github.com/shravanasati/shadowfax/internal/headers"
)

// BaseResponse struct for fluent method chaining.
type BaseResponse struct {
	StatusCode StatusCode
	Headers    *headers.Headers
	Body       io.Reader
}

func NewBaseResponse() *BaseResponse {
	hs := headers.NewHeaders()
	hs.Add("connection", "close")
	return &BaseResponse{
		Headers:    hs,
		StatusCode: 200,
	}
}

func (r *BaseResponse) WithStatusCode(code StatusCode) *BaseResponse {
	r.StatusCode = code
	return r
}

func (r *BaseResponse) WithHeader(key, value string) *BaseResponse {
	r.Headers.Add(key, value)
	return r
}

func (r *BaseResponse) WithHeaders(headers map[string]string) *BaseResponse {
	for key, value := range headers {
		r.Headers.Add(key, value)
	}
	return r
}

func (r *BaseResponse) WithBody(body io.Reader) *BaseResponse {
	r.Body = body
	return r
}


func (r *BaseResponse) Write(w io.Writer) error {
	rw := NewResponseWriter(w)
	err := rw.WriteStatusLine(r.StatusCode)
	if err != nil {
		return err
	}

	err = rw.WriteHeaders(r.Headers)
	if err != nil {
		return err
	}

	err = rw.WriteBody(r.Body)
	if err != nil {
		return err
	}
	return nil
}
