package request

import (
	"bufio"
	"io"
)

type crlfReader struct {
	reader        *bufio.Reader
	atEOF         bool
	consumedBytes int
	maxLineSize   int
}

func newCRLFReader(r io.Reader, maxLineSize int) *crlfReader {
	// Check if already buffered
	if br, ok := r.(*bufio.Reader); ok {
		return &crlfReader{reader: br, maxLineSize: maxLineSize}
	}
	return &crlfReader{reader: bufio.NewReader(r), maxLineSize: maxLineSize}
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

	// ReadSlice returns a fragment when the buffer is full; we append until newline.
	var line []byte
	for {
		fragment, err := cr.reader.ReadSlice('\n')
		line = append(line, fragment...)
		if cr.maxLineSize > 0 && len(line) > cr.maxLineSize+2 {
			return nil, ErrHeaderLineTooLarge
		}
		if err == nil {
			break
		}
		if err == bufio.ErrBufferFull {
			continue
		}
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
