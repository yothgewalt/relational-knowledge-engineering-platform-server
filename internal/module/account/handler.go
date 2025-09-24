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

