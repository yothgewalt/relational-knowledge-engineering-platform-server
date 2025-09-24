package account

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/redis"
)

type HybridAccountIdentityRepository struct {
	otpRepo            OTPRepository
	sessionRepo        SessionRepository
	mongoOTPRepo       OTPRepository
	mongoSessionRepo   SessionRepository
	useCacheForOTP     bool
	useCacheForSession bool
}

type HybridRepositoryConfig struct {
	UseCacheForOTP     bool
	UseCacheForSession bool
	EnableFallback     bool
}

func NewHybridAccountIdentityRepository(
	mongoService *mongo.MongoService,
	cacheService redis.RedisService,
	config HybridRepositoryConfig,
) AccountIdentityRepository {
	mongoRepo := NewAccountIdentityRepository(mongoService)

	hybrid := &HybridAccountIdentityRepository{
		mongoOTPRepo:       mongoRepo,
		mongoSessionRepo:   mongoRepo,
		useCacheForOTP:     config.UseCacheForOTP,
		useCacheForSession: config.UseCacheForSession,
	}

	if config.UseCacheForOTP && cacheService != nil {
		hybrid.otpRepo = NewCacheOTPRepository(cacheService)
	} else {
		hybrid.otpRepo = mongoRepo
	}

	// Configure Session repository
	if config.UseCacheForSession && cacheService != nil {
		hybrid.sessionRepo = NewCacheSessionRepository(cacheService)
	} else {
		hybrid.sessionRepo = mongoRepo
	}

	return hybrid
}

func (h *HybridAccountIdentityRepository) CreateOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error) {
	otp, err := h.otpRepo.CreateOTP(ctx, email, purpose)
	if err != nil && h.useCacheForOTP {
		return h.mongoOTPRepo.CreateOTP(ctx, email, purpose)
	}
	return otp, err
}

func (h *HybridAccountIdentityRepository) GetOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error) {
	otp, err := h.otpRepo.GetOTP(ctx, email, purpose)
	if err != nil && h.useCacheForOTP {
		return h.mongoOTPRepo.GetOTP(ctx, email, purpose)
	}
	return otp, err
}

func (h *HybridAccountIdentityRepository) ValidateOTP(ctx context.Context, email string, purpose OTPPurpose, code string) (*OTP, error) {
	otp, err := h.otpRepo.ValidateOTP(ctx, email, purpose, code)
	if err != nil && h.useCacheForOTP {
		return h.mongoOTPRepo.ValidateOTP(ctx, email, purpose, code)
	}
	return otp, err
}

func (h *HybridAccountIdentityRepository) IncrementOTPAttempts(ctx context.Context, id primitive.ObjectID) error {
	err := h.otpRepo.IncrementOTPAttempts(ctx, id)
	if err != nil && h.useCacheForOTP {
		return h.mongoOTPRepo.IncrementOTPAttempts(ctx, id)
	}
	return err
}

func (h *HybridAccountIdentityRepository) DeleteOTP(ctx context.Context, email string, purpose OTPPurpose) error {
	err := h.otpRepo.DeleteOTP(ctx, email, purpose)
	if err != nil && h.useCacheForOTP {
		return h.mongoOTPRepo.DeleteOTP(ctx, email, purpose)
	}
	return err
}

func (h *HybridAccountIdentityRepository) CleanupExpiredOTPs(ctx context.Context) error {
	var errs []error

	if err := h.otpRepo.CleanupExpiredOTPs(ctx); err != nil {
		errs = append(errs, fmt.Errorf("cache OTP cleanup failed: %w", err))
	}

	if h.useCacheForOTP {
		if err := h.mongoOTPRepo.CleanupExpiredOTPs(ctx); err != nil {
			errs = append(errs, fmt.Errorf("mongo OTP cleanup failed: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}

func (h *HybridAccountIdentityRepository) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	result, err := h.sessionRepo.CreateSession(ctx, session)
	if err != nil && h.useCacheForSession {
		return h.mongoSessionRepo.CreateSession(ctx, session)
	}
	return result, err
}

func (h *HybridAccountIdentityRepository) GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error) {
	session, err := h.sessionRepo.GetSessionByToken(ctx, tokenHash)
	if err != nil && h.useCacheForSession {
		return h.mongoSessionRepo.GetSessionByToken(ctx, tokenHash)
	}
	return session, err
}

func (h *HybridAccountIdentityRepository) GetSessionsByAccountID(ctx context.Context, accountID string) ([]*Session, error) {
	sessions, err := h.sessionRepo.GetSessionsByAccountID(ctx, accountID)
	if err != nil && h.useCacheForSession {
		return h.mongoSessionRepo.GetSessionsByAccountID(ctx, accountID)
	}
	return sessions, err
}

func (h *HybridAccountIdentityRepository) UpdateSessionLastUsed(ctx context.Context, id primitive.ObjectID) error {
	err := h.sessionRepo.UpdateSessionLastUsed(ctx, id)
	if err != nil && h.useCacheForSession {
		return h.mongoSessionRepo.UpdateSessionLastUsed(ctx, id)
	}
	return err
}

func (h *HybridAccountIdentityRepository) DeactivateSession(ctx context.Context, tokenHash string) error {
	err := h.sessionRepo.DeactivateSession(ctx, tokenHash)
	if err != nil && h.useCacheForSession {
		return h.mongoSessionRepo.DeactivateSession(ctx, tokenHash)
	}
	return err
}

func (h *HybridAccountIdentityRepository) DeactivateAllUserSessions(ctx context.Context, accountID string) error {
	err := h.sessionRepo.DeactivateAllUserSessions(ctx, accountID)
	if err != nil && h.useCacheForSession {
		return h.mongoSessionRepo.DeactivateAllUserSessions(ctx, accountID)
	}
	return err
}

func (h *HybridAccountIdentityRepository) CleanupExpiredSessions(ctx context.Context) error {
	var errs []error

	if err := h.sessionRepo.CleanupExpiredSessions(ctx); err != nil {
		errs = append(errs, fmt.Errorf("cache session cleanup failed: %w", err))
	}

	if h.useCacheForSession {
		if err := h.mongoSessionRepo.CleanupExpiredSessions(ctx); err != nil {
			errs = append(errs, fmt.Errorf("mongo session cleanup failed: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}


