package request

import (
	"bytes"
	"io"
)

type crlfReader struct {
	buf    bytes.Buffer
	reader io.Reader
	atEOF  bool
}

func newCRLFReader(r io.Reader) *crlfReader {
	return &crlfReader{reader: r}
}

func (cr *crlfReader) Done() bool {
	return cr.atEOF
}

func (cr *crlfReader) Read() ([]byte, error) {
	if cr.atEOF {
		return nil, io.EOF
	}

	var line []byte
	var foundCR bool

	for {
		// Read one byte at a time into buffer
		b := make([]byte, 1)
		n, err := cr.reader.Read(b)

		if err != nil {
			if err == io.EOF {
				cr.atEOF = true
				// If we have data in the line, return it
				if len(line) > 0 {
					return line, nil
				}
				// flush the buffer
				return cr.buf.Bytes(), io.EOF
			}
			return nil, err
		}

		if n > 0 {
			// Write the byte to buffer
			cr.buf.Write(b)

			// Look strictly for CRLF (\r\n)
			if foundCR && b[0] == '\n' {
				// Found complete CRLF, get the line from buffer (excluding CRLF)
				bufBytes := cr.buf.Bytes()
				// Return everything except the last 2 bytes (\r\n)
				line = make([]byte, len(bufBytes)-2)
				copy(line, bufBytes[:len(bufBytes)-2])
				cr.buf.Reset()
				return line, nil
			}

			if b[0] == '\r' {
				// Found CR, wait for LF
				foundCR = true
			} else {
				foundCR = false
			}
		}
	}
}
