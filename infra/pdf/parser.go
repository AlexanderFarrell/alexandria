package pdf

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"alexandria/domain"
)

func init() {
	// Disable pdfcpu's config file I/O — avoids permission errors in containers
	// where the process user may not have a writable home directory.
	model.ConfigPath = "disable"
}

// Parser implements ports.BookParser for PDF files.
type Parser struct{}

func New() *Parser { return &Parser{} }

// ParseMetadata reads the PDF Information Dictionary and returns a partial Book.
// Non-fatal: returns empty Book on any error rather than failing the upload.
func (p *Parser) ParseMetadata(_ context.Context, filePath string) (*domain.Book, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return &domain.Book{FileType: domain.FileTypePDF}, nil
	}
	defer f.Close()

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	info, err := pdfapi.PDFInfo(f, filePath, nil, false, conf)
	if err != nil {
		return &domain.Book{FileType: domain.FileTypePDF}, nil
	}

	book := &domain.Book{FileType: domain.FileTypePDF}
	book.Title = strings.TrimSpace(info.Title)
	book.Author = strings.TrimSpace(info.Author)
	book.Description = strings.TrimSpace(info.Subject)

	if len(info.Keywords) > 0 {
		tags := make([]string, 0, len(info.Keywords))
		for _, kw := range info.Keywords {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				tags = append(tags, kw)
			}
		}
		book.Metadata.Tags = tags
	}

	if t := parsePDFDate(info.CreationDate); t != nil {
		book.Metadata.PublishedAt = t
	}

	return book, nil
}

// ExtractCover renders the first PDF page to a JPEG at destPath using pdftoppm.
// Returns nil silently if pdftoppm is not installed or rendering fails.
func (p *Parser) ExtractCover(_ context.Context, filePath string, destPath string) error {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil // pdftoppm not available — skip gracefully
	}

	tmpDir, err := os.MkdirTemp("", "pdf-cover-*")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(tmpDir)

	prefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-f", "1", "-l", "1", "-r", "96", "-png", filePath, prefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil // rendering failed — skip gracefully
	}

	// pdftoppm produces: prefix-1.png or prefix-01.png or prefix-001.png
	var pngPath string
	for _, candidate := range []string{
		prefix + "-1.png",
		prefix + "-01.png",
		prefix + "-001.png",
	} {
		if _, err := os.Stat(candidate); err == nil {
			pngPath = candidate
			break
		}
	}
	if pngPath == "" {
		return nil
	}

	return convertPNGToJPEG(pngPath, destPath)
}

// convertPNGToJPEG decodes a PNG and encodes it as JPEG at destPath.
func convertPNGToJPEG(srcPath, destPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open png: %w", err)
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return fmt.Errorf("decode png: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create cover dir: %w", err)
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create jpeg: %w", err)
	}
	defer dst.Close()

	return jpeg.Encode(dst, img, &jpeg.Options{Quality: 85})
}

// parsePDFDate parses the PDF date format "D:YYYYMMDDHHmmSSOHH'mm'" or plain YYYY.
func parsePDFDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// Strip leading "D:" prefix
	s = strings.TrimPrefix(s, "D:")
	// Normalise timezone: replace trailing apostrophe in HH'mm' pattern
	s = strings.ReplaceAll(s, "'", "")

	formats := []string{
		"20060102150405-0700",
		"20060102150405+0700",
		"20060102150405Z0700",
		"20060102150405",
		"20060102",
		"2006",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
