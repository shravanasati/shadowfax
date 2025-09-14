package headers

import (
	"bytes"
	"regexp"
	"strings"
)

// https://datatracker.ietf.org/doc/html/rfc9110#name-tokens
var fieldNameRegex = regexp.MustCompile(`^[a-zA-Z0-9!#$%&'*\+\-.^_\x60\|~]+$`)

type Headers map[string]string

func (h Headers) Add(key, value string) {
	key = strings.ToLower(key)
	if existing, ok := h[key]; ok {
		// multiple values
		h[key] = existing + ", " + value
	} else {
		h[key] = value
	}
}

func (h Headers) Get(key string) (string, error) {
	key = strings.ToLower(key)
	existing, ok := h[key]
	if !ok {
		return "", ErrHeaderNotFound
	}
	return existing, nil
}

func (h Headers) ParseLine(data []byte) (err error) {
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

func NewHeaders() Headers {
	return map[string]string{}
}
