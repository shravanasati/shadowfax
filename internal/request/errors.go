package request

import "errors"

var ErrIncorrectRequestLine = errors.New("incorrect request line")
var ErrIncompleteRequest = errors.New("incomplete request")
