package identity

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type IdentityHandler struct {
	service IdentityService
}

func NewIdentityHandler(service IdentityService) *IdentityHandler {
	return &IdentityHandler{
		service: service,
	}
}

func (h *IdentityHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	userAgent := c.Get("User-Agent")
	ipAddress := c.IP()

	response, err := h.service.Login(c.Context(), &req, userAgent, ipAddress)
	if err != nil {
		statusCode := fiber.StatusUnauthorized
		if strings.Contains(err.Error(), "inactive") {
			statusCode = fiber.StatusForbidden
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Login failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"data":    response,
	})
}

func (h *IdentityHandler) Logout(c *fiber.Ctx) error {
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
	err := h.service.Logout(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Logout failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Logout successful",
	})
}

func (h *IdentityHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	response, err := h.service.Register(c.Context(), &req)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "already exists") {
			statusCode = fiber.StatusConflict
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Registration failed",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": response.Message,
		"data":    response,
	})
}

func (h *IdentityHandler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	err := h.service.VerifyEmail(c.Context(), &req)
	if err != nil {
		statusCode := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "invalid") {
			statusCode = fiber.StatusUnauthorized
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Email verification failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Email verified successfully",
	})
}

func (h *IdentityHandler) ResendEmailVerification(c *fiber.Ctx) error {
	var req ResendVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	err := h.service.ResendEmailVerification(c.Context(), &req)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = fiber.StatusNotFound
		} else if strings.Contains(err.Error(), "already verified") {
			statusCode = fiber.StatusConflict
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Failed to resend verification",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Verification email sent successfully",
	})
}

func (h *IdentityHandler) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	err := h.service.ForgotPassword(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to send reset email",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "If the email exists, a password reset code has been sent",
	})
}

func (h *IdentityHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	err := h.service.ResetPassword(c.Context(), &req)
	if err != nil {
		statusCode := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "invalid") {
			statusCode = fiber.StatusUnauthorized
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Password reset failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

func (h *IdentityHandler) ChangePassword(c *fiber.Ctx) error {
	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	accountID := c.Locals("account_id").(string)
	if accountID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Unauthorized",
			"message": "Account ID not found in context",
		})
	}

	err := h.service.ChangePassword(c.Context(), accountID, &req)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "incorrect") {
			statusCode = fiber.StatusUnauthorized
		} else if strings.Contains(err.Error(), "inactive") {
			statusCode = fiber.StatusForbidden
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Password change failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

func (h *IdentityHandler) ValidateToken(c *fiber.Ctx) error {
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
	response, err := h.service.ValidateToken(c.Context(), token)
	if err != nil || !response.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Unauthorized",
			"message": "Invalid or expired token",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Token is valid",
		"data": fiber.Map{
			"account_id": response.Claims.AccountID,
			"email":      response.Claims.Email,
			"username":   response.Claims.Username,
		},
	})
}

func (h *IdentityHandler) RefreshToken(c *fiber.Ctx) error {
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
	userAgent := c.Get("User-Agent")
	ipAddress := c.IP()

	response, err := h.service.RefreshToken(c.Context(), token, userAgent, ipAddress)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Token refresh failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Token refreshed successfully",
		"data":    response,
	})
}

