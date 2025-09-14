package request

import (
	"bytes"
	"io"
	"regexp"

	"github.com/shravanasati/shadowfax/internal/headers"
)

type MethodType string

const (
	GET     MethodType = "GET"
	HEAD    MethodType = "HEAD"
	POST    MethodType = "POST"
	PUT     MethodType = "PUT"
	PATCH   MethodType = "PATCH"
	DELETE  MethodType = "DELETE"
	TRACE   MethodType = "TRACE"
	OPTIONS MethodType = "OPTIONS"
)

var registeredNurse = []byte("\r\n")
var emptyByteSlice = []byte("")

type RequestLine struct {
	Method      string
	Target      string
	HTTPVersion string
}

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
}

var requestLineRegex = regexp.MustCompile(`^(GET|POST|PUT|PATCH|OPTIONS|TRACE|DELETE|HEAD) ([^\s]*) HTTP\/1.1$`)

func parseRequestLine(reqLine []byte) (*RequestLine, error) {
	matches := requestLineRegex.FindSubmatch(reqLine)
	if matches == nil || len(matches) != 3 {
		return nil, ErrIncorrectRequestLine
	}

	return &RequestLine{
		Method:      string(matches[1]),
		Target:      string(matches[2]),
		HTTPVersion: "1.1",
	}, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	scanner := getCRLFScanner(reader)

	lineCount := 0
	var requestLine *RequestLine
	var err error
	headers := headers.NewHeaders()
	var headersFinished bool

	for scanner.Scan() {
		token := scanner.Bytes()
		lineCount++
		if bytes.Equal(token, emptyByteSlice) {
			// encountered a double CRLF, headers over
			headersFinished = true
			continue
		}
		if lineCount == 1 {
			requestLine, err = parseRequestLine(token)
			if err != nil {
				return nil, err
			}
		} else {
			if !headersFinished {
				err := headers.ParseLine(token)
				if err != nil {
					return nil, err
				}
			} else {
				// parse body
			}
		}
	}

	if !headersFinished {
		return nil, ErrIncompleteRequest
	}

	return &Request{RequestLine: *requestLine, Headers: *headers}, nil
}
