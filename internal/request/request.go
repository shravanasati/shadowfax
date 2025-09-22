package request

import (
	"bytes"
	"io"
	"net/url"
	"regexp"
	"slices"
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

// Parses the HTTP request from the reader. The requests are lazily evaluated,
// only the request line and headers are parsed. The body is parsed when the
// [Request.Body] method is called. Any errors during the body parsing would
// be returned by the same method.
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

func (r *Request) transferEncodings() ([]string, error) {
	transferEncoding := r.Headers.Get("transfer-encoding")
	encodings := strings.Split(transferEncoding, ",")
	// receiver should decode encodings in reverse
	slices.Reverse(encodings)
	chunked := false

	for _, enc := range encodings {
		enc = strings.ToLower(strings.TrimSpace(enc))
		if enc != "chunked" {
			// no other transfer encoding (gzip, deflate, zstd, etc) supported
			return nil, ErrNotImplemented
		} else {
			chunked = true
		}
	}

	if chunked {
		return []string{"chunked"}, nil
	}
	return nil, nil
}

// Returns an [io.ReadCloser] interface. Make sure to close the body after it has been used.
func (r *Request) Body() (io.ReadCloser, error) {
	// check for chunked transfer encoding header first
	tencs, err := r.transferEncodings()
	if err != nil {
		return nil, err
	}

	if len(tencs) > 0 {
		for _, enc := range tencs {
			switch enc {
			case "chunked":
				cr := newChunkedReader(r.reader)
				cr.Ra
			default:
				return nil, ErrNotImplemented
			}
		}
		r.Headers.Remove("transfer-encoding")
	}

	// check for content-length header next
	contentLength := r.contentLength()
	return newBodyReader(r.reader, int64(contentLength)), nil
}
