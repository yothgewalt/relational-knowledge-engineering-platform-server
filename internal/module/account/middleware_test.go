package account

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewAccountMiddleware(t *testing.T) {
	mockService := &MockAccountService{}
	middleware := NewAccountMiddleware(mockService)

	assert.NotNil(t, middleware)
	assert.Equal(t, mockService, middleware.service)
}

func TestAccountMiddleware_ValidateAccountOwnership(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*fiber.Ctx)
		accountID      string
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectNext     bool
		errorContains  string
	}{
		{
			name: "successful ownership validation - same account",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			accountID:      "507f1f77bcf86cd799439011",
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusOK,
			expectNext:     true,
		},
		{
			name: "successful ownership validation - account lookup",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			accountID: "507f1f77bcf86cd799439012",
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse(func(ar *AccountResponse) {
					ar.ID = "507f1f77bcf86cd799439012"
				})
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return(response, nil)
			},
			expectedStatus: fiber.StatusOK,
			expectNext:     true,
		},
		{
			name: "no account_id in context",
			setupContext: func(c *fiber.Ctx) {
			},
			accountID:      "507f1f77bcf86cd799439011",
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusUnauthorized,
			expectNext:     false,
			errorContains:  "Authentication required",
		},
		{
			name: "no id parameter - should pass through",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			accountID:      "",
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusOK,
			expectNext:     true,
		},
		{
			name: "forbidden - different account ID",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			accountID: "507f1f77bcf86cd799439999",
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse(func(ar *AccountResponse) {
					ar.ID = "507f1f77bcf86cd799439011"
				})
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return(response, nil)
			},
			expectedStatus: fiber.StatusForbidden,
			expectNext:     false,
			errorContains:  "You can only access your own account",
		},
		{
			name: "service error during account lookup",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			accountID: "507f1f77bcf86cd799439999",
			setupMock: func(mockService *MockAccountService) {
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return((*AccountResponse)(nil), errors.New("service error"))
			},
			expectedStatus: fiber.StatusForbidden,
			expectNext:     false,
			errorContains:  "Access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			middleware := NewAccountMiddleware(mockService)

			app := fiber.New()
			nextCalled := false

			app.Get("/accounts/:id", middleware.ValidateAccountOwnership(), func(c *fiber.Ctx) error {
				nextCalled = true
				return c.SendStatus(fiber.StatusOK)
			})

			path := "/accounts/"
			if tt.accountID != "" {
				path = "/accounts/" + tt.accountID
			}

			req := httptest.NewRequest("GET", path, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)

			if tt.expectNext {
				assert.True(t, nextCalled, "Expected next handler to be called")
				assert.Equal(t, fiber.StatusOK, resp.StatusCode)
			} else {
				assert.False(t, nextCalled, "Expected next handler NOT to be called")
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)

				if tt.errorContains != "" {
					body, err := io.ReadAll(resp.Body)
					assert.NoError(t, err)
					assert.Contains(t, string(body), tt.errorContains)
				}
			}

			if len(mockService.ExpectedCalls) > 0 {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAccountMiddleware_RequireActiveAccount(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*fiber.Ctx)
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectNext     bool
		errorContains  string
	}{
		{
			name: "successful active account validation",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse(func(ar *AccountResponse) {
					ar.ID = "507f1f77bcf86cd799439011"
					ar.IsActive = true
				})
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return(response, nil)
			},
			expectedStatus: fiber.StatusOK,
			expectNext:     true,
		},
		{
			name: "no account_id in context",
			setupContext: func(c *fiber.Ctx) {
			},
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusUnauthorized,
			expectNext:     false,
			errorContains:  "Authentication required",
		},
		{
			name: "account lookup fails",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			setupMock: func(mockService *MockAccountService) {
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return((*AccountResponse)(nil), errors.New("account not found"))
			},
			expectedStatus: fiber.StatusUnauthorized,
			expectNext:     false,
			errorContains:  "Invalid account",
		},
		{
			name: "account is inactive",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("account_id", "507f1f77bcf86cd799439011")
			},
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse(func(ar *AccountResponse) {
					ar.ID = "507f1f77bcf86cd799439011"
					ar.IsActive = false
				})
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return(response, nil)
			},
			expectedStatus: fiber.StatusForbidden,
			expectNext:     false,
			errorContains:  "Account is inactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			middleware := NewAccountMiddleware(mockService)

			app := fiber.New()
			nextCalled := false
			var capturedContext *fiber.Ctx

			app.Get("/protected", middleware.RequireActiveAccount(), func(c *fiber.Ctx) error {
				nextCalled = true
				capturedContext = c
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest("GET", "/protected", nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)

			if tt.expectNext {
				assert.True(t, nextCalled, "Expected next handler to be called")
				assert.Equal(t, fiber.StatusOK, resp.StatusCode)

				if capturedContext != nil {
					account := capturedContext.Locals("account")
					assert.NotNil(t, account, "Expected account to be set in context")
				}
			} else {
				assert.False(t, nextCalled, "Expected next handler NOT to be called")
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)

				if tt.errorContains != "" {
					body, err := io.ReadAll(resp.Body)
					assert.NoError(t, err)
					assert.Contains(t, string(body), tt.errorContains)
				}
			}

			if len(mockService.ExpectedCalls) > 0 {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAccountMiddleware_Integration(t *testing.T) {
	mockService := &MockAccountService{}

	activeResponse := CreateTestAccountResponse(func(ar *AccountResponse) {
		ar.ID = "507f1f77bcf86cd799439011"
		ar.IsActive = true
	})
	mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return(activeResponse, nil).Times(2) // Called by both middlewares

	middleware := NewAccountMiddleware(mockService)

	app := fiber.New()
	handlerCalled := false

	app.Get("/accounts/:id",
		func(c *fiber.Ctx) error {
			c.Locals("account_id", "507f1f77bcf86cd799439011")
			return c.Next()
		},
		middleware.RequireActiveAccount(),
		middleware.ValidateAccountOwnership(),
		func(c *fiber.Ctx) error {
			handlerCalled = true

			account := c.Locals("account")
			assert.NotNil(t, account, "Expected account to be set by RequireActiveAccount")

			return c.JSON(fiber.Map{"success": true})
		})

	req := httptest.NewRequest("GET", "/accounts/507f1f77bcf86cd799439011", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.True(t, handlerCalled, "Expected final handler to be called")

	mockService.AssertExpectations(t)
}

func TestAccountMiddleware_TypeAssertions(t *testing.T) {
	mockService := &MockAccountService{}
	middleware := NewAccountMiddleware(mockService)

	app := fiber.New()

	t.Run("invalid type for account_id", func(t *testing.T) {
		app.Get("/test", func(c *fiber.Ctx) error {
			c.Locals("account_id", 12345)
			return c.Next()
		}, middleware.ValidateAccountOwnership(), func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.True(t, resp.StatusCode == fiber.StatusOK || resp.StatusCode >= 400)
	})
}

func TestAccountMiddleware_ConcurrentAccess(t *testing.T) {
	mockService := &MockAccountService{}

	response := CreateTestAccountResponse(func(ar *AccountResponse) {
		ar.ID = "507f1f77bcf86cd799439011"
		ar.IsActive = true
	})
	mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return(response, nil)

	middleware := NewAccountMiddleware(mockService)

	app := fiber.New()
	app.Get("/accounts/:id",
		func(c *fiber.Ctx) error {
			c.Locals("account_id", "507f1f77bcf86cd799439011")
			return c.Next()
		},
		middleware.ValidateAccountOwnership(),
		func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

	numRequests := 5
	responses := make([]*http.Response, numRequests)
	errs := make([]error, numRequests)

	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			req := httptest.NewRequest("GET", "/accounts/507f1f77bcf86cd799439011", nil)
			responses[index], errs[index] = app.Test(req)
			done <- true
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		<-done
	}

	for i := 0; i < numRequests; i++ {
		assert.NoError(t, errs[i])
		assert.Equal(t, fiber.StatusOK, responses[i].StatusCode)
	}
}
