package account

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/jwt"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
)

type MockResendService struct {
	mock.Mock
}

func (m *MockResendService) SendEmail(ctx context.Context, req *resend.EmailRequest) (*resend.EmailResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*resend.EmailResponse), args.Error(1)
}

func (m *MockResendService) HealthCheck(ctx context.Context) resend.HealthStatus {
	args := m.Called(ctx)
	return args.Get(0).(resend.HealthStatus)
}

func (m *MockResendService) SendBulkEmails(ctx context.Context, requests []*resend.EmailRequest) ([]*resend.EmailResponse, error) {
	args := m.Called(ctx, requests)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*resend.EmailResponse), args.Error(1)
}

func (m *MockResendService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func setupAccountService() (*accountService, *MockAccountRepository, *MockAccountIdentityRepository, *jwt.JWTService, *MockResendService) {
	mockAccountRepo := &MockAccountRepository{}
	mockAccountIdentityRepo := &MockAccountIdentityRepository{}

	jwtConfig := jwt.JWTConfig{
		SecretKey:     "test-secret-key-for-testing-purposes",
		TokenDuration: 24 * time.Hour,
		Issuer:        "test-platform",
	}
	jwtService, _ := jwt.NewJWTService(jwtConfig)

	mockResendService := &MockResendService{}

	service := &accountService{
		repository:                mockAccountRepo,
		accountIdentityRepository: mockAccountIdentityRepo,
		jwtService:                jwtService,
		resendService:             mockResendService,
		fromEmail:                 "test@platform.com",
	}

	return service, mockAccountRepo, mockAccountIdentityRepo, jwtService, mockResendService
}

func TestAccountService_Login(t *testing.T) {
	tests := []struct {
		name      string
		request   *LoginRequest
		userAgent string
		ipAddress string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "successful login",
			request:   CreateTestLoginRequest(),
			userAgent: "Mozilla/5.0",
			ipAddress: "192.168.1.1",
			wantErr:   false,
		},
		{
			name: "account not found",
			request: CreateTestLoginRequest(func(r *LoginRequest) {
				r.Email = "notfound@example.com"
			}),
			userAgent: "Mozilla/5.0",
			ipAddress: "192.168.1.1",
			wantErr:   true,
			errMsg:    "invalid email or password",
		},
		{
			name:      "inactive account",
			request:   CreateTestLoginRequest(),
			userAgent: "Mozilla/5.0",
			ipAddress: "192.168.1.1",
			wantErr:   true,
			errMsg:    "account is inactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockAccountRepo, mockIdentityRepo, _, _ := setupAccountService()

			switch tt.name {
			case "successful login":
				mockAccountRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(
					CreateTestAccount(func(a *Account) {
						a.Email = "test@example.com"
						a.IsActive = true
					}), nil)
				session := CreateTestSession()
				mockIdentityRepo.On("CreateSession", mock.Anything, mock.MatchedBy(func(s *Session) bool {
					return s.IsActive
				})).Return(session, nil)
			case "account not found":
				mockAccountRepo.On("GetByEmail", mock.Anything, "notfound@example.com").Return(nil, fmt.Errorf("account not found"))
			case "inactive account":
				mockAccountRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(
					CreateTestAccount(func(a *Account) {
						a.IsActive = false
					}), nil)
			}

			result, err := service.Login(context.Background(), tt.request, tt.userAgent, tt.ipAddress)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Token)
				assert.NotNil(t, result.Account)
			}

			mockAccountRepo.AssertExpectations(t)
			mockIdentityRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_Register(t *testing.T) {
	tests := []struct {
		name    string
		request *RegisterRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "successful registration",
			request: CreateTestRegisterRequest(),
			wantErr: false,
		},
		{
			name:    "email already exists",
			request: CreateTestRegisterRequest(),
			wantErr: true,
			errMsg:  "email already exists",
		},
		{
			name:    "username already exists",
			request: CreateTestRegisterRequest(),
			wantErr: true,
			errMsg:  "username already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockAccountRepo, mockIdentityRepo, _, mockResend := setupAccountService()

			if tt.name == "successful registration" {
				mockAccountRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				mockAccountRepo.On("ExistsByUsername", mock.Anything, "testuser").Return(false, nil)
				mockAccountRepo.On("Create", mock.Anything, mock.Anything).Return(CreateTestAccount(), nil)

				otp := CreateTestOTP()
				mockIdentityRepo.On("CreateOTP", mock.Anything, "test@example.com", OTPPurposeEmailVerification).Return(otp, nil)

				emailResp := &resend.EmailResponse{ID: "email-id-123"}
				mockResend.On("SendEmail", mock.Anything, mock.Anything).Return(emailResp, nil).Twice() // Welcome + verification emails
			} else if tt.name == "email already exists" {
				mockAccountRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(true, nil)
			} else if tt.name == "username already exists" {
				mockAccountRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
				mockAccountRepo.On("ExistsByUsername", mock.Anything, "testuser").Return(true, nil)
			}

			result, err := service.Register(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, result.Message, "Registration successful")
			}

			mockAccountRepo.AssertExpectations(t)
			mockIdentityRepo.AssertExpectations(t)
			mockResend.AssertExpectations(t)
		})
	}
}

