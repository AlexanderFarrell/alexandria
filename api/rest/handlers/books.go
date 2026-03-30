package handlers

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"alexandria/api/rest/middleware"
	"alexandria/app/repos"
	"alexandria/app/services"
)

type BookHandler struct {
	books *services.BookService
}

func NewBookHandler(books *services.BookService) *BookHandler {
	return &BookHandler{books: books}
}

// List handles GET /api/v1/books
func (h *BookHandler) List(c *fiber.Ctx) error {
	filter := repos.BookFilter{
		Search: c.Query("search"),
		Author: c.Query("author"),
		Genre:  c.Query("genre"),
		Page:   queryInt(c, "page", 1),
		Limit:  queryInt(c, "limit", 20),
	}

	books, total, err := h.books.List(c.Context(), filter)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{
		"books": books,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}

// Upload handles POST /api/v1/books (multipart form)
func (h *BookHandler) Upload(c *fiber.Ctx) error {
	_ = middleware.UserID(c) // authenticated

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file field is required"})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open uploaded file"})
	}
	defer f.Close()

	input := services.UploadInput{
		Filename:    filepath.Base(file.Filename),
		Content:     f,
		Title:       c.FormValue("title"),
		Author:      c.FormValue("author"),
		Description: c.FormValue("description"),
	}

	book, err := h.books.Upload(c.Context(), input)
	if err != nil {
		return respondErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"book": book})
}

// GetByID handles GET /api/v1/books/:id
func (h *BookHandler) GetByID(c *fiber.Ctx) error {
	book, err := h.books.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"book": book})
}

type updateBookRequest struct {
	Title          *string `json:"title"`
	Author         *string `json:"author"`
	Description    *string `json:"description"`
	ZealotTicketID *string `json:"zealot_ticket_id"`
}

// Update handles PUT /api/v1/books/:id
func (h *BookHandler) Update(c *fiber.Ctx) error {
	var req updateBookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	book, err := h.books.Update(c.Context(), c.Params("id"), services.BookUpdate{
		Title:          req.Title,
		Author:         req.Author,
		Description:    req.Description,
		ZealotTicketID: req.ZealotTicketID,
	})
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"book": book})
}

// Delete handles DELETE /api/v1/books/:id
func (h *BookHandler) Delete(c *fiber.Ctx) error {
	if err := h.books.Delete(c.Context(), c.Params("id")); err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

// ServeContent handles GET /api/v1/books/:id/content
// Reads the book file into memory then sends it — avoids the defer-close/lazy-read race in Fiber.
func (h *BookHandler) ServeContent(c *fiber.Ctx) error {
	rc, book, err := h.books.OpenFile(c.Context(), c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}

	contentType := "application/octet-stream"
	switch book.FileType {
	case "epub":
		contentType = "application/epub+zip"
	case "pdf":
		contentType = "application/pdf"
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(book.FilePath)))
	return c.Send(data)
}

// ServeCover handles GET /api/v1/books/:id/cover
func (h *BookHandler) ServeCover(c *fiber.Ctx) error {
	rc, err := h.books.OpenCover(c.Context(), c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read cover"})
	}
	c.Set("Content-Type", "image/jpeg")
	return c.Send(data)
}

func queryInt(c *fiber.Ctx, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
