package response

import "errors"

var ErrStatusLineAlreadyWritten = errors.New("status line already written")
var ErrHeadersAlreadyWritten = errors.New("headers already written")
var ErrNoBodyState = errors.New("body already written")