func TestAccountService_ValidateToken(t *testing.T) {
	service, mockAccountRepo, mockIdentityRepo, jwtService, _ := setupAccountService()

	customClaims := map[string]any{
		"account_id": "507f1f77bcf86cd799439011",
		"email":      "test@example.com",
		"username":   "testuser",
	}
	validToken, _ := jwtService.Generate(customClaims)
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(validToken)))

	tests := []struct {
		name    string
		token   string
		wantErr bool
		valid   bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
			valid:   true,
		},
		{
			name:    "invalid JWT token",
			token:   "invalid.jwt.token",
			wantErr: false,
			valid:   false,
		},
		{
			name:    "session not found",
			token:   validToken,
			wantErr: false,
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAccountRepo.ExpectedCalls = nil
			mockAccountRepo.Calls = nil
			mockIdentityRepo.ExpectedCalls = nil
			mockIdentityRepo.Calls = nil

			switch tt.name {
			case "valid token":
				session := CreateTestSession(func(s *Session) {
					s.TokenHash = tokenHash
					s.IsActive = true
					s.ExpiresAt = time.Now().Add(24 * time.Hour)
				})
				mockIdentityRepo.On("GetSessionByToken", mock.Anything, tokenHash).Return(session, nil)
				mockIdentityRepo.On("UpdateSessionLastUsed", mock.Anything, session.ID).Return(nil)

				account := CreateTestAccount(func(a *Account) {
					a.IsActive = true
				})
				mockAccountRepo.On("GetByID", mock.Anything, mock.Anything).Return(account, nil)
			case "session not found":
				mockIdentityRepo.On("GetSessionByToken", mock.Anything, tokenHash).Return(nil, fmt.Errorf("session not found"))
			}

			result, err := service.ValidateToken(context.Background(), tt.token)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.valid, result.Valid)

			if tt.valid {
				assert.NotNil(t, result.Claims)
				assert.NotNil(t, result.Account)
			}

			mockAccountRepo.AssertExpectations(t)
			mockIdentityRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_Logout(t *testing.T) {
	service, _, mockIdentityRepo, jwtService, _ := setupAccountService()

	customClaims := map[string]any{
		"account_id": "507f1f77bcf86cd799439011",
	}
	validToken, _ := jwtService.Generate(customClaims)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "successful logout",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "logout fails",
			token:   validToken,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockIdentityRepo.ExpectedCalls = nil
			mockIdentityRepo.Calls = nil

			if tt.name == "successful logout" {
				mockIdentityRepo.On("DeactivateSession", mock.Anything, mock.Anything).Return(nil)
			} else {
				mockIdentityRepo.On("DeactivateSession", mock.Anything, mock.Anything).Return(fmt.Errorf("database error"))
			}

			err := service.Logout(context.Background(), tt.token)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockIdentityRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_GetCurrentUser(t *testing.T) {
	service, mockAccountRepo, mockIdentityRepo, jwtService, _ := setupAccountService()

	customClaims := map[string]any{
		"account_id": "507f1f77bcf86cd799439011",
		"email":      "test@example.com",
		"username":   "testuser",
	}
	validToken, _ := jwtService.Generate(customClaims)
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(validToken)))

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "successful get current user",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid.jwt.token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAccountRepo.ExpectedCalls = nil
			mockAccountRepo.Calls = nil
			mockIdentityRepo.ExpectedCalls = nil
			mockIdentityRepo.Calls = nil

			if tt.name == "successful get current user" {
				session := CreateTestSession(func(s *Session) {
					s.TokenHash = tokenHash
					s.IsActive = true
					s.ExpiresAt = time.Now().Add(24 * time.Hour)
				})
				mockIdentityRepo.On("GetSessionByToken", mock.Anything, tokenHash).Return(session, nil).Twice()
				mockIdentityRepo.On("UpdateSessionLastUsed", mock.Anything, session.ID).Return(nil)

				account := CreateTestAccount()
				mockAccountRepo.On("GetByID", mock.Anything, mock.Anything).Return(account, nil)
			}

			result, err := service.GetCurrentUser(context.Background(), tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.Account)
				assert.NotNil(t, result.Session)
			}

			mockAccountRepo.AssertExpectations(t)
			mockIdentityRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_VerifyEmail(t *testing.T) {
	tests := []struct {
		name    string
		request *VerifyEmailRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "successful verification",
			request: CreateTestVerifyEmailRequest(),
			wantErr: false,
		},
		{
			name: "invalid OTP",
			request: CreateTestVerifyEmailRequest(func(r *VerifyEmailRequest) {
				r.OTP = "invalid"
			}),
			wantErr: true,
			errMsg:  "invalid or expired OTP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockAccountRepo, mockIdentityRepo, _, _ := setupAccountService()

			if tt.name == "successful verification" {
				otp := CreateTestOTP()
				mockIdentityRepo.On("ValidateOTP", mock.Anything, "test@example.com", OTPPurposeEmailVerification, "123456").Return(otp, nil)
				mockIdentityRepo.On("DeleteOTP", mock.Anything, "test@example.com", OTPPurposeEmailVerification).Return(nil)

				account := CreateTestAccount()
				mockAccountRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(account, nil)
				mockAccountRepo.On("Update", mock.Anything, account.ID, mock.Anything).Return(account, nil)
			} else {
				mockIdentityRepo.On("ValidateOTP", mock.Anything, "test@example.com", OTPPurposeEmailVerification, "invalid").Return(nil, fmt.Errorf("invalid OTP"))
			}

			err := service.VerifyEmail(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			mockAccountRepo.AssertExpectations(t)
			mockIdentityRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_hashToken(t *testing.T) {
	service, _, _, _, _ := setupAccountService()

	token1 := "test.token.here"
	token2 := "different.token.here"
	token3 := "test.token.here"

	hash1 := service.hashToken(token1)
	hash2 := service.hashToken(token2)
	hash3 := service.hashToken(token3)

	assert.Equal(t, hash1, hash3)
	assert.NotEqual(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64)
}
