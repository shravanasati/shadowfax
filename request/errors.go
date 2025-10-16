package request

import "errors"

// ErrIncorrectRequestLine is returned when the request line is malformed.
var ErrIncorrectRequestLine = errors.New("incorrect request line")

// ErrIncompleteRequest is returned when the request is incomplete.
var ErrIncompleteRequest = errors.New("incomplete request")

// ErrInvalidHeaderValue is returned when a header value is invalid.
var ErrInvalidHeaderValue = errors.New("invalid header value")

// ErrBodyTooLong is returned when the body length exceeds the content-length.
var ErrBodyTooLong = errors.New("body length exceeds content-length")

// ErrNotImplemented is returned when a transfer encoding is not implemented.
var ErrNotImplemented = errors.New("transfer encoding not implemented")
