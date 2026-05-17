package request

import (
	"bytes"
	"errors"
	"io"
	"strconv"

	"github.com/shravanasati/shadowfax/headers"
)

type chunkState int

const (
	stateHeader chunkState = iota
	stateData
	stateCRLF
	stateTrailers
	stateEOF
)

type chunkedReader struct {
	reader        *crlfReader
	remainInChunk int
	consumedBytes int
	state         chunkState
	trailers      *headers.Headers
	err           error
}

func newChunkedReader(r io.Reader) *chunkedReader {
	return &chunkedReader{
		reader:   newCRLFReader(r),
		state:    stateHeader,
		trailers: headers.NewHeaders(),
	}
}

func parseHexadecimal(hex string) (int, error) {
	n, err := strconv.ParseInt(hex, 16, 64)
	return int(n), err
}

func (cr *chunkedReader) Read(p []byte) (n int, err error) {
	if cr.err != nil {
		return 0, cr.err
	}

	for {
		switch cr.state {
		case stateHeader:
			line, err := cr.reader.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					cr.err = ErrIncompleteRequest
					return 0, cr.err
				}
				cr.err = err
				return 0, err
			}

			chunkSize, _, _ := bytes.Cut(line, []byte(";"))
			chunkSizeInt, err := parseHexadecimal(string(chunkSize))
			if err != nil {
				cr.err = err
				return 0, err
			}

			cr.remainInChunk = chunkSizeInt
			if cr.remainInChunk == 0 {
				cr.state = stateTrailers
			} else {
				cr.state = stateData
			}

		case stateData:
			toRead := min(cr.remainInChunk, len(p))
			n, err = cr.reader.GetReader().Read(p[:toRead])
			cr.remainInChunk -= n
			cr.consumedBytes += n
			if cr.remainInChunk == 0 {
				cr.state = stateCRLF
			}
			if n > 0 {
				return n, nil
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					cr.err = ErrIncompleteRequest
					return 0, cr.err
				}
				cr.err = err
				return 0, err
			}

		case stateCRLF:
			crlfBytes := make([]byte, 2)
			_, err = io.ReadFull(cr.reader.GetReader(), crlfBytes)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					cr.err = ErrIncompleteRequest
					return 0, cr.err
				}
				cr.err = err
				return 0, err
			}
			if !bytes.Equal(crlfBytes, []byte("\r\n")) {
				cr.err = errors.New("expected CRLF after chunk data")
				return 0, cr.err
			}
			cr.state = stateHeader

		case stateTrailers:
			for !cr.reader.Done() {
				line, err := cr.reader.Read()
				if err != nil {
					if errors.Is(err, io.EOF) {
						cr.err = ErrIncompleteRequest
						return 0, cr.err
					}
					cr.err = err
					return 0, err
				}

				if len(line) == 0 {
					break
				}

				err = cr.trailers.ParseFieldLine(line)
				if err != nil {
					cr.err = err
					return 0, err
				}
			}
			cr.state = stateEOF

		case stateEOF:
			return 0, io.EOF
		}
	}
}

func (cr *chunkedReader) Trailers() *headers.Headers {
	return cr.trailers
}

// Close implements the io.Closer interface.
func (cr *chunkedReader) Close() error {
	_, err := io.Copy(io.Discard, cr)
	return err
}

func (cr *chunkedReader) Consumed() int {
	return cr.consumedBytes
}

func (cr *chunkedReader) Decode() (*bytes.Buffer, *headers.Headers, error) {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, cr)
	if err != nil {
		return nil, nil, err
	}
	return buf, cr.trailers, nil
}
