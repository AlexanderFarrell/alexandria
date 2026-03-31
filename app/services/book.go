package services

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"alexandria/app/ports"
	"alexandria/app/repos"
	"alexandria/config"
	"alexandria/domain"
)

// UploadInput carries the file and optional metadata override for a new book.
type UploadInput struct {
	Filename    string
	Content     io.Reader
	FileSize    int64  // byte size of the uploaded file
	Title       string // optional; overrides extracted metadata
	Author      string // optional
	Description string // optional
}

// BookUpdate carries fields that may be updated on an existing book.
type BookUpdate struct {
	Title          *string
	Author         *string
	Description    *string
	ZealotTicketID *string
	// Metadata fields
	ISBN        *string
	Publisher   *string
	PublishedAt *time.Time
	Language    *string
	Genres      *[]string
	Tags        *[]string
	// CoverURL: if set, download this URL and store it as the book's cover image.
	CoverURL *string
}

// BookService handles book library operations.
type BookService struct {
	books   repos.BookRepo
	store   ports.FileStore
	parser  ports.BookParser
	dataDir string
}

func NewBookService(books repos.BookRepo, store ports.FileStore, parser ports.BookParser, cfg *config.Config) *BookService {
	return &BookService{
		books:   books,
		store:   store,
		parser:  parser,
		dataDir: cfg.DataDir,
	}
}

// Upload saves a new book file, parses its metadata, and stores a library record.
func (s *BookService) Upload(ctx context.Context, input UploadInput) (*domain.Book, error) {
	id := uuid.NewString()
	ext := strings.ToLower(filepath.Ext(input.Filename))

	fileType, err := extToFileType(ext)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrBadRequest, err)
	}

	buffered := bufio.NewReader(input.Content)
	header, err := buffered.Peek(8)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("inspect file: %w", err)
	}
	if err := validateFileSignature(fileType, header); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrBadRequest, err)
	}

	// Save the raw file to storage
	filePath := filepath.Join("books", id, "original"+ext)
	if err := s.store.Save(ctx, filePath, buffered); err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	// Parse metadata from the stored file (dispatcher routes by extension)
	absFilePath := filepath.Join(s.dataDir, filePath)
	book, err := s.parser.ParseMetadata(ctx, absFilePath)
	if err != nil || book == nil {
		book = &domain.Book{}
	}
	book.FileType = fileType

	// Apply manual overrides
	if input.Title != "" {
		book.Title = input.Title
	}
	if input.Author != "" {
		book.Author = input.Author
	}
	if input.Description != "" {
		book.Description = input.Description
	}
	if book.Title == "" {
		// Fall back to the filename without extension
		book.Title = strings.TrimSuffix(input.Filename, ext)
	}

	// Extract cover image (dispatcher handles per-format logic)
	coverPath := filepath.Join("books", id, "cover.jpg")
	absCovers := filepath.Join(s.dataDir, coverPath)
	if err := s.parser.ExtractCover(ctx, absFilePath, absCovers); err == nil {
		if exists, _ := s.store.Exists(ctx, coverPath); exists {
			book.CoverPath = coverPath
		}
	}

	now := time.Now()
	book.ID = id
	book.FilePath = filePath
	book.FileSize = input.FileSize
	book.CreatedAt = now
	book.UpdatedAt = now

	if err := s.books.Create(ctx, book); err != nil {
		return nil, fmt.Errorf("create book record: %w", err)
	}

	return book, nil
}

// GetByID returns the book with the given ID.
func (s *BookService) GetByID(ctx context.Context, id string) (*domain.Book, error) {
	return s.books.GetByID(ctx, id)
}

// List returns a paginated, optionally filtered list of books.
func (s *BookService) List(ctx context.Context, filter repos.BookFilter) ([]*domain.Book, int64, error) {
	return s.books.List(ctx, filter)
}

