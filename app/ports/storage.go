package ports

import (
	"context"
	"io"
)

// FileStore abstracts reading and writing book files to storage.
type FileStore interface {
	Save(ctx context.Context, path string, content io.Reader) error
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
}
