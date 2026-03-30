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
) {
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	v1 := s.app.Group("/api/v1")

	// Auth — no middleware
	auth := v1.Group("/auth")
	auth.Post("/register", authH.Register)
	auth.Post("/login", authH.Login)
	auth.Post("/refresh", authH.Refresh)

	// Protected routes
	protected := v1.Use(mw.JWT(s.authSvc))

	protected.Get("/me", authH.Me)

	books := protected.Group("/books")
	books.Get("/", bookH.List)
	books.Post("/", bookH.Upload)
	books.Get("/:id", bookH.GetByID)
	books.Put("/:id", bookH.Update)
	books.Delete("/:id", bookH.Delete)
	books.Get("/:id/content", bookH.ServeContent)
	books.Get("/:id/cover", bookH.ServeCover)
	books.Get("/:id/progress", readerH.GetProgress)
	books.Put("/:id/progress", readerH.SaveProgress)
}
