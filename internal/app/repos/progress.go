package repos

import (
	"context"

	"alexandria/internal/domain"
)

// ProgressRepo is the data-access interface for reading progress.
type ProgressRepo interface {
	Save(ctx context.Context, progress *domain.ReadingProgress) error
	GetByUserAndBook(ctx context.Context, userID, bookID string) (*domain.ReadingProgress, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.ReadingProgress, error)
}
