package response

import (
	"fmt"
	"io"

	"github.com/shravanasati/shadowfax/internal/headers"
)

// Response is the interface that all responses must implement.
type Response interface {
	// Write writes the response to the given writer.
	Write(io.Writer) error

	// GetStatusCode returns the status code of the response.
	GetStatusCode() StatusCode
	// GetHeaders returns the headers of the response.
	GetHeaders() *headers.Headers
	// GetBody returns the body of the response.
	GetBody() io.Reader

	// WithStatusCode sets the status code of the response.
	WithStatusCode(StatusCode) Response
	// WithHeader adds a header to the response.
	WithHeader(key, value string) Response
	// WithHeaders adds multiple headers to the response.
	WithHeaders(map[string]string) Response
	// WithBody sets the body of the response.
	WithBody(io.Reader) Response
}

// ResponseWriter is a writer for responses.
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
