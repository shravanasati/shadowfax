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

// ErrInvalidFraming is returned when the request fails the RFC standards.
var ErrInvalidFraming = errors.New("invalid request framing")

// ErrRequestLineTooLarge is returned when the request line exceeds size limits.
var ErrRequestLineTooLarge = errors.New("request line size exceeded configured limits")

// ErrHeaderLineTooLarge is returned when a header line exceeds size limits.
var ErrHeaderLineTooLarge = errors.New("header line size exceeded configured limits")

// ErrHeadersTooLarge is returned when total header size exceeds limits.
var ErrHeadersTooLarge = errors.New("total headers size exceeded configured limits")

// ErrChunkTooLarge is returned when a chunk size exceeds limits.
var ErrChunkTooLarge = errors.New("chunk size exceeded configured limits")

// ErrBodyTooLarge is returned when the body exceeds configured limits.
var ErrBodyTooLarge = errors.New("body exceeded configured limits")
