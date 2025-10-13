package response

import "errors"

// ErrInvalidWriterState is returned when the response writer state is not what is called.
var ErrInvalidWriterState = errors.New("invalid writer state")
