package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"alexandria/config"
	"alexandria/domain"
)

type memoryUserRepo struct {
	byID       map[string]*domain.User
	byUsername map[string]*domain.User
}

func newMemoryUserRepo(users ...*domain.User) *memoryUserRepo {
	repo := &memoryUserRepo{
		byID:       make(map[string]*domain.User),
		byUsername: make(map[string]*domain.User),
	}
	for _, user := range users {
		_ = repo.Create(context.Background(), user)
	}
	return repo
}

func (r *memoryUserRepo) Create(_ context.Context, user *domain.User) error {
	clone := *user
	r.byID[user.ID] = &clone
	r.byUsername[user.Username] = &clone
	return nil
}

func (r *memoryUserRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

func (r *memoryUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *user
	return &clone, nil
}

func (r *memoryUserRepo) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	user, ok := r.byUsername[username]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *user
	return &clone, nil
}

func (r *memoryUserRepo) Update(_ context.Context, user *domain.User) error {
	clone := *user
	r.byID[user.ID] = &clone
	r.byUsername[user.Username] = &clone
	return nil
}

func (r *memoryUserRepo) Delete(_ context.Context, id string) error {
	user, ok := r.byID[id]
	if !ok {
		return nil
	}
	delete(r.byID, id)
	delete(r.byUsername, user.Username)
	return nil
}

func TestRegisterSingleModeAllowsBootstrapOnly(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := NewAuthService(repo, &config.Config{
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		RegistrationMode: config.RegistrationModeSingle,
	})

	if _, _, err := svc.Register(context.Background(), "owner", "testpass123"); err != nil {
		t.Fatalf("bootstrap registration failed: %v", err)
	}

	if _, _, err := svc.Register(context.Background(), "other", "testpass123"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected second single-mode registration to be forbidden, got %v", err)
	}
}

func TestRegisterDisableModeAlwaysRejects(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := NewAuthService(repo, &config.Config{
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		RegistrationMode: config.RegistrationModeDisable,
	})

	if _, _, err := svc.Register(context.Background(), "owner", "testpass123"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected disabled registration to be forbidden, got %v", err)
	}
}

func TestRegisterMultiModeAllowsMultipleUsers(t *testing.T) {
	repo := newMemoryUserRepo()
	svc := NewAuthService(repo, &config.Config{
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		RegistrationMode: config.RegistrationModeMulti,
	})

	if _, _, err := svc.Register(context.Background(), "reader_one", "testpass123"); err != nil {
		t.Fatalf("first multi registration failed: %v", err)
	}
	if _, _, err := svc.Register(context.Background(), "reader_two", "testpass123"); err != nil {
		t.Fatalf("second multi registration failed: %v", err)
	}
}

func TestValidateStartupRejectsMultipleUsersInSingleMode(t *testing.T) {
	now := time.Now()
	repo := newMemoryUserRepo(
		&domain.User{ID: "u1", Username: "one", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
		&domain.User{ID: "u2", Username: "two", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now},
	)
	svc := NewAuthService(repo, &config.Config{
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		RegistrationMode: config.RegistrationModeSingle,
	})

	if err := svc.ValidateStartup(context.Background()); err == nil {
		t.Fatal("expected startup validation to reject multiple users in single mode")
	}
}
