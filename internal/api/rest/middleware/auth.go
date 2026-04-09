package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"alexandria/internal/app/services"
)

const UserIDKey = "userID"

// JWT returns a Fiber middleware that validates Bearer tokens using AuthService.
func JWT(auth *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header format"})
		}

		userID, err := auth.ValidateAccessToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		c.Locals(UserIDKey, userID)
		return c.Next()
	}
}

// UserID retrieves the authenticated user ID from the Fiber context.
func UserID(c *fiber.Ctx) string {
	return c.Locals(UserIDKey).(string)
}
