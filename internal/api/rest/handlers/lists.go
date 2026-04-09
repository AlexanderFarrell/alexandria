package handlers

import (
	"github.com/gofiber/fiber/v2"

	"alexandria/internal/api/rest/middleware"
	"alexandria/internal/app/services"
)

type ListHandler struct {
	lists *services.ListService
}

func NewListHandler(lists *services.ListService) *ListHandler {
	return &ListHandler{lists: lists}
}

type createListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateList handles POST /api/v1/lists
func (h *ListHandler) CreateList(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	var req createListRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	list, err := h.lists.CreateList(c.Context(), userID, req.Name, req.Description)
	if err != nil {
		return respondErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"list": list})
}

// GetList handles GET /api/v1/lists/:id
func (h *ListHandler) GetList(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	list, err := h.lists.GetList(c.Context(), userID, c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"list": list})
}

// ListLists handles GET /api/v1/lists
func (h *ListHandler) ListLists(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	lists, err := h.lists.ListLists(c.Context(), userID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"lists": lists})
}

type updateListRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// UpdateList handles PUT /api/v1/lists/:id
func (h *ListHandler) UpdateList(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	var req updateListRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	list, err := h.lists.UpdateList(c.Context(), userID, c.Params("id"), req.Name, req.Description)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"list": list})
}

// DeleteList handles DELETE /api/v1/lists/:id
func (h *ListHandler) DeleteList(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	if err := h.lists.DeleteList(c.Context(), userID, c.Params("id")); err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

type addBookRequest struct {
	BookID string `json:"book_id"`
}

// AddBook handles POST /api/v1/lists/:id/books
func (h *ListHandler) AddBook(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	var req addBookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := h.lists.AddBook(c.Context(), userID, c.Params("id"), req.BookID); err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"message": "added"})
}

// RemoveBook handles DELETE /api/v1/lists/:id/books/:bookId
func (h *ListHandler) RemoveBook(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	if err := h.lists.RemoveBook(c.Context(), userID, c.Params("id"), c.Params("bookId")); err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"message": "removed"})
}

// ListItems handles GET /api/v1/lists/:id/books
func (h *ListHandler) ListItems(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	items, err := h.lists.GetItems(c.Context(), userID, c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}
