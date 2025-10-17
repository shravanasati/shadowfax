package response

import (
	"io"
	"io/fs"
	"strconv"
)

// NamedReadSeeker interface implements Read, Seek, Close, Stat and Name methods.
// It is compatible with [os.File].
// Stat is used for content length detection, while Name, Read and Seek methods are used
// for content type detection.
type NamedReadSeeker interface {
	io.ReadSeeker
	io.Closer
	Stat() (fs.FileInfo, error)
	Name() string
}

// NewFileResponse creates a new file response. It sets the content length
// header if the size of the file is known, otherwise it uses chunked encoding.
func NewFileResponse(f NamedReadSeeker) Response {
	st, err := f.Stat()
	br := NewBaseResponse()
	if err == nil {
		contentLen := strconv.Itoa(int(st.Size()))
		etagVal := prepareEtagValue(st.ModTime().String())
		br.WithHeader("Content-Length", contentLen).
			WithHeader("Content-Type", detectContentType(f.Name(), f)).
			WithHeader("ETag", etagVal).
			WithBody(f)
	} else {
		// fallback to chunked if size unknown
		br.WithHeader("Transfer-Encoding", "chunked")
		br.WithBody(&chunkedReader{r: f})
	}
	return br
}
