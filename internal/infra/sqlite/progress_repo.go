package sqlite

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"alexandria/app/repos"
	"alexandria/domain"
)

type progressRepo struct {
	db *gorm.DB
}

// NewProgressRepo returns a ProgressRepo backed by SQLite.
func NewProgressRepo(db *gorm.DB) repos.ProgressRepo {
	return &progressRepo{db: db}
}

// Save upserts the reading progress record (insert or update on conflict).
func (r *progressRepo) Save(ctx context.Context, p *domain.ReadingProgress) error {
	m := ProgressModel{
		ID:                p.ID,
		UserID:            p.UserID,
		BookID:            p.BookID,
		SectionID:         p.SectionID,
		SectionProgress:   p.SectionProgress,
		BlockIndex:        p.BlockIndex,
		Percentage:        p.Percentage,
		Rating:            p.Rating,
		ZealotProgressRef: p.ZealotProgressRef,
		StartedAt:         p.StartedAt,
		LastReadAt:        p.LastReadAt,
		FinishedAt:        p.FinishedAt,
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "book_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"section_id", "section_progress", "block_index", "percentage", "rating", "zealot_progress_ref", "last_read_at", "finished_at"}),
		}).
		Create(&m).Error
	if err != nil {
		return fmt.Errorf("save progress: %w", err)
	}
	return nil
}

func (r *progressRepo) GetByUserAndBook(ctx context.Context, userID, bookID string) (*domain.ReadingProgress, error) {
	var m ProgressModel
	err := r.db.WithContext(ctx).
		First(&m, "user_id = ? AND book_id = ?", userID, bookID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	}
	return fromProgressModel(&m), nil
}

func (r *progressRepo) ListByUser(ctx context.Context, userID string) ([]*domain.ReadingProgress, error) {
	var models []ProgressModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_read_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list progress: %w", err)
	}
	result := make([]*domain.ReadingProgress, len(models))
	for i := range models {
		result[i] = fromProgressModel(&models[i])
	}
	return result, nil
}

func fromProgressModel(m *ProgressModel) *domain.ReadingProgress {
	return &domain.ReadingProgress{
		ID:                m.ID,
		UserID:            m.UserID,
		BookID:            m.BookID,
		SectionID:         m.SectionID,
		SectionProgress:   m.SectionProgress,
		BlockIndex:        m.BlockIndex,
		Percentage:        m.Percentage,
		Rating:            m.Rating,
		ZealotProgressRef: m.ZealotProgressRef,
		StartedAt:         m.StartedAt,
		LastReadAt:        m.LastReadAt,
		FinishedAt:        m.FinishedAt,
	}
}
