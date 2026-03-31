package rest

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"alexandria/api/rest/handlers"
	"alexandria/app"
	"alexandria/app/services"
	"alexandria/config"
)

// FiberServer implements apis.Server using the Fiber framework.
type FiberServer struct {
	app            *fiber.App
	authSvc        *services.AuthService
	readinessCheck func() error
}

// New builds and configures the Fiber application.
func New(application *app.App, staticDir string, cfg *config.Config, readinessCheck func() error) *FiberServer {
	f := fiber.New(fiber.Config{
		BodyLimit: cfg.UploadMaxBytes,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "internal server error"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}
			if code >= fiber.StatusInternalServerError {
				log.Printf("unhandled request error: %s %s: %v", c.Method(), c.OriginalURL(), err)
				message = "internal server error"
			}
			return c.Status(code).JSON(fiber.Map{"error": message})
		},
	})

	f.Use(recover.New())
	f.Use(logger.New())
	// Log response body for any error response so the cause appears alongside the status line.
	f.Use(func(c *fiber.Ctx) error {
		chainErr := c.Next()
		if status := c.Response().StatusCode(); status >= 400 {
			log.Printf("error detail: %s %s -> %d: %s",
				c.Method(), c.OriginalURL(), status, c.Response().Body())
		}
		return chainErr
	})
	f.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		return c.Next()
	})
	if len(cfg.CORSAllowOrigins) > 0 {
		f.Use(cors.New(cors.Config{
			AllowOrigins: strings.Join(cfg.CORSAllowOrigins, ","),
			AllowHeaders: "Origin, Content-Type, Accept, Authorization",
			AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
		}))
	}

	s := &FiberServer{
		app:            f,
		authSvc:        application.Auth,
		readinessCheck: readinessCheck,
	}

	authH := handlers.NewAuthHandler(application.Auth)
	bookH := handlers.NewBookHandler(application.Books)
	readerH := handlers.NewReaderHandler(application.Reader)
	listH := handlers.NewListHandler(application.Lists)
	metadataH := handlers.NewMetadataHandler(application.Metadata)

	s.registerRoutes(authH, bookH, readerH, listH, metadataH)

	// Serve the compiled Vue app for all non-API routes (SPA fallback)
	if staticDir != "" {
		f.Static("/", staticDir)
		f.Get("*", func(c *fiber.Ctx) error {
			if strings.HasPrefix(c.Path(), "/api/") || c.Path() == "/health" || c.Path() == "/readyz" {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return c.SendFile(staticDir + "/index.html")
		})
	}

	return s
}

// Start begins listening on addr (e.g. ":8080").
func (s *FiberServer) Start(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server.
func (s *FiberServer) Shutdown() error {
	return s.app.Shutdown()
}
