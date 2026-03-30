package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"alexandria/app/repos"
	"alexandria/domain"
)

// ReaderService manages reading progress for users.
type ReaderService struct {
	progress repos.ProgressRepo
}

func NewReaderService(progress repos.ProgressRepo) *ReaderService {
	return &ReaderService{progress: progress}
}

// GetProgress returns the reading progress for a user/book pair.
// Returns domain.ErrNotFound if the user has not started the book yet.
func (s *ReaderService) GetProgress(ctx context.Context, userID, bookID string) (*domain.ReadingProgress, error) {
	return s.progress.GetByUserAndBook(ctx, userID, bookID)
}

// SaveProgress creates or updates the reading position.
func (s *ReaderService) SaveProgress(ctx context.Context, userID, bookID, cfi string, percentage float64) (*domain.ReadingProgress, error) {
	if percentage < 0 || percentage > 1 {
		return nil, fmt.Errorf("%w: percentage must be between 0 and 1", domain.ErrBadRequest)
	}

	existing, err := s.progress.GetByUserAndBook(ctx, userID, bookID)
	now := time.Now()

	var p *domain.ReadingProgress

	if errors.Is(err, domain.ErrNotFound) {
		p = &domain.ReadingProgress{
			ID:         uuid.NewString(),
			UserID:     userID,
			BookID:     bookID,
			CFI:        cfi,
			Percentage: percentage,
			StartedAt:  now,
			LastReadAt: now,
		}
	} else if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	} else {
		p = existing
		p.CFI = cfi
		p.Percentage = percentage
		p.LastReadAt = now
		if percentage >= 1.0 && p.FinishedAt == nil {
			p.FinishedAt = &now
		}
	}

	if err := s.progress.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("save progress: %w", err)
	}
	return p, nil
}

// ListProgress returns all reading progress records for a user.
func (s *ReaderService) ListProgress(ctx context.Context, userID string) ([]*domain.ReadingProgress, error) {
	return s.progress.ListByUser(ctx, userID)
}
