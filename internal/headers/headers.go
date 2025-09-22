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

type Headers struct {
	headers map[string]string
}

func (h *Headers) Add(key, value string) {
	key = strings.ToLower(key)
	if existing, ok := h.headers[key]; ok {
		// multiple values
		h.headers[key] = existing + ", " + value
	} else {
		h.headers[key] = value
	}
}

func (h *Headers) Get(key string) string {
	key = strings.ToLower(key)
	return h.headers[key]
}

func (h *Headers) Remove(key string) {
	delete(h.headers, key)
}

func (h *Headers) All() iter.Seq2[string, string] {
	return maps.All(h.headers)
}

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

func (h *Headers) Size() int {
	return len(h.headers)
}

func NewHeaders() *Headers {
	return &Headers{
		headers: map[string]string{},
	}
}
