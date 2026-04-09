package handlers

import (
	"errors"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"alexandria/internal/api/rest/middleware"
	"alexandria/internal/app/services"
	"alexandria/internal/domain"
)

type ReaderHandler struct {
	reader *services.ReaderService
}

func NewReaderHandler(reader *services.ReaderService) *ReaderHandler {
	return &ReaderHandler{reader: reader}
}

// GetManifest handles GET /api/v1/books/:id/reader/manifest
func (h *ReaderHandler) GetManifest(c *fiber.Ctx) error {
	manifest, err := h.reader.GetManifest(c.Context(), c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{"manifest": manifest})
}

// GetSection handles GET /api/v1/books/:id/reader/sections/:sectionID
func (h *ReaderHandler) GetSection(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	section, err := h.reader.GetSection(c.Context(), userID, c.Params("id"), c.Params("sectionID"))
	if err != nil {
		return respondErr(c, err)
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{"section": section})
}

// ServeAsset handles GET /api/v1/books/:id/reader/assets/*
func (h *ReaderHandler) ServeAsset(c *fiber.Ctx) error {
	assetPath, err := url.PathUnescape(c.Params("*"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid asset path"})
	}

	payload, contentType, err := h.reader.GetAsset(c.Context(), c.Params("id"), assetPath, c.Query("rt"))
	if err != nil {
		return respondErr(c, err)
	}

	c.Set("Cache-Control", "no-store")
	c.Set("Content-Type", contentType)
	return c.Send(payload)
}

// GetProgress handles GET /api/v1/books/:id/progress
func (h *ReaderHandler) GetProgress(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	bookID := c.Params("id")

	progress, err := h.reader.GetProgress(c.Context(), userID, bookID)
	if errors.Is(err, domain.ErrNotFound) {
		// Not started yet — return a zero state rather than 404
		return c.JSON(fiber.Map{"progress": nil})
	}
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"progress": progress})
}

type saveProgressRequest struct {
	SectionID       string  `json:"section_id"`
	SectionProgress float64 `json:"section_progress"`
	BlockIndex      *int    `json:"block_index"`
	Percentage      float64 `json:"percentage"`
	Rating          *int    `json:"rating"` // optional; 1–5
}

// SaveProgress handles PUT /api/v1/books/:id/progress
func (h *ReaderHandler) SaveProgress(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	bookID := c.Params("id")

	var req saveProgressRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	progress, err := h.reader.SaveProgress(
		c.Context(),
		userID,
		bookID,
		req.SectionID,
		req.SectionProgress,
		req.BlockIndex,
		req.Percentage,
		req.Rating,
	)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"progress": progress})
}
