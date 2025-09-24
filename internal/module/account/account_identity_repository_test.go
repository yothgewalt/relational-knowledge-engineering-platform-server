package account

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
)

type MockMongoRepository[T any] struct {
	mock.Mock
}

func (m *MockMongoRepository[T]) Create(ctx context.Context, entity T) (*T, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*T), args.Error(1)
}

func (m *MockMongoRepository[T]) FindOne(ctx context.Context, filter bson.M, opts ...*options.FindOneOptions) (*T, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*T), args.Error(1)
}

func (m *MockMongoRepository[T]) Find(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]T, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]T), args.Error(1)
}

func (m *MockMongoRepository[T]) Update(ctx context.Context, filter bson.M, update bson.M, opts ...*options.UpdateOptions) (*T, error) {
	args := m.Called(ctx, filter, update, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*T), args.Error(1)
}

func (m *MockMongoRepository[T]) Delete(ctx context.Context, filter bson.M, opts ...*options.DeleteOptions) error {
	args := m.Called(ctx, filter, opts)
	return args.Error(0)
}

func (m *MockMongoRepository[T]) FindWithPagination(ctx context.Context, filter bson.M, pagination mongo.PaginationOptions, opts ...*options.FindOptions) (*mongo.PaginatedResult[T], error) {
	args := m.Called(ctx, filter, pagination, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.PaginatedResult[T]), args.Error(1)
}

func (m *MockMongoRepository[T]) Count(ctx context.Context, filter bson.M, opts ...*options.CountOptions) (int64, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(int64), args.Error(1)
}

func setupAccountIdentityRepository() (*accountIdentityRepository, *MockMongoRepository[OTP], *MockMongoRepository[Session]) {
	mockOTPRepo := &MockMongoRepository[OTP]{}
	mockSessionRepo := &MockMongoRepository[Session]{}

	repo := &accountIdentityRepository{
		otpRepo:     mockOTPRepo,
		sessionRepo: mockSessionRepo,
	}

	return repo, mockOTPRepo, mockSessionRepo
}

func TestNewAccountIdentityRepository(t *testing.T) {
	mongoService := &mongo.MongoService{}
	repo := NewAccountIdentityRepository(mongoService)
	assert.NotNil(t, repo)
	assert.IsType(t, &accountIdentityRepository{}, repo)
}

func TestAccountIdentityRepository_CreateOTP(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		purpose OTPPurpose
		setup   func(*MockMongoRepository[OTP])
		wantErr bool
	}{
		{
			name:    "successful OTP creation",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("Delete", mock.Anything, bson.M{
					"email":   "test@example.com",
					"purpose": OTPPurposeEmailVerification,
				}, mock.Anything).Return(nil)

				otp := CreateTestOTP(func(o *OTP) {
					o.Email = "test@example.com"
					o.Purpose = OTPPurposeEmailVerification
				})
				mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(o OTP) bool {
					return o.Email == "test@example.com" && o.Purpose == OTPPurposeEmailVerification
				})).Return(otp, nil)
			},
			wantErr: false,
		},
		{
			name:    "OTP creation fails",
			email:   "test@example.com",
			purpose: OTPPurposePasswordReset,
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mockOTPRepo, _ := setupAccountIdentityRepository()
			tt.setup(mockOTPRepo)

			result, err := repo.CreateOTP(context.Background(), tt.email, tt.purpose)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.email, result.Email)
				assert.Equal(t, tt.purpose, result.Purpose)
				assert.Equal(t, 6, len(result.Code))
			}

			mockOTPRepo.AssertExpectations(t)
		})
	}
}

