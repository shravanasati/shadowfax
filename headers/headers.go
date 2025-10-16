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

func isValidFieldName(key string) bool {
	return fieldNameRegex.MatchString(key)
}

func validHeaderValueByte(c byte) bool {
	switch {
	case c == 0x09: // HTAB
		return true
	case c == 0x20: // SP
		return true
	case 0x21 <= c && c <= 0x7E: // VCHAR
		return true
	case c >= 0x80: // obs-text
		return true
	}
	return false
}

func isValidFieldValue(val []byte) bool {
	for _, b := range val {
		if !validHeaderValueByte(b) {
			return false
		}
	}
	return true
}

func normalizeKey(key string) string {
	return strings.ToLower(key)
}

// Add adds a new header. If the header already exists, the new value is appended to the existing value, separated by a comma.
func (h *Headers) Add(key, value string) {
	if !isValidFieldName(key) || !isValidFieldValue([]byte(value)) {
		// drop invalid headers to prevent response splitting
		return
	}

	key = normalizeKey(key)
	if existing, ok := h.headers[key]; ok {
		// multiple values
		h.headers[key] = existing + ", " + value
	} else {
		h.headers[key] = value
	}
}

// Get returns the value of a header.
func (h *Headers) Get(key string) string {
	key = normalizeKey(key)
	return h.headers[key]
}

// Remove removes a header.
func (h *Headers) Remove(key string) {
	delete(h.headers, normalizeKey(key))
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

	if !fieldNameRegex.Match(hkey) || !isValidFieldValue(hvalue) {
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
