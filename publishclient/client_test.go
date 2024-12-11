package publishclient

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadDir(t *testing.T) {
	t.Run("successful upload", func(t *testing.T) {
		dir := t.TempDir()
		content := "content1"
		file1 := filepath.Join(dir, "file1.txt")
		require.NoError(t, os.WriteFile(file1, []byte(content), 0644))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/x-tar", r.Header.Get("Content-Type"))

			tr := tar.NewReader(r.Body)
			var files []string
			var fileContent []byte
			for {
				header, err := tr.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				if !header.FileInfo().IsDir() {
					var buf bytes.Buffer
					_, _ = io.Copy(&buf, tr)
					fileContent = buf.Bytes()
					files = append(files, header.Name)
				}
			}

			assert.ElementsMatch(t, []string{"file1.txt"}, files)
			assert.Equal(t, content, string(fileContent))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := New()
		ctx := context.Background()
		err := client.UploadDir(ctx, server.URL, dir)
		assert.NoError(t, err)
	})

	t.Run("cancelled context", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "file1.txt")
		require.NoError(t, os.WriteFile(file1, []byte("content1"), 0644))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "request cancelled", http.StatusRequestTimeout)
		}))
		defer server.Close()

		client := New()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := client.UploadDir(ctx, server.URL, dir)
		assert.Error(t, err)
	})

	t.Run("server responds with error", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "file1.txt")
		require.NoError(t, os.WriteFile(file1, []byte("content1"), 0644))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer server.Close()

		client := New()
		ctx := context.Background()
		err := client.UploadDir(ctx, server.URL, dir)
		assert.Error(t, err)
	})
}
