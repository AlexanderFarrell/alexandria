package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"alexandria/app/services"
	"alexandria/api/rest/middleware"
	"alexandria/domain"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user, tokens, err := h.auth.Register(c.Context(), req.Username, req.Password)
	if err != nil {
		return respondErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user, tokens, err := h.auth.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tokens, err := h.auth.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(fiber.Map{"tokens": tokens})
}

// Me handles GET /api/v1/me — returns the current authenticated user.
// (Minimal: returns userID; expand once we add a UserService.GetByID)
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"user_id": middleware.UserID(c)})
}

// respondErr maps domain errors to HTTP status codes.
func respondErr(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = fiber.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		status = fiber.StatusConflict
	case errors.Is(err, domain.ErrUnauthorized):
		status = fiber.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		status = fiber.StatusForbidden
	case errors.Is(err, domain.ErrBadRequest):
		status = fiber.StatusBadRequest
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}
