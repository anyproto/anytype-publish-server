package store

import (
	"bufio"
	"mime"
	"net/http"
	"path/filepath"
)

type File struct {
	Name   string
	Size   int
	Reader *bufio.Reader
}

func (f File) ContentType() string {
	ext := filepath.Ext(f.Name)
	if ext != "" {
		return mime.TypeByExtension(ext)
	}

	// No extension, pick the first bytes for content type
	buf, err := f.Reader.Peek(512)
	if err != nil && err != bufio.ErrBufferFull {
		return "application/octet-stream" // Default fallback
	}

	return http.DetectContentType(buf)
}
