package headers

import "errors"

// ErrMalformedHeader is returned when a header line is malformed.
var ErrMalformedHeader = errors.New("malformed header line")

// var ErrHeaderNotFound = errors.New("header key not found")
