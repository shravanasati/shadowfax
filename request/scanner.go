package request

import (
	"bufio"
	"io"
)

type crlfReader struct {
	reader *bufio.Reader
	atEOF  bool
}

func newCRLFReader(r io.Reader) *crlfReader {
	// Check if already buffered
	if br, ok := r.(*bufio.Reader); ok {
		return &crlfReader{reader: br}
	}
	return &crlfReader{reader: bufio.NewReader(r)}
}

func (cr *crlfReader) GetReader() io.Reader {
	return cr.reader
}

func (cr *crlfReader) Done() bool {
	return cr.atEOF
}

func (cr *crlfReader) Read() ([]byte, error) {
	if cr.atEOF {
		return nil, io.EOF
	}

	// ReadBytes returns all bytes up to and including the delimiter
	// We read until '\n' and check that the preceding byte is '\r'
	line, err := cr.reader.ReadBytes('\n')

	if err != nil {
		if err == io.EOF {
			cr.atEOF = true
			if len(line) > 0 {
				return line, nil
			}
			return nil, io.EOF
		}
		return nil, err
	}

	// Verify and strip CRLF (\r\n)
	if len(line) >= 2 && line[len(line)-2] == '\r' && line[len(line)-1] == '\n' {
		return line[:len(line)-2], nil
	}

	// If CRLF not found, this is malformed but return as-is for error handling
	if line[len(line)-1] == '\n' {
		return line[:len(line)-1], nil
	}

	return line, nil
}
