package account

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"

	mongoService "github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	redisService "github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/redis"
)

type MockRedisService struct {
	mock.Mock
}

func (m *MockRedisService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisService) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisService) Exists(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisService) Delete(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisService) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockRedisService) Expire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *MockRedisService) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	args := m.Called(ctx, key, members)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisService) SMembers(ctx context.Context, key string) ([]string, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRedisService) SRemove(ctx context.Context, key string, members ...interface{}) (int64, error) {
	args := m.Called(ctx, key, members)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisService) Keys(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRedisService) GetClient() *redis.Client {
	// For tests that need direct client access
	return nil
}

// Implement remaining methods with defaults or no-ops for interface compliance
func (m *MockRedisService) HealthCheck(ctx context.Context) redisService.HealthStatus {
	return redisService.HealthStatus{Connected: true}
}

func (m *MockRedisService) Close() error { return nil }
func (m *MockRedisService) GetBytes(ctx context.Context, key string) ([]byte, error) { return nil, nil }
func (m *MockRedisService) GetSet(ctx context.Context, key string, value interface{}) (string, error) { return "", nil }
func (m *MockRedisService) HSet(ctx context.Context, key, field string, value interface{}) error { return nil }
func (m *MockRedisService) HGet(ctx context.Context, key, field string) (string, error) { return "", nil }
func (m *MockRedisService) HGetAll(ctx context.Context, key string) (map[string]string, error) { return nil, nil }
func (m *MockRedisService) HDelete(ctx context.Context, key string, fields ...string) (int64, error) { return 0, nil }
func (m *MockRedisService) HExists(ctx context.Context, key, field string) (bool, error) { return false, nil }
func (m *MockRedisService) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) { return 0, nil }
func (m *MockRedisService) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) { return 0, nil }
func (m *MockRedisService) LPop(ctx context.Context, key string) (string, error) { return "", nil }
func (m *MockRedisService) RPop(ctx context.Context, key string) (string, error) { return "", nil }
func (m *MockRedisService) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) { return nil, nil }
func (m *MockRedisService) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) { return false, nil }
func (m *MockRedisService) FlushDB(ctx context.Context) error { return nil }
func (m *MockRedisService) Ping(ctx context.Context) error { return nil }
func (m *MockRedisService) Info(ctx context.Context, section ...string) (string, error) { return "", nil }

type MockMongoService struct {
	mock.Mock
}

func (m *MockMongoService) GetClient() *mongo.Client { return nil }
func (m *MockMongoService) GetDatabase() *mongo.Database { return nil }
func (m *MockMongoService) GetCollection(name string) *mongo.Collection { return &mongo.Collection{} }
func (m *MockMongoService) HealthCheck(ctx context.Context) mongoService.HealthStatus {
	return mongoService.HealthStatus{Connected: true}
}
func (m *MockMongoService) Close() error { return nil }

func TestCacheOTPRepository_CreateOTP(t *testing.T) {
	mockRedis := &MockRedisService{}
	repo := NewCacheOTPRepository(mockRedis)

	email := "test@example.com"
	purpose := OTPPurposeEmailVerification

	// Mock the Redis calls
	mockRedis.On("Delete", mock.Anything, mock.MatchedBy(func(keys []string) bool {
		return len(keys) == 3 // otp, otp_data, otp_attempts keys
	})).Return(int64(0), nil)

	mockRedis.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
		return key == "otp:test@example.com:email_verification"
	}), mock.AnythingOfType("string"), OTPExpiry).Return(nil)

	mockRedis.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
		return key == "otp_data:test@example.com:email_verification"
	}), mock.AnythingOfType("[]uint8"), OTPExpiry).Return(nil)

	mockRedis.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
		return key == "otp_attempts:test@example.com:email_verification"
	}), "0", OTPExpiry).Return(nil)

	otp, err := repo.CreateOTP(context.Background(), email, purpose)

	assert.NoError(t, err)
	assert.NotNil(t, otp)
	assert.Equal(t, email, otp.Email)
	assert.Equal(t, purpose, otp.Purpose)
	assert.Equal(t, 6, len(otp.Code)) // OTP should be 6 digits
	assert.Equal(t, 0, otp.Attempts)

	mockRedis.AssertExpectations(t)
}

func TestCacheOTPRepository_GetOTP(t *testing.T) {
	mockRedis := &MockRedisService{}
	repo := NewCacheOTPRepository(mockRedis)

	email := "test@example.com"
	purpose := OTPPurposeEmailVerification

	// Create expected OTP data
	expectedOTP := CreateTestOTP(func(otp *OTP) {
		otp.Email = email
		otp.Purpose = purpose
		otp.Attempts = 2
	})

	otpDataJSON := `{"id":"` + expectedOTP.ID.Hex() + `","email":"test@example.com","purpose":"email_verification","code":"123456","attempts":0,"expires_at":"` + expectedOTP.ExpiresAt.Format(time.RFC3339Nano) + `","created_at":"` + expectedOTP.CreatedAt.Format(time.RFC3339Nano) + `","updated_at":"` + expectedOTP.UpdatedAt.Format(time.RFC3339Nano) + `"}`

	// Mock successful data retrieval
	mockRedis.On("Get", mock.Anything, "otp_data:test@example.com:email_verification").Return(otpDataJSON, nil)
	mockRedis.On("Get", mock.Anything, "otp_attempts:test@example.com:email_verification").Return("2", nil)

	otp, err := repo.GetOTP(context.Background(), email, purpose)

	assert.NoError(t, err)
	assert.NotNil(t, otp)
	assert.Equal(t, email, otp.Email)
	assert.Equal(t, purpose, otp.Purpose)
	assert.Equal(t, 2, otp.Attempts) // Should get updated attempts count

	mockRedis.AssertExpectations(t)
}

