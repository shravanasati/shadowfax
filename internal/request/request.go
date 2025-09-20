package request

import (
	"bytes"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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

var emptyByteSlice = []byte("")

type RequestLine struct {
	Method      string
	Target      string
	HTTPVersion string
}

type Request struct {
	RequestLine
	Headers    headers.Headers
	PathParams map[string]string
	Query      url.Values
	reader     io.Reader
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

	lineCount := 0
	requestLine := &RequestLine{}
	headers := headers.NewHeaders()
	var headersFinished bool

	scanner := newCRLFReader(reader)

	for !scanner.Done() {
		token, err := scanner.Read()
		if err != nil && err != io.EOF {
			return nil, err
		}
		lineCount++
		// EOF also gives empty byte slice
		// so to differentiate between empty line and EOF
		// we must also take into account the err
		if bytes.Equal(token, emptyByteSlice) && err != io.EOF {
			// encountered a double CRLF, headers over
			headersFinished = true
			break
		}
		if lineCount == 1 {
			requestLine, err = parseRequestLine(token)
			if err != nil {
				return nil, err
			}
		} else {
			// we only parse the headers initially
			// body will be parsed as requested by [Request.Body]
			err := headers.ParseFieldLine(token)
			if err != nil {
				return nil, err
			}
		}
	}

	if !headersFinished {
		return nil, ErrIncompleteRequest
	}

	var query string
	questionMarkPos := strings.IndexRune(requestLine.Target, '?')
	if questionMarkPos != -1 {
		query = requestLine.Target[questionMarkPos+1:]
	}

	q, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	return &Request{RequestLine: *requestLine, Headers: *headers, reader: reader, Query: q}, nil
}

func (r *Request) contentLength() int64 {
	contentLength := r.Headers.Get("content-length")
	if contentLength == "" {
		return 0
	}

	contentLengthInt, err := strconv.Atoi(contentLength)
	if err != nil {
		return 0
	}

	return int64(contentLengthInt)
}

// Returns an [io.ReadCloser] interface. Make sure to close the body after it has been used.
func (r *Request) Body() io.ReadCloser {
	// check for content-length header first
	contentLength := r.contentLength()
	return newBodyReader(r.reader, int64(contentLength))
}
