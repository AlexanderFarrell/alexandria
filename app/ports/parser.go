package ports

import (
	"context"

	"alexandria/domain"
)

// BookParser extracts structured metadata from a book file.
type BookParser interface {
	// ParseMetadata reads a file at filePath and returns a partially-filled Book.
	// Only metadata fields are populated; ID, FilePath, etc. are set by the caller.
	ParseMetadata(ctx context.Context, filePath string) (*domain.Book, error)

	// ExtractCover copies the cover image from the source file to destPath.
	// Returns nil if no cover is found (non-fatal).
	ExtractCover(ctx context.Context, filePath string, destPath string) error
}
