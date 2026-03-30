package repos

import (
	"context"

	"alexandria/domain"
)

// BookFilter describes optional filtering and pagination for book listings.
type BookFilter struct {
	Search string
	Author string
	Genre  string
	Page   int
	Limit  int
}

// BookRepo is the data-access interface for books.
type BookRepo interface {
	Create(ctx context.Context, book *domain.Book) error
	GetByID(ctx context.Context, id string) (*domain.Book, error)
	List(ctx context.Context, filter BookFilter) ([]*domain.Book, int64, error)
	Update(ctx context.Context, book *domain.Book) error
	Delete(ctx context.Context, id string) error
}
