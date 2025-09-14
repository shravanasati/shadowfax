package headers

import (
	"bytes"
	"strings"
)

type Headers map[string]string

func (h Headers) Add(key, value string) {
	if existing, ok := h[key]; ok {
		// multiple values
		h[key] = existing + "," + value
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
	hkey := bytes.TrimLeft(data[:colonPos], " ")
	hvalue := bytes.Trim(data[colonPos+1:], " ")
	// todo add characeter validation
	// todo allow \t too

	if !bytes.Equal(hkey, bytes.TrimRight(hkey, " ")) {
		// space between key and colon, invalid
		return ErrMalformedHeader
	}

	h.Add(string(bytes.ToLower(hkey)), string(hvalue))
	return nil
}

func NewHeaders() Headers {
	h := map[string]string{}
	return h
}
