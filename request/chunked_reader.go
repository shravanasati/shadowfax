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

	maxChunkSize       int
	maxChunksTotalSize int
}

func newChunkedReader(r io.Reader, sizelimits SizeLimits) *chunkedReader {
	return &chunkedReader{
		reader:   newCRLFReader(r, sizelimits.MaxHeaderLine),
		state:    stateHeader,
		trailers: headers.NewHeaders(),
		maxChunkSize: sizelimits.MaxChunkSize,
		maxChunksTotalSize: sizelimits.MaxBodySize,
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
			if chunkSizeInt < 0 {
				cr.err = ErrInvalidFraming
				return 0, cr.err
			}

			if chunkSizeInt > cr.maxChunkSize {
				cr.err = ErrChunkTooLarge
				return 0, cr.err
			}

			cr.remainInChunk = chunkSizeInt
			if cr.remainInChunk == 0 {
				cr.state = stateTrailers
			} else {
				cr.state = stateData
			}

		case stateData:
			remainingAllowed := cr.maxChunksTotalSize - cr.consumedBytes
			if remainingAllowed <= 0 {
				cr.err = ErrBodyTooLarge
				return 0, cr.err
			}
			toRead := min(cr.remainInChunk, len(p))
			if remainingAllowed < toRead {
				toRead = remainingAllowed
			}
			n, err = cr.reader.GetReader().Read(p[:toRead])
			cr.remainInChunk -= n
			cr.consumedBytes += n
			if cr.consumedBytes > cr.maxChunksTotalSize {
				cr.err = ErrBodyTooLarge
				return n, cr.err
			}
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
	if buf.Len() > cr.maxChunksTotalSize {
		return nil, nil, ErrBodyTooLarge
	}
	return buf, cr.trailers, nil
}
