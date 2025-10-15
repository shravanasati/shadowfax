package response

import (
	"os"
	"strconv"
)

// NewFileResponse creates a new file response. It sets the content length
// header if the size of the file is known, otherwise it uses chunked encoding.
func NewFileResponse(f *os.File) Response {
	st, err := f.Stat()
	br := NewBaseResponse()
	if err == nil {
		contentLen := strconv.Itoa(int(st.Size()))
		etagVal := prepareEtagValue(st.ModTime().String())
		br.WithHeader("Content-Length", contentLen).
			WithHeader("ETag", etagVal).
			WithBody(f)
	} else {
		// fallback to chunked if size unknown
		br.WithHeader("Transfer-Encoding", "chunked")
		br.WithBody(&chunkedReader{r: f})
	}
	return br
}