func TestCacheOTPRepository_GetOTP_NotFound(t *testing.T) {
	mockRedis := &MockRedisService{}
	repo := NewCacheOTPRepository(mockRedis)

	email := "test@example.com"
	purpose := OTPPurposeEmailVerification

	// Mock Redis key not found
	mockRedis.On("Get", mock.Anything, "otp_data:test@example.com:email_verification").Return("", assert.AnError)
	mockRedis.On("Exists", mock.Anything, mock.MatchedBy(func(keys []string) bool {
		return len(keys) == 1 && keys[0] == "otp_data:test@example.com:email_verification"
	})).Return(int64(0), nil)

	otp, err := repo.GetOTP(context.Background(), email, purpose)

	assert.NoError(t, err)
	assert.Nil(t, otp) // Should return nil for not found

	mockRedis.AssertExpectations(t)
}

func TestCacheSessionRepository_CreateSession(t *testing.T) {
	mockRedis := &MockRedisService{}
	repo := NewCacheSessionRepository(mockRedis)

	session := CreateTestSession(func(s *Session) {
		s.TokenHash = "test_token_hash"
		s.AccountID = "test_account_id"
		s.ExpiresAt = time.Now().Add(24 * time.Hour)
	})

	// Mock Redis calls
	mockRedis.On("Set", mock.Anything, "session:test_token_hash", mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)
	mockRedis.On("SAdd", mock.Anything, "user_sessions:test_account_id", []interface{}{"test_token_hash"}).Return(int64(1), nil)
	mockRedis.On("Expire", mock.Anything, "user_sessions:test_account_id", mock.AnythingOfType("time.Duration")).Return(nil)
	mockRedis.On("Set", mock.Anything, "session_last_used:test_token_hash", mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	result, err := repo.CreateSession(context.Background(), session)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, session.TokenHash, result.TokenHash)
	assert.Equal(t, session.AccountID, result.AccountID)

	mockRedis.AssertExpectations(t)
}

func TestCacheSessionRepository_GetSessionByToken(t *testing.T) {
	mockRedis := &MockRedisService{}
	repo := NewCacheSessionRepository(mockRedis)

	tokenHash := "test_token_hash"
	expectedSession := CreateTestSession(func(s *Session) {
		s.TokenHash = tokenHash
		s.AccountID = "test_account_id"
	})

	sessionDataJSON := `{"id":"` + expectedSession.ID.Hex() + `","account_id":"test_account_id","token_hash":"test_token_hash","is_active":true,"expires_at":"` + expectedSession.ExpiresAt.Format(time.RFC3339Nano) + `","created_at":"` + expectedSession.CreatedAt.Format(time.RFC3339Nano) + `","last_used_at":"` + expectedSession.LastUsedAt.Format(time.RFC3339Nano) + `","user_agent":"Mozilla/5.0 (Test Browser)","ip_address":"192.168.1.1"}`

	// Mock successful data retrieval
	mockRedis.On("Get", mock.Anything, "session:test_token_hash").Return(sessionDataJSON, nil)
	mockRedis.On("Get", mock.Anything, "session_last_used:test_token_hash").Return("1640995200", nil) // Unix timestamp

	session, err := repo.GetSessionByToken(context.Background(), tokenHash)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, tokenHash, session.TokenHash)
	assert.Equal(t, "test_account_id", session.AccountID)

	mockRedis.AssertExpectations(t)
}

func TestCacheRepositoryIntegration(t *testing.T) {
	// Test that cache repositories work correctly with interface
	mockRedis := &MockRedisService{}
	
	// Test OTP cache repository
	otpRepo := NewCacheOTPRepository(mockRedis)
	
	email := "test@example.com"
	purpose := OTPPurposeEmailVerification

	// Mock successful OTP creation
	mockRedis.On("Delete", mock.Anything, mock.AnythingOfType("[]string")).Return(int64(0), nil)
	mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil).Times(3)

	otp, err := otpRepo.CreateOTP(context.Background(), email, purpose)

	assert.NoError(t, err)
	assert.NotNil(t, otp)
	assert.Equal(t, email, otp.Email)
	assert.Equal(t, purpose, otp.Purpose)

	// Verify that OTP operations work on cache repository
	assert.IsType(t, &CacheOTPRepository{}, otpRepo)

	mockRedis.AssertExpectations(t)
}

func TestAccountModule_CacheConfiguration(t *testing.T) {
	// Test default configuration
	module := NewAccountModule("test@platform.com")
	assert.True(t, module.useCacheForOTP)
	assert.True(t, module.useCacheForSession)

	// Test custom configuration
	module = module.WithCacheConfig(false, true)
	assert.False(t, module.useCacheForOTP)
	assert.True(t, module.useCacheForSession)
}