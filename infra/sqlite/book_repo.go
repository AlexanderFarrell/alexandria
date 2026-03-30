package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	if filter.Genre != "" {
		q = q.Where("meta_genres LIKE ?", "%"+filter.Genre+"%")
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

	var models []BookModel
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
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
