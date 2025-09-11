package request

import (
	"io"
	"regexp"
)

type MethodType string

const (
	GET     MethodType = "GET"
	HEAD    MethodType = "HEAD"
	POST    MethodType = "POST"
	PUT     MethodType = "PUT"
	PATCH   MethodType = "PATCH"
	DELETE  MethodType = "DELETE"
	CONNECT MethodType = "CONNECT"
	TRACE   MethodType = "TRACE"
	OPTIONS MethodType = "OPTIONS"
)

var registeredNurse = []byte("\r\n")

type RequestLine struct {
	Method      string
	Target      string
	HTTPVersion string
}

type Request struct {
	RequestLine RequestLine
	Headers     map[string]string
	Body        []byte
}

var requestLineRegex = regexp.MustCompile(`^(GET|POST|PUT|PATCH|OPTIONS|TRACE|DELETE|HEAD|CONNECT) ([^\s]*) HTTP\/1.1$`)

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

	for scanner.Scan() {
		token := scanner.Bytes()
		lineCount++
		if lineCount == 1 {
			requestLine, err = parseRequestLine(token)
			if err != nil {
				return nil, err
			}
		}
	}

	return &Request{RequestLine: *requestLine}, nil
}
