package handlers

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"alexandria/internal/api/rest/middleware"
	"alexandria/internal/app/repos"
	"alexandria/internal/app/services"
)

type BookHandler struct {
	books *services.BookService
}

type readCloserStream struct {
	io.Reader
	io.Closer
}

func NewBookHandler(books *services.BookService) *BookHandler {
	return &BookHandler{books: books}
}

// List handles GET /api/v1/books
func (h *BookHandler) List(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	filter := repos.BookFilter{
		Search:    c.Query("search"),
		Author:    c.Query("author"),
		Genre:     c.Query("genre"),
		Publisher: c.Query("publisher"),
		Year:      queryInt(c, "year", 0),
		UserID:    userID,
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Page:      queryInt(c, "page", 1),
		Limit:     queryInt(c, "limit", 20),
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

// ListPublishers handles GET /api/v1/books/publishers
func (h *BookHandler) ListPublishers(c *fiber.Ctx) error {
	publishers, err := h.books.ListPublishers(c.Context())
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"publishers": publishers})
}

// ListYears handles GET /api/v1/books/years
func (h *BookHandler) ListYears(c *fiber.Ctx) error {
	years, err := h.books.ListYears(c.Context())
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"years": years})
}

// ListAuthors handles GET /api/v1/books/authors
func (h *BookHandler) ListAuthors(c *fiber.Ctx) error {
	authors, err := h.books.ListAuthors(c.Context())
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"authors": authors})
}

// ListGenres handles GET /api/v1/books/genres
func (h *BookHandler) ListGenres(c *fiber.Ctx) error {
	genres, err := h.books.ListGenres(c.Context())
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"genres": genres})
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
		FileSize:    file.Size,
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

type updateBookMetadataRequest struct {
	ISBN        *string   `json:"isbn"`
	Publisher   *string   `json:"publisher"`
	PublishedAt *string   `json:"published_at"` // YYYY-MM-DD from <input type="date">
	Language    *string   `json:"language"`
	Genres      *[]string `json:"genres"`
	Tags        *[]string `json:"tags"`
}

type updateBookRequest struct {
	Title          *string                    `json:"title"`
	Author         *string                    `json:"author"`
	Description    *string                    `json:"description"`
	ZealotTicketID *string                    `json:"zealot_ticket_id"`
	Metadata       *updateBookMetadataRequest `json:"metadata"`
	CoverURL       *string                    `json:"cover_url"`
}

// Update handles PUT /api/v1/books/:id
func (h *BookHandler) Update(c *fiber.Ctx) error {
	var req updateBookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	u := services.BookUpdate{
		Title:          req.Title,
		Author:         req.Author,
		Description:    req.Description,
		ZealotTicketID: req.ZealotTicketID,
		CoverURL:       req.CoverURL,
	}
	if req.Metadata != nil {
		u.ISBN = req.Metadata.ISBN
		u.Publisher = req.Metadata.Publisher
		if req.Metadata.PublishedAt != nil && *req.Metadata.PublishedAt != "" {
			t, err := parseDateInput(*req.Metadata.PublishedAt)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "published_at must be YYYY-MM-DD"})
			}
			u.PublishedAt = &t
		}
		u.Language = req.Metadata.Language
		u.Genres = req.Metadata.Genres
		u.Tags = req.Metadata.Tags
	}

	book, err := h.books.Update(c.Context(), c.Params("id"), u)
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

// RefreshMetadata handles POST /api/v1/books/:id/refresh
func (h *BookHandler) RefreshMetadata(c *fiber.Ctx) error {
	book, err := h.books.RefreshMetadata(c.Context(), c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"book": book})
}

// ServeContent handles GET /api/v1/books/:id/content.
// Pass ?inline=true to serve with Content-Disposition: inline (for in-browser viewing).
func (h *BookHandler) ServeContent(c *fiber.Ctx) error {
	rc, book, mtime, err := h.books.OpenFile(c.Context(), c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}

	if !mtime.IsZero() {
		etag := fmt.Sprintf(`"%x"`, mtime.UnixNano())
		c.Set("ETag", etag)
		c.Set("Last-Modified", mtime.UTC().Format(http.TimeFormat))
		c.Set("Cache-Control", "private, max-age=604800")
		if c.Get("If-None-Match") == etag {
			rc.Close()
			return c.SendStatus(fiber.StatusNotModified)
		}
		if ims := c.Get("If-Modified-Since"); ims != "" {
			if t, err := http.ParseTime(ims); err == nil && !mtime.After(t) {
				rc.Close()
				return c.SendStatus(fiber.StatusNotModified)
			}
		}
	}

	contentType := "application/octet-stream"
	switch book.FileType {
	case "epub":
		contentType = "application/epub+zip"
	case "pdf":
		contentType = "application/pdf"
	}

	disposition := "attachment"
	if c.QueryBool("inline") {
		disposition = "inline"
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, filepath.Base(book.FilePath)))
	return c.SendStream(rc)
}

// ServeCover handles GET /api/v1/books/:id/cover
func (h *BookHandler) ServeCover(c *fiber.Ctx) error {
	rc, mtime, err := h.books.OpenCover(c.Context(), c.Params("id"))
	if err != nil {
		return respondErr(c, err)
	}

	if !mtime.IsZero() {
		etag := fmt.Sprintf(`"%x"`, mtime.UnixNano())
		c.Set("ETag", etag)
		c.Set("Last-Modified", mtime.UTC().Format(http.TimeFormat))
		c.Set("Cache-Control", "private, max-age=86400")
		if c.Get("If-None-Match") == etag {
			rc.Close()
			return c.SendStatus(fiber.StatusNotModified)
		}
		if ims := c.Get("If-Modified-Since"); ims != "" {
			if t, err := http.ParseTime(ims); err == nil && !mtime.After(t) {
				rc.Close()
				return c.SendStatus(fiber.StatusNotModified)
			}
		}
	}

	buffered := bufio.NewReader(rc)
	header, err := buffered.Peek(512)
	if err != nil && err != io.EOF {
		rc.Close()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read cover"})
	}
	c.Set("Content-Type", http.DetectContentType(header))
	return c.SendStream(readCloserStream{Reader: buffered, Closer: rc})
}

type linkRequest struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// AddLink handles POST /api/v1/books/:id/links
func (h *BookHandler) AddLink(c *fiber.Ctx) error {
	var req linkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	link, err := h.books.AddLink(c.Context(), c.Params("id"), req.Label, req.URL)
	if err != nil {
		return respondErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"link": link})
}

// UpdateLink handles PUT /api/v1/books/:id/links/:linkId
func (h *BookHandler) UpdateLink(c *fiber.Ctx) error {
	var req linkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	link, err := h.books.UpdateLink(c.Context(), c.Params("id"), c.Params("linkId"), req.Label, req.URL)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"link": link})
}

// DeleteLink handles DELETE /api/v1/books/:id/links/:linkId
func (h *BookHandler) DeleteLink(c *fiber.Ctx) error {
	if err := h.books.DeleteLink(c.Context(), c.Params("id"), c.Params("linkId")); err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

// parseDateInput accepts YYYY-MM-DD (from HTML date inputs) or RFC3339.
func parseDateInput(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date %q", s)
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
