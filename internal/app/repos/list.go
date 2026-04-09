package repos

import (
	"context"

	"alexandria/internal/domain"
)

// ListRepo is the data-access interface for book lists.
type ListRepo interface {
	Create(ctx context.Context, list *domain.BookList) error
	GetByID(ctx context.Context, id string) (*domain.BookList, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.BookList, error)
	Update(ctx context.Context, list *domain.BookList) error
	Delete(ctx context.Context, id string) error
	AddBook(ctx context.Context, listID, bookID string) error
	RemoveBook(ctx context.Context, listID, bookID string) error
	GetItems(ctx context.Context, listID string) ([]*domain.BookListItem, error)
}