func TestAccountIdentityRepository_GetOTP(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		purpose  OTPPurpose
		setup    func(*MockMongoRepository[OTP])
		expected *OTP
		wantErr  bool
	}{
		{
			name:    "OTP found",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				otp := CreateTestOTP()
				mockRepo.On("FindOne", mock.Anything, bson.M{
					"email":   "test@example.com",
					"purpose": OTPPurposeEmailVerification,
				}, mock.Anything).Return(otp, nil)
			},
			expected: CreateTestOTP(),
			wantErr:  false,
		},
		{
			name:    "OTP not found",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name:    "database error",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mockOTPRepo, _ := setupAccountIdentityRepository()
			tt.setup(mockOTPRepo)

			result, err := repo.GetOTP(context.Background(), tt.email, tt.purpose)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected == nil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
				}
			}

			mockOTPRepo.AssertExpectations(t)
		})
	}
}

func TestAccountIdentityRepository_ValidateOTP(t *testing.T) {
	validOTP := CreateTestOTP(func(o *OTP) {
		o.Code = "123456"
		o.Attempts = 2
		o.ExpiresAt = time.Now().Add(5 * time.Minute)
	})

	expiredOTP := CreateTestOTP(func(o *OTP) {
		o.Code = "123456"
		o.ExpiresAt = time.Now().Add(-1 * time.Minute)
	})

	maxAttemptsOTP := CreateTestOTP(func(o *OTP) {
		o.Code = "123456"
		o.Attempts = MaxOTPAttempts
		o.ExpiresAt = time.Now().Add(5 * time.Minute)
	})

	tests := []struct {
		name    string
		email   string
		purpose OTPPurpose
		code    string
		setup   func(*MockMongoRepository[OTP])
		wantErr bool
	}{
		{
			name:    "valid OTP",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			code:    "123456",
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything).Return(validOTP, nil)
			},
			wantErr: false,
		},
		{
			name:    "OTP not found",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			code:    "123456",
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name:    "expired OTP",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			code:    "123456",
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything).Return(expiredOTP, nil)
				mockRepo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: true,
		},
		{
			name:    "max attempts reached",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			code:    "123456",
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything).Return(maxAttemptsOTP, nil)
				mockRepo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: true,
		},
		{
			name:    "invalid code",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			code:    "wrong",
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything).Return(validOTP, nil)
				mockRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(validOTP, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mockOTPRepo, _ := setupAccountIdentityRepository()
			tt.setup(mockOTPRepo)

			result, err := repo.ValidateOTP(context.Background(), tt.email, tt.purpose, tt.code)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.name != "valid OTP" {
					assert.Nil(t, result)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockOTPRepo.AssertExpectations(t)
		})
	}
}

func TestAccountIdentityRepository_CreateSession(t *testing.T) {
	tests := []struct {
		name    string
		session *Session
		setup   func(*MockMongoRepository[Session])
		wantErr bool
	}{
		{
			name:    "successful session creation",
			session: CreateTestSession(),
			setup: func(mockRepo *MockMongoRepository[Session]) {
				session := CreateTestSession()
				mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(s Session) bool {
					return s.IsActive == true
				})).Return(session, nil)
			},
			wantErr: false,
		},
		{
			name:    "session creation fails",
			session: CreateTestSession(),
			setup: func(mockRepo *MockMongoRepository[Session]) {
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, mockSessionRepo := setupAccountIdentityRepository()
			tt.setup(mockSessionRepo)

			result, err := repo.CreateSession(context.Background(), tt.session)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockSessionRepo.AssertExpectations(t)
		})
	}
}

func TestAccountIdentityRepository_GetSessionByToken(t *testing.T) {
	tests := []struct {
		name      string
		tokenHash string
		setup     func(*MockMongoRepository[Session])
		expected  *Session
		wantErr   bool
	}{
		{
			name:      "session found",
			tokenHash: "hashedtoken123",
			setup: func(mockRepo *MockMongoRepository[Session]) {
				session := CreateTestSession()
				mockRepo.On("FindOne", mock.Anything, bson.M{
					"token_hash": "hashedtoken123",
					"is_active":  true,
				}).Return(session, nil)
			},
			expected: CreateTestSession(),
			wantErr:  false,
		},
		{
			name:      "session not found",
			tokenHash: "nonexistent",
			setup: func(mockRepo *MockMongoRepository[Session]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name:      "database error",
			tokenHash: "hashedtoken123",
			setup: func(mockRepo *MockMongoRepository[Session]) {
				mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, mockSessionRepo := setupAccountIdentityRepository()
			tt.setup(mockSessionRepo)

			result, err := repo.GetSessionByToken(context.Background(), tt.tokenHash)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected == nil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
				}
			}

			mockSessionRepo.AssertExpectations(t)
		})
	}
}

