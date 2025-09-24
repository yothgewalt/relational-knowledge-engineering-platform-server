package account

import (
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

// CreateAccount godoc
// @Summary Create account
// @Description Create a new account (admin operation)
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body CreateAccountRequest true "Account creation details"
// @Success 201 {object} map[string]interface{} "Account created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 409 {object} map[string]interface{} "Conflict - account already exists"
// @Router /accounts [post]
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

// GetAccount godoc
// @Summary Get account by ID
// @Description Get account information by account ID (requires authentication and ownership validation)
// @Tags accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} map[string]interface{} "Account retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Router /accounts/{id} [get]
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

// GetAccountByEmail godoc
// @Summary Get account by email
// @Description Get account information by email address (optional authentication)
// @Tags accounts
// @Accept json
// @Produce json
// @Param email query string true "Email address"
// @Success 200 {object} map[string]interface{} "Account retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - email parameter required"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Router /accounts/email [get]
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

// GetAccountByUsername godoc
// @Summary Get account by username
// @Description Get account information by username (optional authentication)
// @Tags accounts
// @Accept json
// @Produce json
// @Param username query string true "Username"
// @Success 200 {object} map[string]interface{} "Account retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Bad request - username parameter required"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Router /accounts/username [get]
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

// UpdateAccount godoc
// @Summary Update account
// @Description Update account information (requires authentication and ownership validation)
// @Tags accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body UpdateAccountRequest true "Account update details"
// @Success 200 {object} map[string]interface{} "Account updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 409 {object} map[string]interface{} "Conflict - username already taken"
// @Router /accounts/{id} [put]
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

// DeleteAccount godoc
// @Summary Delete account
// @Description Delete account by ID (requires authentication and ownership validation)
// @Tags accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} map[string]interface{} "Account deleted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Router /accounts/{id} [delete]
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


// Login godoc
// @Summary User login
// @Description Authenticate user with email and password
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login successful"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid credentials"
// @Failure 403 {object} map[string]interface{} "Forbidden - account inactive"
// @Router /accounts/login [post]
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

// Logout godoc
// @Summary User logout
// @Description Logout user by invalidating the session token
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Logout successful"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /accounts/logout [post]
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

// Register godoc
// @Summary User registration
// @Description Register a new user account with email verification
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} map[string]interface{} "Account created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 409 {object} map[string]interface{} "Conflict - account already exists"
// @Router /accounts/register [post]
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

// VerifyEmail godoc
// @Summary Verify email address
// @Description Verify user's email address with OTP code
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body VerifyEmailRequest true "Email verification details"
// @Success 200 {object} map[string]interface{} "Email verified successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or expired OTP"
// @Router /accounts/verify-email [post]
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

// ResendEmailVerification godoc
// @Summary Resend email verification
// @Description Resend email verification OTP to user's email address
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body ResendVerificationRequest true "Resend verification details"
// @Success 200 {object} map[string]interface{} "Verification email sent successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 409 {object} map[string]interface{} "Email already verified"
// @Router /accounts/resend-verification [post]
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

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send password reset OTP to user's email address
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Forgot password details"
// @Success 200 {object} map[string]interface{} "Password reset email sent"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /accounts/forgot-password [post]
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

// ResetPassword godoc
// @Summary Reset password
// @Description Reset user's password using OTP code
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Password reset details"
// @Success 200 {object} map[string]interface{} "Password reset successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or expired OTP"
// @Router /accounts/reset-password [post]
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

// ChangePassword godoc
// @Summary Change password
// @Description Change user's password (requires authentication)
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "Change password details"
// @Success 200 {object} map[string]interface{} "Password changed successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized - incorrect password"
// @Failure 403 {object} map[string]interface{} "Forbidden - account inactive"
// @Router /accounts/change-password [post]
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

// ValidateToken godoc
// @Summary Validate JWT token
// @Description Validate the provided JWT token and return user information
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Token is valid"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or expired token"
// @Router /accounts/validate [post]
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

// RefreshToken godoc
// @Summary Refresh JWT token
// @Description Refresh the provided JWT token to extend session
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Token refreshed successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - token refresh failed"
// @Router /accounts/refresh [post]
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

// GetMe godoc
// @Summary Get current user
// @Description Get current authenticated user's information
// @Tags accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Current user retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /accounts/me [get]
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

