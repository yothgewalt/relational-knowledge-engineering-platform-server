package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

)

func TestNewAccountService(t *testing.T) {
	t.Skip("Skipping NewAccountService test as it requires real MongoDB connection")
}

func TestAccountService_CreateAccount(t *testing.T) {
	tests := []struct {
		name          string
		request       *CreateAccountRequest
		setupMock     func(*MockAccountRepository)
		expectedError string
		expectSuccess bool
	}{
		{
			name:    "successful account creation",
			request: CreateTestCreateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				mockRepo.On("ExistsByUsername", mock.Anything, "testuser").Return(false, nil)

				createdAccount := CreateTestAccount()
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Return(createdAccount, nil)
			},
			expectSuccess: true,
		},
		{
			name:    "email already exists",
			request: CreateTestCreateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(true, nil)
			},
			expectedError: "account with email test@example.com already exists",
			expectSuccess: false,
		},
		{
			name:    "username already exists",
			request: CreateTestCreateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				mockRepo.On("ExistsByUsername", mock.Anything, "testuser").Return(true, nil)
			},
			expectedError: "account with username testuser already exists",
			expectSuccess: false,
		},
		{
			name:    "email check fails",
			request: CreateTestCreateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, errors.New("database error"))
			},
			expectedError: "failed to check email existence",
			expectSuccess: false,
		},
		{
			name:    "username check fails",
			request: CreateTestCreateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				mockRepo.On("ExistsByUsername", mock.Anything, "testuser").Return(false, errors.New("database error"))
			},
			expectedError: "failed to check username existence",
			expectSuccess: false,
		},
		{
			name:    "account creation fails",
			request: CreateTestCreateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				mockRepo.On("ExistsByUsername", mock.Anything, "testuser").Return(false, nil)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Return((*Account)(nil), errors.New("create failed"))
			},
			expectedError: "failed to create account",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAccountRepository{}
			tt.setupMock(mockRepo)

			service := &accountService{repository: mockRepo}

			result, err := service.CreateAccount(context.Background(), tt.request)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.Email, result.Email)
				assert.Equal(t, tt.request.Username, result.Username)
				assert.Equal(t, tt.request.FirstName, result.FirstName)
				assert.Equal(t, tt.request.LastName, result.LastName)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_GetAccountByID(t *testing.T) {
	validID := primitive.NewObjectID()
	invalidID := "invalid-id"

	tests := []struct {
		name          string
		id            string
		setupMock     func(*MockAccountRepository)
		expectedError string
		expectSuccess bool
	}{
		{
			name: "successful get by ID",
			id:   validID.Hex(),
			setupMock: func(mockRepo *MockAccountRepository) {
				account := CreateTestAccount(func(a *Account) {
					a.ID = validID
				})
				mockRepo.On("GetByID", mock.Anything, validID).Return(account, nil)
			},
			expectSuccess: true,
		},
		{
			name:          "invalid ObjectID",
			id:            invalidID,
			setupMock:     func(mockRepo *MockAccountRepository) {},
			expectedError: "invalid account ID format",
			expectSuccess: false,
		},
		{
			name: "account not found",
			id:   validID.Hex(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("GetByID", mock.Anything, validID).Return((*Account)(nil), errors.New("not found"))
			},
			expectedError: "failed to get account by ID",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAccountRepository{}
			tt.setupMock(mockRepo)

			service := &accountService{repository: mockRepo}

			result, err := service.GetAccountByID(context.Background(), tt.id)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.id, result.ID)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_GetAccountByEmail(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		setupMock     func(*MockAccountRepository)
		expectedError string
		expectSuccess bool
	}{
		{
			name:  "successful get by email",
			email: "test@example.com",
			setupMock: func(mockRepo *MockAccountRepository) {
				account := CreateTestAccount()
				mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(account, nil)
			},
			expectSuccess: true,
		},
		{
			name:  "account not found",
			email: "notfound@example.com",
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("GetByEmail", mock.Anything, "notfound@example.com").Return((*Account)(nil), errors.New("not found"))
			},
			expectedError: "failed to get account by email",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAccountRepository{}
			tt.setupMock(mockRepo)

			service := &accountService{repository: mockRepo}

			result, err := service.GetAccountByEmail(context.Background(), tt.email)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.email, result.Email)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_GetAccountByUsername(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		setupMock     func(*MockAccountRepository)
		expectedError string
		expectSuccess bool
	}{
		{
			name:     "successful get by username",
			username: "testuser",
			setupMock: func(mockRepo *MockAccountRepository) {
				account := CreateTestAccount()
				mockRepo.On("GetByUsername", mock.Anything, "testuser").Return(account, nil)
			},
			expectSuccess: true,
		},
		{
			name:     "account not found",
			username: "notfound",
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("GetByUsername", mock.Anything, "notfound").Return((*Account)(nil), errors.New("not found"))
			},
			expectedError: "failed to get account by username",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAccountRepository{}
			tt.setupMock(mockRepo)

			service := &accountService{repository: mockRepo}

			result, err := service.GetAccountByUsername(context.Background(), tt.username)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.username, result.Username)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_UpdateAccount(t *testing.T) {
	validID := primitive.NewObjectID()

	tests := []struct {
		name          string
		id            string
		request       *UpdateAccountRequest
		setupMock     func(*MockAccountRepository)
		expectedError string
		expectSuccess bool
	}{
		{
			name:    "successful update",
			id:      validID.Hex(),
			request: CreateTestUpdateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				updatedAccount := CreateTestAccount(func(a *Account) {
					a.ID = validID
					a.Username = "updateduser"
					a.UpdatedAt = time.Now()
				})
				mockRepo.On("Update", mock.Anything, validID, mock.Anything).Return(updatedAccount, nil)
			},
			expectSuccess: true,
		},
		{
			name:          "invalid ID",
			id:            "invalid-id",
			request:       CreateTestUpdateAccountRequest(),
			setupMock:     func(mockRepo *MockAccountRepository) {},
			expectedError: "invalid account ID format",
			expectSuccess: false,
		},
		{
			name:    "update fails",
			id:      validID.Hex(),
			request: CreateTestUpdateAccountRequest(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("Update", mock.Anything, validID, mock.Anything).Return((*Account)(nil), errors.New("update failed"))
			},
			expectedError: "failed to update account",
			expectSuccess: false,
		},
		{
			name: "partial update",
			id:   validID.Hex(),
			request: CreateTestUpdateAccountRequest(func(req *UpdateAccountRequest) {
				req.FirstName = StringPtr("NewName")
				req.Username = nil
				req.LastName = nil
				req.Avatar = nil
				req.IsActive = nil
			}),
			setupMock: func(mockRepo *MockAccountRepository) {
				updatedAccount := CreateTestAccount(func(a *Account) {
					a.ID = validID
					a.FirstName = "NewName"
				})
				mockRepo.On("Update", mock.Anything, validID, mock.Anything).Return(updatedAccount, nil)
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAccountRepository{}
			tt.setupMock(mockRepo)

			service := &accountService{repository: mockRepo}

			result, err := service.UpdateAccount(context.Background(), tt.id, tt.request)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.id, result.ID)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_DeleteAccount(t *testing.T) {
	validID := primitive.NewObjectID()

	tests := []struct {
		name          string
		id            string
		setupMock     func(*MockAccountRepository)
		expectedError string
		expectSuccess bool
	}{
		{
			name: "successful delete",
			id:   validID.Hex(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("Delete", mock.Anything, validID).Return(nil)
			},
			expectSuccess: true,
		},
		{
			name:          "invalid ID",
			id:            "invalid-id",
			setupMock:     func(mockRepo *MockAccountRepository) {},
			expectedError: "invalid account ID format",
			expectSuccess: false,
		},
		{
			name: "delete fails",
			id:   validID.Hex(),
			setupMock: func(mockRepo *MockAccountRepository) {
				mockRepo.On("Delete", mock.Anything, validID).Return(errors.New("delete failed"))
			},
			expectedError: "failed to delete account",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAccountRepository{}
			tt.setupMock(mockRepo)

			service := &accountService{repository: mockRepo}

			err := service.DeleteAccount(context.Background(), tt.id)

			if tt.expectSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}


func TestAccountService_AccountCreationTimestamps(t *testing.T) {
	mockRepo := &MockAccountRepository{}

	var capturedAccount *Account
	mockRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
	mockRepo.On("ExistsByUsername", mock.Anything, "testuser").Return(false, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Run(func(args mock.Arguments) {
		capturedAccount = args.Get(1).(*Account)
	}).Return(CreateTestAccount(), nil)

	service := &accountService{repository: mockRepo}
	request := CreateTestCreateAccountRequest()

	result, err := service.CreateAccount(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, capturedAccount)

	assert.False(t, capturedAccount.CreatedAt.IsZero())
	assert.False(t, capturedAccount.UpdatedAt.IsZero())
	assert.Equal(t, capturedAccount.CreatedAt, capturedAccount.UpdatedAt)

	assert.True(t, capturedAccount.IsActive)

	mockRepo.AssertExpectations(t)
}

func TestAccountService_UpdateAccountTimestamp(t *testing.T) {
	validID := primitive.NewObjectID()
	mockRepo := &MockAccountRepository{}

	var capturedUpdateData interface{}
	mockRepo.On("Update", mock.Anything, validID, mock.Anything).Run(func(args mock.Arguments) {
		capturedUpdateData = args.Get(2)
	}).Return(CreateTestAccount(func(a *Account) {
		a.ID = validID
		a.UpdatedAt = time.Now()
	}), nil)

	service := &accountService{repository: mockRepo}
	request := CreateTestUpdateAccountRequest()

	result, err := service.UpdateAccount(context.Background(), validID.Hex(), request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, capturedUpdateData)

	mockRepo.AssertExpectations(t)
}