func TestAccountIdentityRepository_DeactivateSession(t *testing.T) {
	tests := []struct {
		name      string
		tokenHash string
		setup     func(*MockMongoRepository[Session])
		wantErr   bool
	}{
		{
			name:      "successful deactivation",
			tokenHash: "hashedtoken123",
			setup: func(mockRepo *MockMongoRepository[Session]) {
				session := CreateTestSession(func(s *Session) {
					s.IsActive = false
				})
				mockRepo.On("Update", mock.Anything,
					bson.M{"token_hash": "hashedtoken123"},
					mock.MatchedBy(func(update bson.M) bool {
						set, ok := update["$set"].(bson.M)
						if !ok {
							return false
						}
						return set["is_active"] == false
					})).Return(session, nil)
			},
			wantErr: false,
		},
		{
			name:      "deactivation fails",
			tokenHash: "hashedtoken123",
			setup: func(mockRepo *MockMongoRepository[Session]) {
				mockRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, mockSessionRepo := setupAccountIdentityRepository()
			tt.setup(mockSessionRepo)

			err := repo.DeactivateSession(context.Background(), tt.tokenHash)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockSessionRepo.AssertExpectations(t)
		})
	}
}

func TestAccountIdentityRepository_DeleteOTP(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		purpose OTPPurpose
		setup   func(*MockMongoRepository[OTP])
		wantErr bool
	}{
		{
			name:    "successful deletion",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("Delete", mock.Anything, bson.M{
					"email":   "test@example.com",
					"purpose": OTPPurposeEmailVerification,
				}, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "deletion fails",
			email:   "test@example.com",
			purpose: OTPPurposeEmailVerification,
			setup: func(mockRepo *MockMongoRepository[OTP]) {
				mockRepo.On("Delete", mock.Anything, mock.Anything).Return(fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mockOTPRepo, _ := setupAccountIdentityRepository()
			tt.setup(mockOTPRepo)

			err := repo.DeleteOTP(context.Background(), tt.email, tt.purpose)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockOTPRepo.AssertExpectations(t)
		})
	}
}

func TestAccountIdentityRepository_UpdateSessionLastUsed(t *testing.T) {
	tests := []struct {
		name    string
		id      primitive.ObjectID
		setup   func(*MockMongoRepository[Session])
		wantErr bool
	}{
		{
			name: "successful update",
			id:   primitive.NewObjectID(),
			setup: func(mockRepo *MockMongoRepository[Session]) {
				session := CreateTestSession()
				mockRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(session, nil)
			},
			wantErr: false,
		},
		{
			name: "update fails",
			id:   primitive.NewObjectID(),
			setup: func(mockRepo *MockMongoRepository[Session]) {
				mockRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, mockSessionRepo := setupAccountIdentityRepository()
			tt.setup(mockSessionRepo)

			err := repo.UpdateSessionLastUsed(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockSessionRepo.AssertExpectations(t)
		})
	}
}

func TestGenerateOTPCode(t *testing.T) {
	for i := 0; i < 10; i++ {
		code, err := generateOTPCode()
		assert.NoError(t, err)
		assert.Len(t, code, OTPLength)

		// Check all characters are digits
		for _, char := range code {
			assert.True(t, char >= '0' && char <= '9', "OTP should contain only digits")
		}
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "otps", OTPCollectionName)
	assert.Equal(t, "sessions", SessionCollectionName)
}
