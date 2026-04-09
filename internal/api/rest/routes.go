package rest

import (
	"github.com/gofiber/fiber/v2"

	"alexandria/api/rest/handlers"
	mw "alexandria/api/rest/middleware"
)

func (s *FiberServer) registerRoutes(
	authH *handlers.AuthHandler,
	bookH *handlers.BookHandler,
	readerH *handlers.ReaderHandler,
	listH *handlers.ListHandler,
	metadataH *handlers.MetadataHandler,
) {
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	s.app.Get("/readyz", func(c *fiber.Ctx) error {
		if s.readinessCheck == nil {
			return c.JSON(fiber.Map{"status": "ready"})
		}
		if err := s.readinessCheck(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	v1 := s.app.Group("/api/v1")

	// Auth — no middleware
	auth := v1.Group("/auth")
	auth.Get("/status", authH.Status)
	auth.Post("/register", authH.Register)
	auth.Post("/login", authH.Login)
	auth.Post("/refresh", authH.Refresh)
	v1.Get("/books/:id/reader/assets/*", readerH.ServeAsset)

	// Protected routes
	protected := v1.Use(mw.JWT(s.authSvc))

	protected.Get("/me", authH.Me)

	books := protected.Group("/books")
	books.Get("/", bookH.List)
	books.Post("/", bookH.Upload)
	// Named sub-routes must come before /:id to avoid being matched as an ID
	books.Get("/authors", bookH.ListAuthors)
	books.Get("/genres", bookH.ListGenres)
	books.Get("/publishers", bookH.ListPublishers)
	books.Get("/years", bookH.ListYears)
	books.Get("/:id", bookH.GetByID)
	books.Put("/:id", bookH.Update)
	books.Delete("/:id", bookH.Delete)
	books.Get("/:id/content", bookH.ServeContent)
	books.Get("/:id/cover", bookH.ServeCover)
	books.Post("/:id/refresh", bookH.RefreshMetadata)
	books.Post("/:id/links", bookH.AddLink)
	books.Put("/:id/links/:linkId", bookH.UpdateLink)
	books.Delete("/:id/links/:linkId", bookH.DeleteLink)
	books.Get("/:id/reader/manifest", readerH.GetManifest)
	books.Get("/:id/reader/sections/:sectionID", readerH.GetSection)
	books.Get("/:id/progress", readerH.GetProgress)
	books.Put("/:id/progress", readerH.SaveProgress)

	lists := protected.Group("/lists")
	lists.Get("/", listH.ListLists)
	lists.Post("/", listH.CreateList)
	lists.Get("/:id", listH.GetList)
	lists.Put("/:id", listH.UpdateList)
	lists.Delete("/:id", listH.DeleteList)
	lists.Get("/:id/books", listH.ListItems)
	lists.Post("/:id/books", listH.AddBook)
	lists.Delete("/:id/books/:bookId", listH.RemoveBook)

	metadataG := protected.Group("/metadata")
	metadataG.Get("/search", metadataH.Search)
}
