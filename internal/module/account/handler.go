package account

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AccountHandler struct {
	service AccountService
}

func NewAccountHandler(service AccountService) *AccountHandler {
	return &AccountHandler{
		service: service,
	}
}

func (h *AccountHandler) CreateAccount(c *fiber.Ctx) error {
	var req CreateAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	account, err := h.service.CreateAccount(c.Context(), &req)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "already exists") {
			statusCode = fiber.StatusConflict
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Failed to create account",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Account created successfully",
		"data":    account,
	})
}

func (h *AccountHandler) GetAccount(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request",
			"message": "Account ID is required",
		})
	}

	account, err := h.service.GetAccountByID(c.Context(), id)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid") {
			statusCode = fiber.StatusNotFound
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Failed to get account",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account retrieved successfully",
		"data":    account,
	})
}

func (h *AccountHandler) GetAccountByEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request",
			"message": "Email parameter is required",
		})
	}

	account, err := h.service.GetAccountByEmail(c.Context(), email)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = fiber.StatusNotFound
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Failed to get account",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account retrieved successfully",
		"data":    account,
	})
}

func (h *AccountHandler) GetAccountByUsername(c *fiber.Ctx) error {
	username := c.Query("username")
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request",
			"message": "Username parameter is required",
		})
	}

	account, err := h.service.GetAccountByUsername(c.Context(), username)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = fiber.StatusNotFound
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Failed to get account",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account retrieved successfully",
		"data":    account,
	})
}

func (h *AccountHandler) UpdateAccount(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request",
			"message": "Account ID is required",
		})
	}

	var req UpdateAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	account, err := h.service.UpdateAccount(c.Context(), id, &req)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid") {
			statusCode = fiber.StatusNotFound
		} else if strings.Contains(err.Error(), "already taken") {
			statusCode = fiber.StatusConflict
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Failed to update account",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account updated successfully",
		"data":    account,
	})
}

func (h *AccountHandler) DeleteAccount(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request",
			"message": "Account ID is required",
		})
	}

	err := h.service.DeleteAccount(c.Context(), id)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid") {
			statusCode = fiber.StatusNotFound
		}
		
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "Failed to delete account",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Account deleted successfully",
	})
}

func (h *AccountHandler) ListAccounts(c *fiber.Ctx) error {
	var req ListAccountsRequest
	
	if page := c.Query("page"); page != "" {
		if p, err := strconv.ParseInt(page, 10, 64); err == nil {
			req.Page = p
		}
	}
	
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.ParseInt(limit, 10, 64); err == nil {
			req.Limit = l
		}
	}
	
	req.Email = c.Query("email")
	req.Username = c.Query("username")
	
	if isActive := c.Query("is_active"); isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			req.IsActive = &active
		}
	}

	result, err := h.service.ListAccounts(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to list accounts",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Accounts retrieved successfully",
		"data":    result.Data,
		"meta": fiber.Map{
			"total":       result.Total,
			"page":        result.Page,
			"limit":       result.Limit,
			"total_pages": result.TotalPages,
			"has_next":    result.HasNext,
			"has_prev":    result.HasPrev,
		},
	})
}

func (h *AccountHandler) Login(c *fiber.Ctx) error {
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

func (h *AccountHandler) Logout(c *fiber.Ctx) error {
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

func (h *AccountHandler) Register(c *fiber.Ctx) error {
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

func (h *AccountHandler) VerifyEmail(c *fiber.Ctx) error {
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

func (h *AccountHandler) ResendEmailVerification(c *fiber.Ctx) error {
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

func (h *AccountHandler) ForgotPassword(c *fiber.Ctx) error {
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

func (h *AccountHandler) ResetPassword(c *fiber.Ctx) error {
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

func (h *AccountHandler) ChangePassword(c *fiber.Ctx) error {
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

func (h *AccountHandler) ValidateToken(c *fiber.Ctx) error {
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

func (h *AccountHandler) RefreshToken(c *fiber.Ctx) error {
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

func (h *AccountHandler) GetMe(c *fiber.Ctx) error {
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
	response, err := h.service.GetCurrentUser(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Failed to get current user",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Current user retrieved successfully",
		"data":    response,
	})
}

