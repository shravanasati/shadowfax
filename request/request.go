package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/shravanasati/shadowfax/headers"
)

// MethodType is the type of an HTTP method.
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

// RequestLine is the first line of an HTTP request.
type RequestLine struct {
	Method      string
	Target      string
	HTTPVersion string
}

// Request is an HTTP request.
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

// RequestFromReader parses an HTTP request from a reader.
// The requests are lazily evaluated, only the request line and headers are parsed.
// The body is parsed when the [Request.Body] method is called.
// Any errors during the body parsing would be returned by the same method.
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

	req := &Request{RequestLine: *requestLine, Headers: *headers, reader: scanner.GetReader(), Query: q}

	err = validateFraming(req)
	if err != nil {
		return nil, err
	}

	return req, nil
}

var nonMergeableHeaders = []string{
	"content-length",
	"host",
	"authorization",
	"proxy-authorization",
	"content-type",
	"retry-after",
	"etag",
	"last-modified",
	"location",
}

func validateFraming(req *Request) error {
	hostVal := req.Headers.Get("host")
	if hostVal == "" {
		return ErrInvalidFraming
	}

	for _, hed := range nonMergeableHeaders {
		hedVal := req.Headers.Get(hed)
		if len(strings.Split(hedVal, ",")) > 1 {
			// more than one non-mergeable headers not allowed
			return ErrInvalidFraming
		}
	}

	if req.Headers.Get("content-length") != "" && req.Headers.Get("transfer-encoding") != "" {
		// requests containing both content length and transfer encoding
		// headers MAY be rejected by the server as per the RFC
		// https://datatracker.ietf.org/doc/html/rfc9112#section-6.1-15
		// we're going to reject it
		return ErrInvalidFraming
	}

	_, err := req.TransferEncodings()
	if err != nil {
		// last transfer encoding must be chunked
		// https://datatracker.ietf.org/doc/html/rfc9112#section-6.3-2.4.3
		return err
	}

	return nil
}

func (r *Request) ContentLength() int64 {
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

func (r *Request) TransferEncodings() ([]string, error) {
	transferEncoding := r.Headers.Get("transfer-encoding")
	if transferEncoding == "" {
		return nil, nil
	}
	encodings := strings.Split(transferEncoding, ",")
	// receiver should decode encodings in reverse
	slices.Reverse(encodings)
	chunked := false
	chunkedPos := 0

	for i, enc := range encodings {
		enc = strings.ToLower(strings.TrimSpace(enc))
		if enc != "chunked" {
			// no other transfer encoding (gzip, deflate, zstd, etc) supported
			return nil, ErrNotImplemented
		} else {
			chunked = true
			chunkedPos = i
		}
	}

	if chunked && chunkedPos != 0 {
		// https://datatracker.ietf.org/doc/html/rfc9112#name-transfer-encoding
		return nil, fmt.Errorf("chunked must be the last transfer encoding")
	}

	if chunked {
		return []string{"chunked"}, nil
	}
	return nil, nil
}

var denyTrailers = []string{
	"content-length",
	"transfer-encoding",
	"trailer",
	"host",
	"connection",
	"proxy-connection",
	"upgrade",
	"keep-alive",
	"authorization",
	"proxy-authorization",
	"content-type",
	"content-encoding",
	"expect",
	"max-forward",
	"te",
}

type chunkedBodyReader struct {
	cr  *chunkedReader
	req *Request
}

func (cbr *chunkedBodyReader) Read(p []byte) (n int, err error) {
	n, err = cbr.cr.Read(p)
	if errors.Is(err, io.EOF) {
		cbr.req.Headers.Set("content-length", strconv.Itoa(cbr.cr.Consumed()))
		// Update headers with trailers
		allowedTrailers := strings.Split(cbr.req.Headers.Get("trailer"), ",")
		for i := range allowedTrailers {
			allowedTrailers[i] = headers.NormalizeKey(allowedTrailers[i])
		}
		for k, v := range cbr.cr.Trailers().All() {
			if slices.Contains(allowedTrailers, k) && !slices.Contains(denyTrailers, k) {
				cbr.req.Headers.Add(k, v)
			}
		}
	}
	return n, err
}

func (cbr *chunkedBodyReader) Close() error {
	return cbr.cr.Close()
}

// Body returns an [io.ReadCloser] for the request body.
// Make sure to close the body after it has been used.
func (r *Request) Body() (io.ReadCloser, error) {
	if br, ok := r.reader.(io.ReadCloser); ok {
		return br, nil
	}

	// check for chunked transfer encoding header first
	tencs, err := r.TransferEncodings()
	if err != nil {
		return nil, err
	}

	if len(tencs) > 0 {
		for _, enc := range tencs {
			switch enc {
			case "chunked":
				cr := newChunkedReader(r.reader)
				cbr := &chunkedBodyReader{cr: cr, req: r}
				r.reader = cbr
				r.Headers.Remove("transfer-encoding")
				return cbr, nil
			default:
				return nil, ErrNotImplemented
			}
		}
	}

	// check for content-length header next
	contentLength := r.ContentLength()
	return newBodyReader(r.reader, int64(contentLength)), nil
}
