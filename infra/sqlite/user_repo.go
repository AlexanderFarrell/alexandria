package sqlite

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"alexandria/app/repos"
	"alexandria/domain"
)

type userRepo struct {
	db *gorm.DB
}

// NewUserRepo returns a UserRepo backed by SQLite.
func NewUserRepo(db *gorm.DB) repos.UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	m := UserModel{
		ID:           user.ID,
		Username:     user.Username,
		Email:        nullableEmail(user.Email),
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *userRepo) List(ctx context.Context) ([]*domain.User, error) {
	var models []UserModel
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]*domain.User, len(models))
	for index := range models {
		users[index] = fromUserModel(&models[index])
	}
	return users, nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var m UserModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return fromUserModel(&m), nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var m UserModel
	err := r.db.WithContext(ctx).First(&m, "username = ?", username).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return fromUserModel(&m), nil
}

func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	m := UserModel{
		ID:           user.ID,
		Username:     user.Username,
		Email:        nullableEmail(user.Email),
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *userRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&UserModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func nullableEmail(email string) *string {
	if email == "" {
		return nil
	}
	return &email
}

func fromUserModel(m *UserModel) *domain.User {
	email := ""
	if m.Email != nil {
		email = *m.Email
	}
	return &domain.User{
		ID:           m.ID,
		Username:     m.Username,
		Email:        email,
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
