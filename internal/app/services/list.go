package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"alexandria/app/repos"
	"alexandria/domain"
)

// ListService manages user-curated book lists.
type ListService struct {
	lists repos.ListRepo
}

func NewListService(lists repos.ListRepo) *ListService {
	return &ListService{lists: lists}
}

func (s *ListService) CreateList(ctx context.Context, userID, name, description string) (*domain.BookList, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrBadRequest)
	}
	now := time.Now()
	list := &domain.BookList{
		ID:          uuid.NewString(),
		UserID:      userID,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.lists.Create(ctx, list); err != nil {
		return nil, fmt.Errorf("create list: %w", err)
	}
	return list, nil
}

func (s *ListService) GetList(ctx context.Context, userID, listID string) (*domain.BookList, error) {
	list, err := s.lists.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return list, nil
}

func (s *ListService) ListLists(ctx context.Context, userID string) ([]*domain.BookList, error) {
	return s.lists.ListByUser(ctx, userID)
}

func (s *ListService) UpdateList(ctx context.Context, userID, listID string, name, description *string) (*domain.BookList, error) {
	list, err := s.lists.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if name != nil {
		if *name == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", domain.ErrBadRequest)
		}
		list.Name = *name
	}
	if description != nil {
		list.Description = *description
	}
	list.UpdatedAt = time.Now()
	if err := s.lists.Update(ctx, list); err != nil {
		return nil, fmt.Errorf("update list: %w", err)
	}
	return list, nil
}

func (s *ListService) DeleteList(ctx context.Context, userID, listID string) error {
	list, err := s.lists.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return domain.ErrForbidden
	}
	return s.lists.Delete(ctx, listID)
}

func (s *ListService) AddBook(ctx context.Context, userID, listID, bookID string) error {
	list, err := s.lists.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return domain.ErrForbidden
	}
	return s.lists.AddBook(ctx, listID, bookID)
}

func (s *ListService) RemoveBook(ctx context.Context, userID, listID, bookID string) error {
	list, err := s.lists.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return domain.ErrForbidden
	}
	return s.lists.RemoveBook(ctx, listID, bookID)
}

func (s *ListService) GetItems(ctx context.Context, userID, listID string) ([]*domain.BookListItem, error) {
	list, err := s.lists.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return s.lists.GetItems(ctx, listID)
}
