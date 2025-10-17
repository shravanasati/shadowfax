package middleware

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/server"
)

// NamedReadSeekerFS is a custom FS interface which returns [response.NamedReadSeeker].
// It abstracts filesystem operations for serving static files from different sources.
type NamedReadSeekerFS interface {
	// Open opens a file by name and returns a NamedReadSeeker that can read and seek within the file.
	Open(name string) (response.NamedReadSeeker, error)
}

// DirFS abstracts directory filesystem and implements the NamedReadSeekerFS interface.
// It serves files from a specified root directory on the filesystem.
type DirFS struct {
	root string
}

// NewDirFS creates a new DirFS instance that serves files from the given root directory.
func NewDirFS(root string) *DirFS {
	return &DirFS{root: root}
}

func (d *DirFS) Open(name string) (response.NamedReadSeeker, error) {
	f, err := os.Open(filepath.Join(d.root, name))
	if err != nil {
		return nil, err
	}
	return f, nil
}

// EmbedFS implements the NamedReadSeekerFS interface for embedded filesystems.
// It serves files from Go's embed.FS, allowing static files to be embedded in the binary.
type EmbedFS struct {
	fsys embed.FS
}

// NewEmbedFS creates a new EmbedFS instance wrapping the given embedded filesystem.
func NewEmbedFS(fsys embed.FS) *EmbedFS {
	return &EmbedFS{fsys: fsys}
}

func (e *EmbedFS) Open(name string) (response.NamedReadSeeker, error) {
	f, err := e.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return &embedFile{
		name: name,
		data: bytes.NewReader(data),
		info: info,
	}, nil
}

// embedFile implements response.NamedReadSeeker for files within [embed.FS].
type embedFile struct {
	name string
	data io.ReadSeeker
	info fs.FileInfo
}

func (f *embedFile) Read(p []byte) (int, error)         { return f.data.Read(p) }
func (f *embedFile) Seek(o int64, w int) (int64, error) { return f.data.Seek(o, w) }
func (f *embedFile) Close() error                       { return nil }
func (f *embedFile) Stat() (fs.FileInfo, error)         { return f.info, nil }
func (f *embedFile) Name() string                       { return f.name }

// NewStaticHandler creates a middleware handler for serving static files.
// It takes a wildcard parameter name (from URL routing) and a filesystem implementation.
// The middleware serves files from the filesystem, with automatic index.html serving for directories.
// For security, it prevents directory traversal attacks using ".." and rejects absolute paths.
// If a requested file is not found, it passes control to the next handler in the chain.
func NewStaticHandler(wildcardParam string, fsys NamedReadSeekerFS) server.Handler {
	notFoundResp := response.NewTextResponse("File Not Found").WithStatusCode(response.StatusNotFound)

	return func(r *request.Request) response.Response {
		reqFilePath := r.PathParams[wildcardParam]

		// Security: clean the path and reject if it contains ".." or is absolute.
		cleanedPath := path.Clean(reqFilePath)
		if strings.Contains(cleanedPath, "..") || path.IsAbs(cleanedPath) {
			return response.NewTextResponse("Not Found").
				WithStatusCode(response.StatusNotFound)
		}

		// The path from the request might start with a `/`. `path.Clean` doesn't remove a leading `/`.
		// But the file system paths are relative. So we should trim it.
		if len(cleanedPath) > 0 && cleanedPath[0] == '/' {
			cleanedPath = cleanedPath[1:]
		}

		// Try to open the file.
		f, err := fsys.Open(cleanedPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// File does not exist, pass to next handler.
				return notFoundResp
			}
			// Other error, so return internal server error.
			return response.NewTextResponse("Internal Server Error").
				WithStatusCode(response.StatusInternalServerError)
		}

		stat, err := f.Stat()
		if err != nil {
			f.Close()
			return response.NewTextResponse("Internal Server Error").
				WithStatusCode(response.StatusInternalServerError)
		}

		if stat.IsDir() {
			f.Close()
			// If it's a directory, try to serve index.html
			indexPath := path.Join(cleanedPath, "index.html")
			indexFile, err := fsys.Open(indexPath)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					// index.html does not exist, pass to next handler.
					return notFoundResp
				}
				return response.NewTextResponse("Internal Server Error").
					WithStatusCode(response.StatusInternalServerError)
			}

			// It's a directory, but we are serving index.html, so it's a file response.
			return response.NewFileResponse(indexFile)
		}

		// It's a file, serve it.
		return response.NewFileResponse(f)
	}
}
