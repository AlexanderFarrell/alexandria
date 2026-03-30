package repos

import (
	"context"

	"alexandria/domain"
)

// BookFilter describes optional filtering, sorting, and pagination for book listings.
type BookFilter struct {
	Search    string
	Author    string
	Genre     string
	UserID    string // used for rating-sort JOIN
	SortBy    string // "title" | "author" | "created_at" | "rating" — default "created_at"
	SortOrder string // "asc" | "desc" — default "desc"
	Page      int
	Limit     int
}

// AuthorSummary is a distinct author with a book count.
type AuthorSummary struct {
	Author string `json:"author"`
	Count  int    `json:"count"`
}

// GenreSummary is a distinct genre with a book count.
type GenreSummary struct {
	Genre string `json:"genre"`
	Count int    `json:"count"`
}

// BookRepo is the data-access interface for books.
type BookRepo interface {
	Create(ctx context.Context, book *domain.Book) error
	GetByID(ctx context.Context, id string) (*domain.Book, error)
	List(ctx context.Context, filter BookFilter) ([]*domain.Book, int64, error)
	Update(ctx context.Context, book *domain.Book) error
	Delete(ctx context.Context, id string) error
	ListAuthors(ctx context.Context) ([]AuthorSummary, error)
	ListGenres(ctx context.Context) ([]GenreSummary, error)
}
