package request

import "errors"

var ErrIncorrectRequestLine = errors.New("incorrect request line")
var ErrIncompleteRequest = errors.New("incomplete request")
var ErrInvalidHeaderValue = errors.New("invalid header value")
var ErrBodyTooLong = errors.New("body length exceeds content-length")
var ErrNotImplemented = errors.New("transfer encoding not implemented")
