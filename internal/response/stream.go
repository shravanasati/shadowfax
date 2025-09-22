package response

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/shravanasati/shadowfax/internal/headers"
)

// TrailerSetter is a function that sets a trailer header.
type TrailerSetter func(key, value string)

// StreamFunc is a function that writes to a stream.
type StreamFunc func(w io.Writer, setTrailer TrailerSetter) error

// Reader returns a reader for the stream.
func (sr *StreamResponse) Reader() io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		setTrailer := func(key, value string) {
			sr.Trailers.Add(key, value)
		}

		if err := sr.Stream(pw, setTrailer); err != nil {
			// propagate error to the reader
			pw.CloseWithError(err)
		}
	}()

	return pr
}

type chunkedReader struct {
	r        io.Reader
	buf      bytes.Buffer
	eof      bool
	trailers *headers.Headers
}

func (cr *chunkedReader) Read(p []byte) (int, error) {
	// Serve from buffer first
	if cr.buf.Len() > 0 {
		n, _ := cr.buf.Read(p)
		return n, nil
	}

	if cr.eof {
		return 0, io.EOF
	}

	// Read from underlying
	raw := make([]byte, 4096)
	n, err := cr.r.Read(raw)
	if n > 0 {
		// Build chunk: size\r\n + data + \r\n
		header := fmt.Appendf(nil, "%x\r\n", n)
		footer := []byte("\r\n")

		cr.buf.Write(header)
		cr.buf.Write(raw[:n])
		cr.buf.Write(footer)

		// Serve from buf
		n, _ := cr.buf.Read(p)
		return n, nil
	}

	if err == io.EOF {
		// Write final chunk with trailers
		cr.buf.WriteString("0\r\n")

		if cr.trailers.Size() > 0 {
			fmt.Println(cr.trailers)
			for key, value := range cr.trailers.All() {
				trailerLine := fmt.Sprintf("%s: %s\r\n", key, value)
				cr.buf.WriteString(trailerLine)
			}
		}

		// Final CRLF to end the response
		cr.buf.WriteString("\r\n")

		cr.eof = true
		n, _ := cr.buf.Read(p)
		return n, nil
	}

	return 0, err
}

// StreamResponse is a response that streams data.
type StreamResponse struct {
	Response
	Stream      StreamFunc
	trailerList []string
	Trailers    *headers.Headers
}

// NewStreamResponse creates a new stream response.
func NewStreamResponse(sf StreamFunc, trailers []string) *StreamResponse {
	sr := &StreamResponse{
		Response: NewBaseResponse().
			WithHeader("transfer-encoding", "chunked"),
		Stream:      sf,
		trailerList: trailers,
		Trailers:    headers.NewHeaders(),
	}

	if len(trailers) > 0 {
		sr.WithHeader("Trailer", strings.Join(trailers, ", "))
	}

	sr.WithBody(&chunkedReader{
		r:        sr.Reader(),
		trailers: sr.Trailers,
	})

	return sr
}
