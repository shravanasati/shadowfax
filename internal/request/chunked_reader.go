package request

import (
	"bytes"
	"errors"
	"io"
	"strconv"

	"github.com/shravanasati/shadowfax/internal/headers"
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

func (cr *chunkedReader) Decode() (*bytes.Buffer, *headers.Headers, error) {
	buf := bytes.NewBuffer([]byte{})
	crlfReader := newCRLFReader(cr.reader)

	// first chunk size
	line, err := crlfReader.Read()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, err
	}

	chunkSize, _, _ := bytes.Cut(line, []byte(";"))
	chunkSizeInt, err := parseHexadecimal(string(chunkSize))
	if err != nil {
		return nil, nil, err
	}

	for chunkSizeInt > 0 {
		// read chunk size bytes
		chunkData := make([]byte, chunkSizeInt)
		_, err := io.ReadFull(cr.reader, chunkData)
		if err != nil {
			return nil, nil, err
		}
		buf.Write(chunkData)

		// consume crlf
		crlfBytes := make([]byte, 2)
		_, err = io.ReadFull(cr.reader, crlfBytes)
		if err != nil {
			return nil, nil, err
		}
		if !bytes.Equal(crlfBytes, []byte("\r\n")) {
			return nil, nil, errors.New("expected CRLF after chunk data")
		}

		// next chunk size
		line, err = crlfReader.Read()
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, err
		}

		chunkSize, _, _ = bytes.Cut(line, []byte(";"))
		chunkSizeInt, err = parseHexadecimal(string(chunkSize))
		if err != nil {
			return nil, nil, err
		}
	}

	// Read trailers using ParseFieldLine with CRLF reader
	trailers := headers.NewHeaders()
	for !crlfReader.Done() {
		line, err := crlfReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, err
		}

		// Empty line indicates end of trailers
		if len(line) == 0 {
			break
		}

		// Parse trailer field line
		err = trailers.ParseFieldLine(line)
		if err != nil {
			return nil, nil, err
		}
	}

	return buf, trailers, nil
}
