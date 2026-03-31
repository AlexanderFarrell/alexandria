package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"alexandria/app/repos"
	"alexandria/domain"
)

type bookRepo struct {
	db *gorm.DB
}

// NewBookRepo returns a BookRepo backed by SQLite.
func NewBookRepo(db *gorm.DB) repos.BookRepo {
	return &bookRepo{db: db}
}

func (r *bookRepo) Create(ctx context.Context, book *domain.Book) error {
	m := toBookModel(book)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create book: %w", err)
	}
	return nil
}

func (r *bookRepo) GetByID(ctx context.Context, id string) (*domain.Book, error) {
	var m BookModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get book: %w", err)
	}
	return fromBookModel(&m), nil
}

func (r *bookRepo) List(ctx context.Context, filter repos.BookFilter) ([]*domain.Book, int64, error) {
	q := r.db.WithContext(ctx).Model(&BookModel{})

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("title LIKE ? OR author LIKE ? OR description LIKE ?", like, like, like)
	}
	if filter.Author != "" {
		q = q.Where("author LIKE ?", "%"+filter.Author+"%")
	}
	// Use json_each for exact genre matching (SQLite 3.38+, which modernc.org/sqlite 1.23+ provides)
	if filter.Genre != "" {
		q = q.Where("EXISTS (SELECT 1 FROM json_each(meta_genres) WHERE value = ?)", filter.Genre)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count books: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Build sort clause from allowlist to prevent injection
	validSortCols := map[string]string{
		"title":      "LOWER(books.title)",
		"author":     "LOWER(books.author)",
		"created_at": "books.created_at",
		"rating":     "rp.rating",
		"file_size":  "books.file_size",
	}
	col, ok := validSortCols[filter.SortBy]
	if !ok {
		col = "books.created_at"
	}
	dir := "DESC"
	if strings.EqualFold(filter.SortOrder, "asc") {
		dir = "ASC"
	}

	// When sorting by rating, LEFT JOIN reading_progress scoped to the requesting user
	if col == "rp.rating" && filter.UserID != "" {
		q = q.Joins("LEFT JOIN reading_progress rp ON rp.book_id = books.id AND rp.user_id = ?", filter.UserID)
	} else if col == "rp.rating" {
		// No user context — fall back to created_at
		col = "books.created_at"
	}

	var models []BookModel
	if err := q.Order(col + " " + dir).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list books: %w", err)
	}

	books := make([]*domain.Book, len(models))
	for i := range models {
		books[i] = fromBookModel(&models[i])
	}
	return books, total, nil
}

func (r *bookRepo) Update(ctx context.Context, book *domain.Book) error {
	m := toBookModel(book)
	m.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("update book: %w", err)
	}
	return nil
}

func (r *bookRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&BookModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("delete book: %w", err)
	}
	return nil
}

// ListAuthors returns distinct authors with book counts, ordered by count descending.
func (r *bookRepo) ListAuthors(ctx context.Context) ([]repos.AuthorSummary, error) {
	var results []repos.AuthorSummary
	err := r.db.WithContext(ctx).Raw(
		"SELECT author, COUNT(*) as count FROM books WHERE author != '' GROUP BY author ORDER BY count DESC",
	).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("list authors: %w", err)
	}
	return results, nil
}

// ListGenres returns distinct genres with book counts using SQLite's json_each virtual table.
// Requires SQLite 3.38+ (modernc.org/sqlite v1.23+ embeds SQLite 3.43).
func (r *bookRepo) ListGenres(ctx context.Context) ([]repos.GenreSummary, error) {
	var results []repos.GenreSummary
	err := r.db.WithContext(ctx).Raw(
		"SELECT value as genre, COUNT(*) as count FROM books, json_each(books.meta_genres) GROUP BY value ORDER BY count DESC",
	).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("list genres: %w", err)
	}
	return results, nil
}

// --- mapping helpers ---

func toBookModel(b *domain.Book) BookModel {
	return BookModel{
		ID:              b.ID,
		Title:           b.Title,
		Author:          b.Author,
		Description:     b.Description,
		CoverPath:       b.CoverPath,
		FilePath:        b.FilePath,
		FileType:        string(b.FileType),
		FileSize:        b.FileSize,
		MetaISBN:        b.Metadata.ISBN,
		MetaPublisher:   b.Metadata.Publisher,
		MetaPublishedAt: b.Metadata.PublishedAt,
		MetaLanguage:    b.Metadata.Language,
		MetaGenres:      marshalStrings(b.Metadata.Genres),
		MetaTags:        marshalStrings(b.Metadata.Tags),
		ZealotTicketID:  b.ZealotTicketID,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}
}

func fromBookModel(m *BookModel) *domain.Book {
	return &domain.Book{
		ID:          m.ID,
		Title:       m.Title,
		Author:      m.Author,
		Description: m.Description,
		CoverPath:   m.CoverPath,
		FilePath:    m.FilePath,
		FileType:    domain.FileType(m.FileType),
		FileSize:    m.FileSize,
		Metadata: domain.BookMetadata{
			ISBN:        m.MetaISBN,
			Publisher:   m.MetaPublisher,
			PublishedAt: m.MetaPublishedAt,
			Language:    m.MetaLanguage,
			Genres:      unmarshalStrings(m.MetaGenres),
			Tags:        unmarshalStrings(m.MetaTags),
		},
		ZealotTicketID: m.ZealotTicketID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func marshalStrings(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ss)
	return string(b)
}

func unmarshalStrings(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var result []string
	_ = json.Unmarshal([]byte(s), &result)
	return result
}
