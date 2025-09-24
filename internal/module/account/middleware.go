package account

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AccountMiddleware struct {
	service AccountService
}

func NewAccountMiddleware(service AccountService) *AccountMiddleware {
	return &AccountMiddleware{
		service: service,
	}
}

func (m *AccountMiddleware) ValidateAccountOwnership() fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Locals("account_id")
		if accountID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Authentication required",
			})
		}

		requestedAccountID := c.Params("id")
		if requestedAccountID == "" {
			return c.Next()
		}

		if accountID.(string) != requestedAccountID {
			account, err := m.service.GetAccountByID(c.Context(), accountID.(string))
			if err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "Forbidden",
					"message": "Access denied",
				})
			}

			if requestedAccountID != account.ID {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "Forbidden",
					"message": "You can only access your own account",
				})
			}
		}

		return c.Next()
	}
}

func (m *AccountMiddleware) RequireActiveAccount() fiber.Handler {
	return func(c *fiber.Ctx) error {
		accountID := c.Locals("account_id")
		if accountID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Authentication required",
			})
		}

		account, err := m.service.GetAccountByID(c.Context(), accountID.(string))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Invalid account",
			})
		}

		if !account.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "Forbidden",
				"message": "Account is inactive",
			})
		}

		c.Locals("account", account)
		return c.Next()
	}
}

func (m *AccountMiddleware) RequireAuth() fiber.Handler {
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

func (m *AccountMiddleware) OptionalAuth() fiber.Handler {
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