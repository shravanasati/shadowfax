package response

import (
	"fmt"
	"io"

	"github.com/shravanasati/shadowfax/internal/headers"
)

type Response interface {
	Write(io.Writer) error
}

type ResponseWriter struct {
	conn  io.Writer
	state responseState
}

func NewResponseWriter(conn io.Writer) *ResponseWriter {
	return &ResponseWriter{conn: conn, state: newResponseState()}
}

func (rw *ResponseWriter) WriteStatusLine(statusCode StatusCode) error {
	if rw.state != stateStatusLine {
		return ErrStatusLineAlreadyWritten
	}
	_, err := fmt.Fprintf(rw.conn, "HTTP/1.1 %d %s\r\n", statusCode, GetStatusReason(statusCode))
	if err != nil {
		return err
	}

	rw.state = rw.state.advance()
	return nil
}

func (rw *ResponseWriter) WriteHeaders(h *headers.Headers) error {
	if rw.state != stateHeaders {
		return ErrHeadersAlreadyWritten
	}
	for k, v := range h.All() {
		fmt.Fprintf(rw.conn, "%s: %s\r\n", k, v)
	}
	rw.conn.Write([]byte("\r\n"))
	rw.state = rw.state.advance()
	return nil
}

func (rw *ResponseWriter) WriteBody(b io.Reader) error {
	if rw.state != stateBody {
		return ErrNoBodyState
	}
	_, err := io.Copy(rw.conn, b)
	if err != nil {
		return err
	}
	rw.state = rw.state.advance()
	return nil
}

// func NewResponse() *Response {
// 	return &Response{
// 		Headers:    headers.NewHeaders(),
// 		StatusCode: 200,
// 	}
// }

// func (r *Response) WithStatusCode(code StatusCode) *Response {
// 	r.StatusCode = code
// 	return r
// }

// func (r *Response) WithBody(body []byte) *Response {
// 	r.Body = body
// 	return r
// }

// func (r *Response) WithBodyString(body string) *Response {
// 	r.Body = []byte(body)
// 	return r
// }

// func (r *Response) WithHeader(key, value string) *Response {
// 	r.Headers.Add(key, value)
// 	return r
// }

// func (r *Response) WithHeaders(headers map[string]string) *Response {
// 	for key, value := range headers {
// 		r.Headers.Add(key, value)
// 	}
// 	return r
// }

// func (r *Response) Bytes() []byte {
// 	var b bytes.Buffer
// 	b.WriteString(r.statusLine())
// 	if r.Headers.Size() == 0 {
// 		r.Headers.AddDefaultHeaders(len(r.Body))
// 	}
// 	b.WriteString(r.allHeaders())
// 	b.Write((r.Body))
// 	return b.Bytes()
// }
