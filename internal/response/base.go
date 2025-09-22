package response

import (
	"io"

	"github.com/shravanasati/shadowfax/internal/headers"
)

// BaseResponse is a basic implementation of the Response interface.
type BaseResponse struct {
	StatusCode StatusCode
	Headers    *headers.Headers
	Body       io.Reader
}

// NewBaseResponse creates a new BaseResponse with 200 status code.
func NewBaseResponse() Response {
	hs := headers.NewHeaders()
	return &BaseResponse{
		Headers:    hs,
		StatusCode: 200,
	}
}

// GetStatusCode returns the status code of the response.
func (r *BaseResponse) GetStatusCode() StatusCode {
	return r.StatusCode
}

// GetHeaders returns the headers of the response.
func (r *BaseResponse) GetHeaders() *headers.Headers {
	return r.Headers
}

// GetBody returns the body of the response.
func (r *BaseResponse) GetBody() io.Reader {
	return r.Body
}

// WithStatusCode sets the status code of the response.
func (r *BaseResponse) WithStatusCode(code StatusCode) Response {
	r.StatusCode = code
	return r
}

// WithHeader adds a header to the response.
func (r *BaseResponse) WithHeader(key, value string) Response {
	r.Headers.Add(key, value)
	return r
}

// WithHeaders adds multiple headers to the response.
func (r *BaseResponse) WithHeaders(headers map[string]string) Response {
	for key, value := range headers {
		r.Headers.Add(key, value)
	}
	return r
}

// WithBody sets the body of the response.
func (r *BaseResponse) WithBody(body io.Reader) Response {
	r.Body = body
	return r
}

// Write writes the response to the given writer.
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
