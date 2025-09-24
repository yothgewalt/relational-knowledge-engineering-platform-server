package identity

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	service IdentityService
}

func NewAuthMiddleware(service IdentityService) *AuthMiddleware {
	return &AuthMiddleware{
		service: service,
	}
}

func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Authorization header is required",
			})
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Invalid authorization header format",
			})
		}

		token := tokenParts[1]
		response, err := m.service.ValidateToken(c.Context(), token)
		if err != nil || !response.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Invalid or expired token",
			})
		}

		c.Locals("account_id", response.Claims.AccountID)
		c.Locals("email", response.Claims.Email)
		c.Locals("username", response.Claims.Username)

		return c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Next()
		}

		token := tokenParts[1]
		response, err := m.service.ValidateToken(c.Context(), token)
		if err != nil || !response.Valid {
			return c.Next()
		}

		c.Locals("account_id", response.Claims.AccountID)
		c.Locals("email", response.Claims.Email)
		c.Locals("username", response.Claims.Username)

		return c.Next()
	}
}