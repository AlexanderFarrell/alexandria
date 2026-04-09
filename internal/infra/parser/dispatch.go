package parser

import (
	"context"
	"path/filepath"
	"strings"

	"alexandria/app/ports"
	"alexandria/domain"
)

// DispatchParser routes ParseMetadata and ExtractCover calls to the appropriate
// per-format parser based on the file extension.
type DispatchParser struct {
	parsers map[domain.FileType]ports.BookParser
}

// New creates a DispatchParser with the provided format-specific parsers.
func New(parsers map[domain.FileType]ports.BookParser) *DispatchParser {
	return &DispatchParser{parsers: parsers}
}

func (d *DispatchParser) ParseMetadata(ctx context.Context, filePath string) (*domain.Book, error) {
	if p := d.parserFor(filePath); p != nil {
		return p.ParseMetadata(ctx, filePath)
	}
	return &domain.Book{}, nil
}

func (d *DispatchParser) ExtractCover(ctx context.Context, filePath string, destPath string) error {
	if p := d.parserFor(filePath); p != nil {
		return p.ExtractCover(ctx, filePath, destPath)
	}
	return nil
}

func (d *DispatchParser) parserFor(filePath string) ports.BookParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".epub":
		return d.parsers[domain.FileTypeEPUB]
	case ".pdf":
		return d.parsers[domain.FileTypePDF]
	}
	return nil
}
