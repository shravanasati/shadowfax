package response

import (
	"fmt"
	"io"

	"github.com/shravanasati/shadowfax/internal/headers"
)

type Response interface {
	Write(io.Writer) error

	GetStatusCode() StatusCode
	GetHeaders() *headers.Headers
	GetBody() io.Reader

	WithStatusCode(StatusCode) Response
	WithHeader(key, value string) Response
	WithHeaders(map[string]string) Response
	WithBody(io.Reader) Response
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
	if rw.conn == nil {
		return fmt.Errorf("(write status line) writer is nil")
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
	if rw.conn == nil {
		return fmt.Errorf("(write headers) writer is nil")
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
	if rw.conn == nil {
		return fmt.Errorf("(write body) writer is nil")
	}
	_, err := io.Copy(rw.conn, b)
	if err != nil {
		return err
	}
	rw.state = rw.state.advance()
	return nil
}
