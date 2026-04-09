package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"alexandria/internal/app/repos"
	"alexandria/internal/config"
	"alexandria/internal/domain"
	"alexandria/internal/infra/epub"
)

const readerAssetTokenTTL = time.Hour

type readerAssetClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	BookID string `json:"bid"`
	Kind   string `json:"kind"`
}

// ReaderService manages reader state, EPUB delivery, and reading progress for users.
type ReaderService struct {
	progress repos.ProgressRepo
	books    repos.BookRepo
	dataDir  string
	secret   []byte
}

func NewReaderService(progress repos.ProgressRepo, books repos.BookRepo, cfg *config.Config) *ReaderService {
	return &ReaderService{
		progress: progress,
		books:    books,
		dataDir:  cfg.DataDir,
		secret:   []byte(cfg.JWTSecret),
	}
}

// GetProgress returns the reading progress for a user/book pair.
// Returns domain.ErrNotFound if the user has not started the book yet.
func (s *ReaderService) GetProgress(ctx context.Context, userID, bookID string) (*domain.ReadingProgress, error) {
	return s.progress.GetByUserAndBook(ctx, userID, bookID)
}

// SaveProgress creates or updates the reading position.
// rating is optional (nil = no change); valid values are 1–5.
func (s *ReaderService) SaveProgress(
	ctx context.Context,
	userID string,
	bookID string,
	sectionID string,
	sectionProgress float64,
	blockIndex *int,
	percentage float64,
	rating *int,
) (*domain.ReadingProgress, error) {
	if sectionProgress < 0 || sectionProgress > 1 {
		return nil, fmt.Errorf("%w: section_progress must be between 0 and 1", domain.ErrBadRequest)
	}
	if percentage < 0 || percentage > 1 {
		return nil, fmt.Errorf("%w: percentage must be between 0 and 1", domain.ErrBadRequest)
	}
	if blockIndex != nil && *blockIndex < 0 {
		return nil, fmt.Errorf("%w: block_index must be zero or greater", domain.ErrBadRequest)
	}
	if rating != nil && (*rating < 1 || *rating > 5) {
		return nil, fmt.Errorf("%w: rating must be between 1 and 5", domain.ErrBadRequest)
	}

	existing, err := s.progress.GetByUserAndBook(ctx, userID, bookID)
	now := time.Now()

	var p *domain.ReadingProgress
	if errors.Is(err, domain.ErrNotFound) {
		if sectionID == "" && (rating == nil || sectionProgress != 0 || blockIndex != nil || percentage != 0) {
			return nil, fmt.Errorf("%w: section_id is required", domain.ErrBadRequest)
		}
		p = &domain.ReadingProgress{
			ID:              uuid.NewString(),
			UserID:          userID,
			BookID:          bookID,
			SectionID:       sectionID,
			SectionProgress: sectionProgress,
			BlockIndex:      blockIndex,
			Percentage:      percentage,
			StartedAt:       now,
			LastReadAt:      now,
		}
	} else if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	} else {
		p = existing
		if sectionID != "" {
			p.SectionID = sectionID
			p.SectionProgress = sectionProgress
			p.BlockIndex = blockIndex
			p.Percentage = percentage
		}
		p.LastReadAt = now
		if percentage >= 1.0 && p.FinishedAt == nil {
			p.FinishedAt = &now
		}
	}

	if rating != nil {
		p.Rating = rating
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

// GetManifest returns the normalized reader manifest for an EPUB.
func (s *ReaderService) GetManifest(ctx context.Context, bookID string) (*domain.ReaderManifest, error) {
	_, readerBook, err := s.openReaderBook(ctx, bookID)
	if err != nil {
		return nil, err
	}
	defer readerBook.Close()

	return readerBook.Manifest(), nil
}

// GetSection returns a rewritten EPUB section ready for iframe srcdoc rendering.
func (s *ReaderService) GetSection(ctx context.Context, userID, bookID, sectionID string) (*domain.ReaderSection, error) {
	_, readerBook, err := s.openReaderBook(ctx, bookID)
	if err != nil {
		return nil, err
	}
	defer readerBook.Close()

	assetToken, err := s.issueAssetToken(userID, bookID)
	if err != nil {
		return nil, err
	}

	return readerBook.Section(sectionID, func(assetPath string) string {
		return readerAssetURL(bookID, assetPath, assetToken)
	})
}

// GetAsset returns a rewritten EPUB asset after validating the short-lived reader token.
func (s *ReaderService) GetAsset(ctx context.Context, bookID, assetPath, token string) ([]byte, string, error) {
	claims, err := s.validateAssetToken(token)
	if err != nil {
		return nil, "", err
	}
	if claims.BookID != bookID {
		return nil, "", fmt.Errorf("%w: reader asset token does not match this book", domain.ErrUnauthorized)
	}

	_, readerBook, err := s.openReaderBook(ctx, bookID)
	if err != nil {
		return nil, "", err
	}
	defer readerBook.Close()

	return readerBook.Asset(assetPath, func(innerAssetPath string) string {
		return readerAssetURL(bookID, innerAssetPath, token)
	})
}

func (s *ReaderService) openReaderBook(ctx context.Context, bookID string) (*domain.Book, *epub.ReaderBook, error) {
	book, err := s.books.GetByID(ctx, bookID)
	if err != nil {
		return nil, nil, err
	}
	if book.FileType != domain.FileTypeEPUB {
		return nil, nil, fmt.Errorf("%w: book %q is not an epub", domain.ErrBadRequest, bookID)
	}

	readerBook, err := epub.OpenReaderBook(filepath.Join(s.dataDir, book.FilePath))
	if err != nil {
		return nil, nil, err
	}
	return book, readerBook, nil
}

func (s *ReaderService) issueAssetToken(userID, bookID string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, readerAssetClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   bookID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(readerAssetTokenTTL)),
		},
		UserID: userID,
		BookID: bookID,
		Kind:   "reader_asset",
	})
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign reader asset token: %w", err)
	}
	return signed, nil
}

func (s *ReaderService) validateAssetToken(token string) (*readerAssetClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &readerAssetClaims{}, func(tok *jwt.Token) (interface{}, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("%w: invalid reader asset token", domain.ErrUnauthorized)
	}

	claims, ok := parsed.Claims.(*readerAssetClaims)
	if !ok || claims.Kind != "reader_asset" {
		return nil, fmt.Errorf("%w: invalid reader asset token claims", domain.ErrUnauthorized)
	}
	return claims, nil
}

func readerAssetURL(bookID, assetPath, token string) string {
	assetPath = strings.TrimPrefix(assetPath, "/")
	parts := strings.Split(assetPath, "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}

	return "/api/v1/books/" + url.PathEscape(bookID) + "/reader/assets/" +
		strings.Join(parts, "/") + "?rt=" + url.QueryEscape(token)
}
