package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"alexandria/app/repos"
	"alexandria/config"
	"alexandria/domain"
)

// Tokens holds the JWT pair returned after a successful auth operation.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds until access token expires
}

type accessClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
}

type refreshClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	Kind   string `json:"kind"`
}

// AuthService handles registration, login, and token lifecycle.
type AuthService struct {
	users  repos.UserRepo
	secret []byte
	access time.Duration
	refresh time.Duration
}

func NewAuthService(users repos.UserRepo, cfg *config.Config) *AuthService {
	return &AuthService{
		users:   users,
		secret:  []byte(cfg.JWTSecret),
		access:  cfg.JWTExpiry,
		refresh: cfg.RefreshExpiry,
	}
}

// Register creates a new user account and returns tokens.
func (s *AuthService) Register(ctx context.Context, username, password string) (*domain.User, *Tokens, error) {
	if username == "" || password == "" {
		return nil, nil, fmt.Errorf("%w: username and password are required", domain.ErrBadRequest)
	}
	if len(password) < 8 {
		return nil, nil, fmt.Errorf("%w: password must be at least 8 characters", domain.ErrBadRequest)
	}

	_, err := s.users.GetByUsername(ctx, username)
	if err == nil {
		return nil, nil, fmt.Errorf("%w: username already taken", domain.ErrConflict)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, fmt.Errorf("check username: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	tokens, err := s.issueTokens(user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

// Login validates credentials and returns tokens.
func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.User, *Tokens, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil, fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
	}

	tokens, err := s.issueTokens(user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

// RefreshToken validates a refresh token and returns a new token pair.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*Tokens, error) {
	token, err := jwt.ParseWithClaims(refreshToken, &refreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: invalid refresh token", domain.ErrUnauthorized)
	}

	claims, ok := token.Claims.(*refreshClaims)
	if !ok || claims.Kind != "refresh" {
		return nil, fmt.Errorf("%w: not a refresh token", domain.ErrUnauthorized)
	}

	// Verify user still exists
	if _, err := s.users.GetByID(ctx, claims.UserID); err != nil {
		return nil, fmt.Errorf("%w: user no longer exists", domain.ErrUnauthorized)
	}

	return s.issueTokens(claims.UserID)
}

// ValidateAccessToken parses an access token and returns the user ID.
func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("%w: invalid token", domain.ErrUnauthorized)
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok {
		return "", fmt.Errorf("%w: invalid claims", domain.ErrUnauthorized)
	}
	return claims.UserID, nil
}

func (s *AuthService) issueTokens(userID string) (*Tokens, error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.access)),
		},
		UserID: userID,
	})
	accessStr, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refresh)),
		},
		UserID: userID,
		Kind:   "refresh",
	})
	refreshStr, err := refreshToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &Tokens{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int64(s.access.Seconds()),
	}, nil
}
