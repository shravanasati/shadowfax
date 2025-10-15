package response

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
)

func detectContentType(filename string, r io.ReadSeeker) string {
    // try by extension
    ext := filepath.Ext(filename)
    if ctype := mime.TypeByExtension(ext); ctype != "" {
        return ctype
    }

    // fallback to sniffing
    buf := make([]byte, 512)
    n, _ := r.Read(buf)
    r.Seek(0, io.SeekStart)
    return http.DetectContentType(buf[:n])
	// using stdlib is kinda cheating here, but I'll let it slide for once :)
}
