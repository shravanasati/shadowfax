package request

// SizeLimits constraints the size of the request. All fields are in bytes.
type SizeLimits struct {
	// Max size of the request line. Defaults to 8KiB.
	MaxRequestLine int

	// Max size of a header line. Defaults to 8KiB.
	MaxHeaderLine int

	// Max size of the headers in total. Defaults to 64KiB.
	MaxHeaders int

	// Max size of a particular chunk in chunked encoding. Defaults to 1MiB.
	MaxChunkSize int

	// Max size of the accumulated body. Defaults to 10MiB.
	MaxBodySize int
}

const kib = 1024
const mib = 1024 * kib

const maxRequestLineBytes = 8 * kib
const maxHeaderLineBytes = 8 * kib
const maxHeadersBytes = 64 * kib
const maxChunkSizeBytes = mib
const maxBodyBytes = 10 * mib

var DefaultSizeLimits = SizeLimits{
	MaxRequestLine: maxRequestLineBytes,
	MaxHeaderLine:  maxHeaderLineBytes,
	MaxHeaders:     maxHeadersBytes,

	MaxChunkSize: maxChunkSizeBytes,
	MaxBodySize:  maxBodyBytes,
}

// Fills empty size limits with defaults.
func fillEmptySizeLimits(sl *SizeLimits) *SizeLimits {
	if sl == nil {
		return &DefaultSizeLimits
	}
	if sl.MaxRequestLine == 0 {
		sl.MaxRequestLine = maxRequestLineBytes
	}
	if sl.MaxHeaderLine == 0 {
		sl.MaxHeaderLine = maxHeaderLineBytes
	}
	if sl.MaxHeaders == 0 {
		sl.MaxHeaders = maxHeadersBytes
	}
	if sl.MaxChunkSize == 0 {
		sl.MaxChunkSize = maxChunkSizeBytes
	}
	if sl.MaxBodySize == 0 {
		sl.MaxBodySize = maxBodyBytes
	}

	return sl
}
