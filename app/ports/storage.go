package ports

import (
	"context"
	"io"
	"time"
)

// FileInfo holds metadata about a stored file.
type FileInfo struct {
	ModTime time.Time
}

// FileStore abstracts reading and writing book files to storage.
type FileStore interface {
	Save(ctx context.Context, path string, content io.Reader) error
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	Stat(ctx context.Context, path string) (FileInfo, error)
}
