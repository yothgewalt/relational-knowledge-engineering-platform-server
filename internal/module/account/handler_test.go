package account

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

)

func setupTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
}

func TestNewAccountHandler(t *testing.T) {
	mockService := &MockAccountService{}
	handler := NewAccountHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

func TestAccountHandler_CreateAccount(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name: "successful account creation",
			requestBody: `{
				"email": "test@example.com",
				"username": "testuser",
				"first_name": "Test",
				"last_name": "User",
				"avatar": "https://example.com/avatar.jpg"
			}`,
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse()
				mockService.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.CreateAccountRequest")).Return(response, nil)
			},
			expectedStatus: fiber.StatusCreated,
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid json}`,
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectError:    true,
			errorContains:  "Invalid request body",
		},
		{
			name: "account already exists",
			requestBody: `{
				"email": "existing@example.com",
				"username": "existinguser",
				"first_name": "Test",
				"last_name": "User"
			}`,
			setupMock: func(mockService *MockAccountService) {
				mockService.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.CreateAccountRequest")).Return((*AccountResponse)(nil), errors.New("account with email existing@example.com already exists"))
			},
			expectedStatus: fiber.StatusConflict,
			expectError:    true,
			errorContains:  "Failed to create account",
		},
		{
			name: "internal server error",
			requestBody: `{
				"email": "test@example.com",
				"username": "testuser",
				"first_name": "Test",
				"last_name": "User"
			}`,
			setupMock: func(mockService *MockAccountService) {
				mockService.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.CreateAccountRequest")).Return((*AccountResponse)(nil), errors.New("database error"))
			},
			expectedStatus: fiber.StatusInternalServerError,
			expectError:    true,
			errorContains:  "Failed to create account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			handler := NewAccountHandler(mockService)
			app := setupTestApp()
			app.Post("/accounts", handler.CreateAccount)

			req := httptest.NewRequest("POST", "/accounts", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.expectError {
				var errorResponse fiber.Map
				err = json.Unmarshal(body, &errorResponse)
				assert.NoError(t, err)
				if tt.errorContains != "" {
					errorMsg := ""
					if msg, ok := errorResponse["message"].(string); ok {
						errorMsg += msg
					}
					if errField, ok := errorResponse["error"].(string); ok {
						errorMsg += errField
					}
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				var response fiber.Map
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
				assert.Equal(t, "Account created successfully", response["message"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAccountHandler_GetAccount(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name:      "successful get account",
			accountID: "507f1f77bcf86cd799439011",
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse(func(ar *AccountResponse) {
					ar.ID = "507f1f77bcf86cd799439011"
				})
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return(response, nil)
			},
			expectedStatus: fiber.StatusOK,
			expectError:    false,
		},
		{
			name:      "account not found",
			accountID: "507f1f77bcf86cd799439999",
			setupMock: func(mockService *MockAccountService) {
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439999").Return((*AccountResponse)(nil), errors.New("account not found"))
			},
			expectedStatus: fiber.StatusNotFound,
			expectError:    true,
			errorContains:  "Failed to get account",
		},
		{
			name:      "internal server error",
			accountID: "507f1f77bcf86cd799439011",
			setupMock: func(mockService *MockAccountService) {
				mockService.On("GetAccountByID", mock.Anything, "507f1f77bcf86cd799439011").Return((*AccountResponse)(nil), errors.New("database error"))
			},
			expectedStatus: fiber.StatusInternalServerError,
			expectError:    true,
			errorContains:  "Failed to get account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			handler := NewAccountHandler(mockService)
			app := setupTestApp()
			app.Get("/accounts/:id", handler.GetAccount)

			req := httptest.NewRequest("GET", "/accounts/"+tt.accountID, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.expectError {
				var errorResponse fiber.Map
				err = json.Unmarshal(body, &errorResponse)
				assert.NoError(t, err)
				if tt.errorContains != "" {
					errorMsg := ""
					if msg, ok := errorResponse["message"].(string); ok {
						errorMsg += msg
					}
					if errField, ok := errorResponse["error"].(string); ok {
						errorMsg += errField
					}
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				var response fiber.Map
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAccountHandler_UpdateAccount(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		requestBody    string
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name:      "successful account update",
			accountID: "507f1f77bcf86cd799439011",
			requestBody: `{
				"username": "updateduser",
				"first_name": "Updated",
				"is_active": false
			}`,
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse(func(ar *AccountResponse) {
					ar.ID = "507f1f77bcf86cd799439011"
					ar.Username = "updateduser"
					ar.FirstName = "Updated"
					ar.IsActive = false
				})
				mockService.On("UpdateAccount", mock.Anything, "507f1f77bcf86cd799439011", mock.AnythingOfType("*account.UpdateAccountRequest")).Return(response, nil)
			},
			expectedStatus: fiber.StatusOK,
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			accountID:      "507f1f77bcf86cd799439011",
			requestBody:    `{invalid json}`,
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectError:    true,
			errorContains:  "Invalid request body",
		},
		{
			name:      "account not found",
			accountID: "507f1f77bcf86cd799439999",
			requestBody: `{
				"username": "updateduser"
			}`,
			setupMock: func(mockService *MockAccountService) {
				mockService.On("UpdateAccount", mock.Anything, "507f1f77bcf86cd799439999", mock.AnythingOfType("*account.UpdateAccountRequest")).Return((*AccountResponse)(nil), errors.New("account not found"))
			},
			expectedStatus: fiber.StatusNotFound,
			expectError:    true,
			errorContains:  "Failed to update account",
		},
		{
			name:      "partial update",
			accountID: "507f1f77bcf86cd799439011",
			requestBody: `{
				"first_name": "OnlyFirstName"
			}`,
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse(func(ar *AccountResponse) {
					ar.ID = "507f1f77bcf86cd799439011"
					ar.FirstName = "OnlyFirstName"
				})
				mockService.On("UpdateAccount", mock.Anything, "507f1f77bcf86cd799439011", mock.AnythingOfType("*account.UpdateAccountRequest")).Return(response, nil)
			},
			expectedStatus: fiber.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			handler := NewAccountHandler(mockService)
			app := setupTestApp()
			app.Put("/accounts/:id", handler.UpdateAccount)

			req := httptest.NewRequest("PUT", "/accounts/"+tt.accountID, strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.expectError {
				var errorResponse fiber.Map
				err = json.Unmarshal(body, &errorResponse)
				assert.NoError(t, err)
				if tt.errorContains != "" {
					errorMsg := ""
					if msg, ok := errorResponse["message"].(string); ok {
						errorMsg += msg
					}
					if errField, ok := errorResponse["error"].(string); ok {
						errorMsg += errField
					}
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				var response fiber.Map
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
				assert.Equal(t, "Account updated successfully", response["message"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAccountHandler_DeleteAccount(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name:      "successful delete account",
			accountID: "507f1f77bcf86cd799439011",
			setupMock: func(mockService *MockAccountService) {
				mockService.On("DeleteAccount", mock.Anything, "507f1f77bcf86cd799439011").Return(nil)
			},
			expectedStatus: fiber.StatusOK,
			expectError:    false,
		},
		{
			name:      "account not found",
			accountID: "507f1f77bcf86cd799439999",
			setupMock: func(mockService *MockAccountService) {
				mockService.On("DeleteAccount", mock.Anything, "507f1f77bcf86cd799439999").Return(errors.New("account not found"))
			},
			expectedStatus: fiber.StatusNotFound,
			expectError:    true,
			errorContains:  "Failed to delete account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			handler := NewAccountHandler(mockService)
			app := setupTestApp()
			app.Delete("/accounts/:id", handler.DeleteAccount)

			req := httptest.NewRequest("DELETE", "/accounts/"+tt.accountID, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.expectError {
				var errorResponse fiber.Map
				err = json.Unmarshal(body, &errorResponse)
				assert.NoError(t, err)
				if tt.errorContains != "" {
					errorMsg := ""
					if msg, ok := errorResponse["message"].(string); ok {
						errorMsg += msg
					}
					if errField, ok := errorResponse["error"].(string); ok {
						errorMsg += errField
					}
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				var response fiber.Map
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Account deleted successfully", response["message"])
			}

			mockService.AssertExpectations(t)
		})
	}
}


func TestAccountHandler_GetAccountByEmail(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name:        "successful get by email",
			queryParams: "?email=test@example.com",
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse()
				mockService.On("GetAccountByEmail", mock.Anything, "test@example.com").Return(response, nil)
			},
			expectedStatus: fiber.StatusOK,
			expectError:    false,
		},
		{
			name:           "missing email parameter",
			queryParams:    "",
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectError:    true,
			errorContains:  "Email parameter is required",
		},
		{
			name:        "account not found",
			queryParams: "?email=notfound@example.com",
			setupMock: func(mockService *MockAccountService) {
				mockService.On("GetAccountByEmail", mock.Anything, "notfound@example.com").Return((*AccountResponse)(nil), errors.New("account not found"))
			},
			expectedStatus: fiber.StatusNotFound,
			expectError:    true,
			errorContains:  "Failed to get account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			handler := NewAccountHandler(mockService)
			app := setupTestApp()
			app.Get("/accounts/email", handler.GetAccountByEmail)

			req := httptest.NewRequest("GET", "/accounts/email"+tt.queryParams, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.expectError {
				var errorResponse fiber.Map
				err = json.Unmarshal(body, &errorResponse)
				assert.NoError(t, err)
				if tt.errorContains != "" {
					errorMsg := ""
					if msg, ok := errorResponse["message"].(string); ok {
						errorMsg += msg
					}
					if errField, ok := errorResponse["error"].(string); ok {
						errorMsg += errField
					}
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				var response fiber.Map
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
			}

			if len(mockService.ExpectedCalls) > 0 {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAccountHandler_GetAccountByUsername(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*MockAccountService)
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name:        "successful get by username",
			queryParams: "?username=testuser",
			setupMock: func(mockService *MockAccountService) {
				response := CreateTestAccountResponse()
				mockService.On("GetAccountByUsername", mock.Anything, "testuser").Return(response, nil)
			},
			expectedStatus: fiber.StatusOK,
			expectError:    false,
		},
		{
			name:           "missing username parameter",
			queryParams:    "",
			setupMock:      func(mockService *MockAccountService) {},
			expectedStatus: fiber.StatusBadRequest,
			expectError:    true,
			errorContains:  "Username parameter is required",
		},
		{
			name:        "account not found",
			queryParams: "?username=notfound",
			setupMock: func(mockService *MockAccountService) {
				mockService.On("GetAccountByUsername", mock.Anything, "notfound").Return((*AccountResponse)(nil), errors.New("account not found"))
			},
			expectedStatus: fiber.StatusNotFound,
			expectError:    true,
			errorContains:  "Failed to get account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAccountService{}
			tt.setupMock(mockService)

			handler := NewAccountHandler(mockService)
			app := setupTestApp()
			app.Get("/accounts/username", handler.GetAccountByUsername)

			req := httptest.NewRequest("GET", "/accounts/username"+tt.queryParams, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.expectError {
				var errorResponse fiber.Map
				err = json.Unmarshal(body, &errorResponse)
				assert.NoError(t, err)
				if tt.errorContains != "" {
					errorMsg := ""
					if msg, ok := errorResponse["message"].(string); ok {
						errorMsg += msg
					}
					if errField, ok := errorResponse["error"].(string); ok {
						errorMsg += errField
					}
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				var response fiber.Map
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
			}

			if len(mockService.ExpectedCalls) > 0 {
				mockService.AssertExpectations(t)
			}
		})
	}
}
