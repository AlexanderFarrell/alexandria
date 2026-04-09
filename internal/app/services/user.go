package services

import (
	"context"

	"alexandria/app/repos"
	"alexandria/domain"
)

// UserService exposes read-only user lookups needed by API layers.
type UserService struct {
	users repos.UserRepo
}

func NewUserService(users repos.UserRepo) *UserService {
	return &UserService{users: users}
}

func (s *UserService) List(ctx context.Context) ([]*domain.User, error) {
	return s.users.List(ctx)
}

func (s *UserService) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return s.users.GetByUsername(ctx, username)
}
