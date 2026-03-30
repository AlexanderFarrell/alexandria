package services

import (
	"context"
	"fmt"
	"io"
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

	// Save the raw file to storage
	filePath := filepath.Join("books", id, "original"+ext)
	if err := s.store.Save(ctx, filePath, input.Content); err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	// Parse metadata (EPUB only for now; PDF gets minimal metadata)
	absFilePath := filepath.Join(s.dataDir, filePath)
	var book *domain.Book
	if fileType == domain.FileTypeEPUB {
		parsed, err := s.parser.ParseMetadata(ctx, absFilePath)
		if err != nil {
			// Non-fatal: log and continue with empty metadata
			parsed = &domain.Book{FileType: domain.FileTypeEPUB}
		}
		book = parsed
	} else {
		book = &domain.Book{FileType: fileType}
	}

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

	// Extract cover image
	coverPath := filepath.Join("books", id, "cover.jpg")
	absCovers := filepath.Join(s.dataDir, coverPath)
	if fileType == domain.FileTypeEPUB {
		if err := s.parser.ExtractCover(ctx, absFilePath, absCovers); err == nil {
			book.CoverPath = coverPath
		}
	}

	now := time.Now()
	book.ID = id
	book.FilePath = filePath
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
	book.UpdatedAt = time.Now()

	if err := s.books.Update(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
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

// OpenFile returns a reader for the book's raw file (epub/pdf).
func (s *BookService) OpenFile(ctx context.Context, id string) (io.ReadCloser, *domain.Book, error) {
	book, err := s.books.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	rc, err := s.store.Open(ctx, book.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open book file: %w", err)
	}
	return rc, book, nil
}

// ListAuthors returns all distinct authors with their book counts.
func (s *BookService) ListAuthors(ctx context.Context) ([]repos.AuthorSummary, error) {
	return s.books.ListAuthors(ctx)
}

// ListGenres returns all distinct genres with their book counts.
func (s *BookService) ListGenres(ctx context.Context) ([]repos.GenreSummary, error) {
	return s.books.ListGenres(ctx)
}

// OpenCover returns a reader for the book's cover image.
func (s *BookService) OpenCover(ctx context.Context, id string) (io.ReadCloser, error) {
	book, err := s.books.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if book.CoverPath == "" {
		return nil, domain.ErrNotFound
	}
	return s.store.Open(ctx, book.CoverPath)
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
