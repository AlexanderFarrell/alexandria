package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"alexandria/app/ports"
	"alexandria/domain"
)

type localFileStore struct {
	baseDir string
}

// NewLocalFileStore returns a FileStore that persists files under baseDir.
func NewLocalFileStore(baseDir string) ports.FileStore {
	return &localFileStore{baseDir: baseDir}
}

func (s *localFileStore) Save(_ context.Context, path string, content io.Reader) error {
	full := filepath.Join(s.baseDir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, content); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (s *localFileStore) Open(_ context.Context, path string) (io.ReadCloser, error) {
	full := filepath.Join(s.baseDir, path)
	f, err := os.Open(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (s *localFileStore) Delete(_ context.Context, path string) error {
	full := filepath.Join(s.baseDir, path)
	if err := os.RemoveAll(full); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (s *localFileStore) Exists(_ context.Context, path string) (bool, error) {
	full := filepath.Join(s.baseDir, path)
	_, err := os.Stat(full)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat: %w", err)
	}
	return true, nil
}

func (s *localFileStore) Stat(_ context.Context, path string) (ports.FileInfo, error) {
	full := filepath.Join(s.baseDir, path)
	fi, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return ports.FileInfo{}, domain.ErrNotFound
		}
		return ports.FileInfo{}, fmt.Errorf("stat: %w", err)
	}
	return ports.FileInfo{ModTime: fi.ModTime()}, nil
}
