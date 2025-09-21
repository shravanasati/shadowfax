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

func NewBaseResponse() Response {
	hs := headers.NewHeaders()
	hs.Add("connection", "close")
	return &BaseResponse{
		Headers:    hs,
		StatusCode: 200,
	}
}

func (r *BaseResponse) GetStatusCode() StatusCode {
	return r.StatusCode
}

func (r *BaseResponse) GetHeaders() *headers.Headers {
	return r.Headers
}

func (r *BaseResponse) GetBody() io.Reader {
	return r.Body
}

func (r *BaseResponse) WithStatusCode(code StatusCode) Response {
	r.StatusCode = code
	return r
}

func (r *BaseResponse) WithHeader(key, value string) Response {
	r.Headers.Add(key, value)
	return r
}

func (r *BaseResponse) WithHeaders(headers map[string]string) Response {
	for key, value := range headers {
		r.Headers.Add(key, value)
	}
	return r
}

func (r *BaseResponse) WithBody(body io.Reader) Response {
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

	if r.Body != nil {
		err = rw.WriteBody(r.Body)
		if err != nil {
			return err
		}
	}
	return nil
}
