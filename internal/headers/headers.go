package headers

import (
	"bytes"
	"iter"
	"maps"
	"regexp"
	"strings"
)

// https://datatracker.ietf.org/doc/html/rfc9110#name-tokens
var fieldNameRegex = regexp.MustCompile(`^[a-zA-Z0-9!#$%&'*\+\-.^_\x60\|~]+$`)

// Headers represents a collection of HTTP headers.
type Headers struct {
	headers map[string]string
}

// Add adds a new header. If the header already exists, the new value is appended to the existing value, separated by a comma.
func (h *Headers) Add(key, value string) {
	key = strings.ToLower(key)
	if existing, ok := h.headers[key]; ok {
		// multiple values
		h.headers[key] = existing + ", " + value
	} else {
		h.headers[key] = value
	}
}

// Get returns the value of a header.
func (h *Headers) Get(key string) string {
	key = strings.ToLower(key)
	return h.headers[key]
}

// Remove removes a header.
func (h *Headers) Remove(key string) {
	delete(h.headers, key)
}

// All returns an iterator over all headers.
func (h *Headers) All() iter.Seq2[string, string] {
	return maps.All(h.headers)
}

// ParseFieldLine parses a single header line and adds it to the headers.
func (h *Headers) ParseFieldLine(data []byte) (err error) {
	colonPos := bytes.IndexByte(data, ':')
	if colonPos == -1 {
		// colon not found
		return ErrMalformedHeader
	}

	// leading whitespace in header key is allowed
	hkey := bytes.TrimLeft(data[:colonPos], " \t")
	hvalue := bytes.Trim(data[colonPos+1:], " \t")

	if !bytes.Equal(hkey, bytes.TrimRight(hkey, " ")) {
		// space between key and colon, invalid
		return ErrMalformedHeader
	}

	if !fieldNameRegex.Match(hkey) {
		return ErrMalformedHeader
	}

	h.Add(string(hkey), string(hvalue))
	return nil
}

// Size returns the number of headers.
func (h *Headers) Size() int {
	return len(h.headers)
}

// NewHeaders creates a new Headers object.
func NewHeaders() *Headers {
	return &Headers{
		headers: map[string]string{},
	}
}