// Update applies partial updates to a book's editable fields.
func (s *BookService) Update(ctx context.Context, id string, u BookUpdate) (*domain.Book, error) {
	book, err := s.books.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if u.Title != nil {
		book.Title = *u.Title
	}
	if u.Author != nil {
		book.Author = *u.Author
	}
	if u.Description != nil {
		book.Description = *u.Description
	}
	if u.ZealotTicketID != nil {
		book.ZealotTicketID = u.ZealotTicketID
	}
	if u.ISBN != nil {
		book.Metadata.ISBN = *u.ISBN
	}
	if u.Publisher != nil {
		book.Metadata.Publisher = *u.Publisher
	}
	if u.PublishedAt != nil {
		book.Metadata.PublishedAt = u.PublishedAt
	}
	if u.Language != nil {
		book.Metadata.Language = *u.Language
	}
	if u.Genres != nil {
		book.Metadata.Genres = *u.Genres
	}
	if u.Tags != nil {
		book.Metadata.Tags = *u.Tags
	}
	if u.CoverURL != nil {
		if err := s.applyCoverFromURL(ctx, book, *u.CoverURL); err != nil {
			log.Printf("apply cover from URL for book %s: %v", book.ID, err)
			// non-fatal — continue with the rest of the update
		}
	}

	book.UpdatedAt = time.Now()

	if err := s.books.Update(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
}

// applyCoverFromURL downloads an image from rawURL and stores it as the book's cover.
func (s *BookService) applyCoverFromURL(ctx context.Context, book *domain.Book, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse cover URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("cover URL scheme must be http or https, got %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" {
		return fmt.Errorf("cover URL host %q is not allowed", host)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build cover request: %w", err)
	}
	hc := &http.Client{Timeout: 15 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("fetch cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetch cover: HTTP %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return fmt.Errorf("cover URL returned non-image content-type %q", ct)
	}

	coverPath := filepath.Join("books", book.ID, "cover.jpg")
	limited := io.LimitReader(resp.Body, 10<<20) // 10 MB cap
	if err := s.store.Save(ctx, coverPath, limited); err != nil {
		return fmt.Errorf("save cover: %w", err)
	}
	book.CoverPath = coverPath
	return nil
}

// Delete removes a book record and its associated files.
func (s *BookService) Delete(ctx context.Context, id string) error {
	if _, err := s.books.GetByID(ctx, id); err != nil {
		return err
	}
	bookDir := filepath.Join("books", id)
	_ = s.store.Delete(ctx, bookDir) // best-effort; record deletion is authoritative
	return s.books.Delete(ctx, id)
}

// OpenFile returns a reader for the book's raw file (epub/pdf) along with the file's modification time.
func (s *BookService) OpenFile(ctx context.Context, id string) (io.ReadCloser, *domain.Book, time.Time, error) {
	book, err := s.books.GetByID(ctx, id)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	rc, err := s.store.Open(ctx, book.FilePath)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, time.Time{}, domain.ErrNotFound
		}
		return nil, nil, time.Time{}, fmt.Errorf("open book file: %w", err)
	}
	fi, err := s.store.Stat(ctx, book.FilePath)
	if err != nil {
		return rc, book, time.Time{}, nil
	}
	return rc, book, fi.ModTime, nil
}

// ListAuthors returns all distinct authors with their book counts.
func (s *BookService) ListAuthors(ctx context.Context) ([]repos.AuthorSummary, error) {
	return s.books.ListAuthors(ctx)
}

// ListGenres returns all distinct genres with their book counts.
func (s *BookService) ListGenres(ctx context.Context) ([]repos.GenreSummary, error) {
	return s.books.ListGenres(ctx)
}

// OpenCover returns a reader for the book's cover image along with the file's modification time.
func (s *BookService) OpenCover(ctx context.Context, id string) (io.ReadCloser, time.Time, error) {
	book, err := s.books.GetByID(ctx, id)
	if err != nil {
		return nil, time.Time{}, err
	}
	if book.CoverPath == "" {
		return nil, time.Time{}, domain.ErrNotFound
	}
	rc, err := s.store.Open(ctx, book.CoverPath)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, time.Time{}, domain.ErrNotFound
		}
		return nil, time.Time{}, fmt.Errorf("open cover: %w", err)
	}
	fi, err := s.store.Stat(ctx, book.CoverPath)
	if err != nil {
		return rc, time.Time{}, nil
	}
	return rc, fi.ModTime, nil
}

// RefreshMetadata re-extracts metadata and cover from the stored file,
// overwriting auto-extractable fields while preserving user-set tags.
func (s *BookService) RefreshMetadata(ctx context.Context, id string) (*domain.Book, error) {
	book, err := s.books.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	absFilePath := filepath.Join(s.dataDir, book.FilePath)
	parsed, err := s.parser.ParseMetadata(ctx, absFilePath)
	if err == nil && parsed != nil {
		if parsed.Title != "" {
			book.Title = parsed.Title
		}
		if parsed.Author != "" {
			book.Author = parsed.Author
		}
		if parsed.Description != "" {
			book.Description = parsed.Description
		}
		if parsed.Metadata.Publisher != "" {
			book.Metadata.Publisher = parsed.Metadata.Publisher
		}
		if parsed.Metadata.Language != "" {
			book.Metadata.Language = parsed.Metadata.Language
		}
		if parsed.Metadata.ISBN != "" {
			book.Metadata.ISBN = parsed.Metadata.ISBN
		}
		if parsed.Metadata.PublishedAt != nil {
			book.Metadata.PublishedAt = parsed.Metadata.PublishedAt
		}
		if len(parsed.Metadata.Genres) > 0 {
			book.Metadata.Genres = parsed.Metadata.Genres
		}
		// Preserve existing user-set tags; only apply parsed tags if none exist yet
		if len(book.Metadata.Tags) == 0 && len(parsed.Metadata.Tags) > 0 {
			book.Metadata.Tags = parsed.Metadata.Tags
		}
	}

	// Re-extract cover (overwrite existing)
	coverPath := filepath.Join("books", id, "cover.jpg")
	absCovers := filepath.Join(s.dataDir, coverPath)
	if err := s.parser.ExtractCover(ctx, absFilePath, absCovers); err == nil {
		if exists, _ := s.store.Exists(ctx, coverPath); exists {
			book.CoverPath = coverPath
		}
	}

	book.UpdatedAt = time.Now()
	if err := s.books.Update(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
}

func extToFileType(ext string) (domain.FileType, error) {
	switch ext {
	case ".epub":
		return domain.FileTypeEPUB, nil
	case ".pdf":
		return domain.FileTypePDF, nil
	default:
		return "", fmt.Errorf("unsupported file type %q; accepted: .epub, .pdf", ext)
	}
}

func validateFileSignature(fileType domain.FileType, header []byte) error {
	switch fileType {
	case domain.FileTypeEPUB:
		if bytes.HasPrefix(header, []byte("PK\x03\x04")) ||
			bytes.HasPrefix(header, []byte("PK\x05\x06")) ||
			bytes.HasPrefix(header, []byte("PK\x07\x08")) {
			return nil
		}
		return fmt.Errorf("file contents do not match the .epub extension")
	case domain.FileTypePDF:
		if bytes.HasPrefix(header, []byte("%PDF-")) {
			return nil
		}
		return fmt.Errorf("file contents do not match the .pdf extension")
	default:
		return nil
	}
}
