package response

import (
	"fmt"
	"io"
)

type StreamFunc func(io.Writer) error

func (sf StreamFunc) Reader() io.Reader {
    pr, pw := io.Pipe()

    go func() {
        defer pw.Close()
        if err := sf(pw); err != nil {
            // propagate error to the reader
            pw.CloseWithError(err)
        }
    }()

    return pr
}

type chunkedReader struct {
    r   io.Reader
    buf []byte
    eof bool
}

func (cr *chunkedReader) Read(p []byte) (int, error) {
    // Serve from buffer first
    if len(cr.buf) > 0 {
        n := copy(p, cr.buf)
        cr.buf = cr.buf[n:]
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
        header := []byte(fmt.Sprintf("%x\r\n", n))
        footer := []byte("\r\n")

        cr.buf = append(cr.buf, header...)
        cr.buf = append(cr.buf, raw[:n]...)
        cr.buf = append(cr.buf, footer...)

        // Serve from buf
        n := copy(p, cr.buf)
        cr.buf = cr.buf[n:]
        return n, nil
    }

    if err == io.EOF {
        cr.buf = []byte("0\r\n\r\n")
        cr.eof = true
        n := copy(p, cr.buf)
        cr.buf = cr.buf[n:]
        return n, nil
    }

    return 0, err
}


type StreamResponse struct {
	*BaseResponse
	Stream StreamFunc
}

func NewStreamResponse(sf StreamFunc) *StreamResponse {
	br := NewBaseResponse().
		WithHeader("transfer-encoding", "chunked").
		WithBody(&chunkedReader{r: sf.Reader()})

	return &StreamResponse{
		BaseResponse: br,
	}
}
