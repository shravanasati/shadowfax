package response

import (
	"os"
	"strconv"
)

func NewFileResponse(f *os.File) Response {
	st, err := f.Stat()
	br := NewBaseResponse()
	if err == nil {
		br.WithHeader("Content-Length", strconv.Itoa(int(st.Size())))
		br.WithBody(f)
	} else {
		// fallback to chunked if size unknown
		br.WithHeader("Transfer-Encoding", "chunked")
		br.WithBody(&chunkedReader{r: f})
	}
	return br
}
