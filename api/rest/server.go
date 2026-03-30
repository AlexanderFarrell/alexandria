package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"alexandria/api/rest/handlers"
	"alexandria/app"
	"alexandria/app/services"
)

// FiberServer implements apis.Server using the Fiber framework.
type FiberServer struct {
	app     *fiber.App
	authSvc *services.AuthService
}

// New builds and configures the Fiber application.
func New(application *app.App, staticDir string) *FiberServer {
	f := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	f.Use(recover.New())
	f.Use(logger.New())
	f.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	s := &FiberServer{app: f, authSvc: application.Auth}

	authH := handlers.NewAuthHandler(application.Auth)
	bookH := handlers.NewBookHandler(application.Books)
	readerH := handlers.NewReaderHandler(application.Reader)

	s.registerRoutes(authH, bookH, readerH)

	// Serve the compiled Vue app for all non-API routes (SPA fallback)
	if staticDir != "" {
		f.Static("/", staticDir)
		f.Get("*", func(c *fiber.Ctx) error {
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
