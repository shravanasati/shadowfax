package response

import "errors"

// ErrStatusLineAlreadyWritten is returned when the status line has already been written.
var ErrStatusLineAlreadyWritten = errors.New("status line already written")

// ErrHeadersAlreadyWritten is returned when the headers have already been written.
var ErrHeadersAlreadyWritten = errors.New("headers already written")

// ErrNoBodyState is returned when the response is not in the body state.
var ErrNoBodyState = errors.New("body already written")
