package store

import (
	"io"
	"mime"
	"path/filepath"
)

type File struct {
	Name        string
	ContentSize int
	io.Reader
}

func (f File) ContentType() string {
	return mime.TypeByExtension(filepath.Ext(f.Name))
}

func (f File) Len() int {
	return f.ContentSize
}
