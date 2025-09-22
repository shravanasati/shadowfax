package request

import (
	"bytes"
	"errors"
	"io"
	"strconv"
)

type chunkedReader struct {
	reader io.Reader
}

func newChunkedReader(r io.Reader) *chunkedReader {
	return &chunkedReader{reader: r}
}

func parseHexadecimal(hex string) (int, error) {
	n, err := strconv.ParseInt(hex, 16, 64)
	return int(n), err
}

func (cr *chunkedReader) Decode() (*bytes.Buffer, error) {
	buf := bytes.NewBuffer([]byte{})
	crlfReader := newCRLFReader(cr.reader)

	line, err := crlfReader.Read()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	chunkSize, _, _ := bytes.Cut(line, []byte(";"))

	chunkSizeInt, err := parseHexadecimal(string(chunkSize))
	if err != nil {
		return nil, err
	}

	for !crlfReader.Done() && chunkSizeInt > 0 {
		line, err := crlfReader.Read()
		if err != nil {
			return nil, err
		}

		buf.Write(line)

		line, err = crlfReader.Read()
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		chunkSize, _, _ := bytes.Cut(line, []byte(";"))

		chunkSizeInt, err = parseHexadecimal(string(chunkSize))
		if err != nil {
			return nil, err
		}
	}

	// todo read trailers

	return buf, nil
}
