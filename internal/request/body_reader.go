package request

import (
	"errors"
	"io"
)

type bodyReader struct {
	reader        io.Reader // will be io.LimitReader
	bytesConsumed int
	contentLength int
}

// Read implements the io.Reader interface.
func (br *bodyReader) Read(p []byte) (int, error) {
	n, err := br.reader.Read(p)
	br.bytesConsumed += n
	
	if errors.Is(err, io.EOF) && br.bytesConsumed < br.contentLength {
		return 0, ErrIncompleteRequest
	}

	return n, err
}

// Close implements the io.Closer interface.
// It discards the unread portion of the body.
func (br *bodyReader) Close() error {
	_, err := io.Copy(io.Discard, br.reader)
	return err
}

func newBodyReader(r io.Reader, contentLength int64) *bodyReader {
	return &bodyReader{reader: io.LimitReader(r, contentLength), contentLength: int(contentLength)}
}
