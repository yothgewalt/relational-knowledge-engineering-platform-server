package account

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
)

type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) Create(ctx context.Context, account *Account) (*Account, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Account), args.Error(1)
}

func (m *MockAccountRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Account), args.Error(1)
}

func (m *MockAccountRepository) GetByEmail(ctx context.Context, email string) (*Account, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Account), args.Error(1)
}

func (m *MockAccountRepository) GetByUsername(ctx context.Context, username string) (*Account, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Account), args.Error(1)
}

func (m *MockAccountRepository) Update(ctx context.Context, id primitive.ObjectID, updateData bson.M) (*Account, error) {
	args := m.Called(ctx, id, updateData)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Account), args.Error(1)
}

func (m *MockAccountRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountRepository) List(ctx context.Context, filter bson.M, pagination mongo.PaginationOptions) (*mongo.PaginatedResult[Account], error) {
	args := m.Called(ctx, filter, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.PaginatedResult[Account]), args.Error(1)
}

func (m *MockAccountRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccountRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountResponse), args.Error(1)
}

func (m *MockAccountService) GetAccountByID(ctx context.Context, id string) (*AccountResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountResponse), args.Error(1)
}

func (m *MockAccountService) GetAccountByEmail(ctx context.Context, email string) (*AccountResponse, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountResponse), args.Error(1)
}

func (m *MockAccountService) GetAccountByUsername(ctx context.Context, username string) (*AccountResponse, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountResponse), args.Error(1)
}

func (m *MockAccountService) UpdateAccount(ctx context.Context, id string, req *UpdateAccountRequest) (*AccountResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountResponse), args.Error(1)
}

func (m *MockAccountService) DeleteAccount(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountService) ListAccounts(ctx context.Context, req *ListAccountsRequest) (*mongo.PaginatedResult[AccountResponse], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.PaginatedResult[AccountResponse]), args.Error(1)
}

func (m *MockAccountService) Login(ctx context.Context, req *LoginRequest, userAgent, ipAddress string) (*LoginResponse, error) {
	args := m.Called(ctx, req, userAgent, ipAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoginResponse), args.Error(1)
}

func (m *MockAccountService) Logout(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockAccountService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RegisterResponse), args.Error(1)
}

func (m *MockAccountService) VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockAccountService) ResendEmailVerification(ctx context.Context, req *ResendVerificationRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockAccountService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockAccountService) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockAccountService) ChangePassword(ctx context.Context, accountID string, req *ChangePasswordRequest) error {
	args := m.Called(ctx, accountID, req)
	return args.Error(0)
}

func (m *MockAccountService) ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ValidateTokenResponse), args.Error(1)
}

func (m *MockAccountService) RefreshToken(ctx context.Context, token string, userAgent, ipAddress string) (*RefreshTokenResponse, error) {
	args := m.Called(ctx, token, userAgent, ipAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RefreshTokenResponse), args.Error(1)
}

func (m *MockAccountService) GetCurrentUser(ctx context.Context, token string) (*MeResponse, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MeResponse), args.Error(1)
}

func CreateTestAccount(overrides ...func(*Account)) *Account {
	objectID := primitive.NewObjectID()
	now := time.Now()

	account := &Account{
		ID:        objectID,
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Avatar:    "https://example.com/avatar.jpg",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, override := range overrides {
		override(account)
	}

	return account
}

func CreateTestAccountResponse(overrides ...func(*AccountResponse)) *AccountResponse {
	objectID := primitive.NewObjectID()
	now := time.Now()

	response := &AccountResponse{
		ID:        objectID.Hex(),
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Avatar:    "https://example.com/avatar.jpg",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, override := range overrides {
		override(response)
	}

	return response
}

func CreateTestCreateAccountRequest(overrides ...func(*CreateAccountRequest)) *CreateAccountRequest {
	req := &CreateAccountRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Avatar:    "https://example.com/avatar.jpg",
	}

	for _, override := range overrides {
		override(req)
	}

	return req
}

func CreateTestUpdateAccountRequest(overrides ...func(*UpdateAccountRequest)) *UpdateAccountRequest {
	username := "updateduser"
	firstName := "Updated"
	lastName := "User"
	avatar := "https://example.com/new-avatar.jpg"
	isActive := true

	req := &UpdateAccountRequest{
		Username:  &username,
		FirstName: &firstName,
		LastName:  &lastName,
		Avatar:    &avatar,
		IsActive:  &isActive,
	}

	for _, override := range overrides {
		override(req)
	}

	return req
}

func CreateTestListAccountsRequest(overrides ...func(*ListAccountsRequest)) *ListAccountsRequest {
	isActive := true

	req := &ListAccountsRequest{
		Page:     1,
		Limit:    10,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: &isActive,
	}

	for _, override := range overrides {
		override(req)
	}

	return req
}

func CreateTestAccountJWTClaims(overrides ...func(*AccountJWTClaims)) *AccountJWTClaims {
	claims := &AccountJWTClaims{
		AccountID: primitive.NewObjectID().Hex(),
		Email:     "test@example.com",
		Username:  "testuser",
	}

	for _, override := range overrides {
		override(claims)
	}

	return claims
}

func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}
