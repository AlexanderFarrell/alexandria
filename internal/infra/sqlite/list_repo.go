package sqlite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"alexandria/internal/app/repos"
	"alexandria/internal/domain"
)

type listRepo struct {
	db *gorm.DB
}

// NewListRepo returns a ListRepo backed by SQLite.
func NewListRepo(db *gorm.DB) repos.ListRepo {
	return &listRepo{db: db}
}

func (r *listRepo) Create(ctx context.Context, list *domain.BookList) error {
	m := BookListModel{
		ID:          list.ID,
		UserID:      list.UserID,
		Name:        list.Name,
		Description: list.Description,
		CreatedAt:   list.CreatedAt,
		UpdatedAt:   list.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create list: %w", err)
	}
	return nil
}

func (r *listRepo) GetByID(ctx context.Context, id string) (*domain.BookList, error) {
	var m BookListModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get list: %w", err)
	}
	return fromListModel(&m), nil
}

func (r *listRepo) ListByUser(ctx context.Context, userID string) ([]*domain.BookList, error) {
	var models []BookListModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list lists: %w", err)
	}
	result := make([]*domain.BookList, len(models))
	for i := range models {
		result[i] = fromListModel(&models[i])
	}
	return result, nil
}

func (r *listRepo) Update(ctx context.Context, list *domain.BookList) error {
	m := BookListModel{
		ID:          list.ID,
		UserID:      list.UserID,
		Name:        list.Name,
		Description: list.Description,
		CreatedAt:   list.CreatedAt,
		UpdatedAt:   list.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("update list: %w", err)
	}
	return nil
}

func (r *listRepo) Delete(ctx context.Context, id string) error {
	// Delete items first, then the list
	if err := r.db.WithContext(ctx).Delete(&BookListItemModel{}, "list_id = ?", id).Error; err != nil {
		return fmt.Errorf("delete list items: %w", err)
	}
	if err := r.db.WithContext(ctx).Delete(&BookListModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("delete list: %w", err)
	}
	return nil
}

func (r *listRepo) AddBook(ctx context.Context, listID, bookID string) error {
	m := BookListItemModel{
		ListID:  listID,
		BookID:  bookID,
		AddedAt: time.Now(),
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&m).Error
	if err != nil {
		return fmt.Errorf("add book to list: %w", err)
	}
	return nil
}

func (r *listRepo) RemoveBook(ctx context.Context, listID, bookID string) error {
	if err := r.db.WithContext(ctx).
		Delete(&BookListItemModel{}, "list_id = ? AND book_id = ?", listID, bookID).Error; err != nil {
		return fmt.Errorf("remove book from list: %w", err)
	}
	return nil
}

func (r *listRepo) GetItems(ctx context.Context, listID string) ([]*domain.BookListItem, error) {
	var models []BookListItemModel
	if err := r.db.WithContext(ctx).
		Where("list_id = ?", listID).
		Order("added_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("get list items: %w", err)
	}
	result := make([]*domain.BookListItem, len(models))
	for i := range models {
		result[i] = &domain.BookListItem{
			ListID:  models[i].ListID,
			BookID:  models[i].BookID,
			AddedAt: models[i].AddedAt,
		}
	}
	return result, nil
}

func fromListModel(m *BookListModel) *domain.BookList {
	return &domain.BookList{
		ID:          m.ID,
		UserID:      m.UserID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
