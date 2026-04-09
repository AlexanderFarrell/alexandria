package handlers

import (
	"github.com/gofiber/fiber/v2"

	"alexandria/internal/app/ports"
	"alexandria/internal/app/services"
)

// MetadataHandler exposes online metadata search to the API layer.
type MetadataHandler struct {
	metadata *services.MetadataService
}

func NewMetadataHandler(metadata *services.MetadataService) *MetadataHandler {
	return &MetadataHandler{metadata: metadata}
}

type metadataSourceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Link string `json:"link"`
}

type metadataResultResponse struct {
	Title         string                 `json:"title"`
	Authors       []string               `json:"authors"`
	Description   string                 `json:"description,omitempty"`
	CoverURL      string                 `json:"cover_url,omitempty"`
	Publisher     string                 `json:"publisher,omitempty"`
	PublishedDate string                 `json:"published_date,omitempty"`
	ISBN          string                 `json:"isbn,omitempty"`
	Language      string                 `json:"language,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Source        metadataSourceResponse `json:"source"`
	ExternalURL   string                 `json:"external_url,omitempty"`
}

// Search handles GET /api/v1/metadata/search?q=&title=&author=&isbn=
func (h *MetadataHandler) Search(c *fiber.Ctx) error {
	q := ports.MetadataQuery{
		Q:      c.Query("q"),
		Title:  c.Query("title"),
		Author: c.Query("author"),
		ISBN:   c.Query("isbn"),
	}
	if q.Q == "" && q.Title == "" && q.Author == "" && q.ISBN == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provide at least one of: q, title, author, isbn",
		})
	}

	results, errs, err := h.metadata.Search(c.Context(), q)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "metadata search failed"})
	}

	resp := make([]metadataResultResponse, 0, len(results))
	for _, r := range results {
		authors := r.Authors
		if authors == nil {
			authors = []string{}
		}
		resp = append(resp, metadataResultResponse{
			Title:         r.Title,
			Authors:       authors,
			Description:   r.Description,
			CoverURL:      r.CoverURL,
			Publisher:     r.Publisher,
			PublishedDate: r.PublishedDate,
			ISBN:          r.ISBN,
			Language:      r.Language,
			Tags:          r.Tags,
			ExternalURL:   r.ExternalURL,
			Source: metadataSourceResponse{
				ID:   r.Source.ID,
				Name: r.Source.Name,
				Link: r.Source.Link,
			},
		})
	}

	if errs == nil {
		errs = []services.ProviderError{}
	}

	return c.JSON(fiber.Map{
		"results": resp,
		"errors":  errs,
	})
}
