package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"alexandria/api/rest/middleware"
	"alexandria/app/services"
	"alexandria/domain"
)

type ReaderHandler struct {
	reader *services.ReaderService
}

func NewReaderHandler(reader *services.ReaderService) *ReaderHandler {
	return &ReaderHandler{reader: reader}
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
	CFI        string  `json:"cfi"`
	Percentage float64 `json:"percentage"`
}

// SaveProgress handles PUT /api/v1/books/:id/progress
func (h *ReaderHandler) SaveProgress(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	bookID := c.Params("id")

	var req saveProgressRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	progress, err := h.reader.SaveProgress(c.Context(), userID, bookID, req.CFI, req.Percentage)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"progress": progress})
}
