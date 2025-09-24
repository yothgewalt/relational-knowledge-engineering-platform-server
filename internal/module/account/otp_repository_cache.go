package account

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/redis"
)

type CacheOTPRepository struct {
	cacheService redis.RedisService
}

var _ OTPRepository = (*CacheOTPRepository)(nil)

func NewCacheOTPRepository(cacheService redis.RedisService) *CacheOTPRepository {
	return &CacheOTPRepository{
		cacheService: cacheService,
	}
}

func (r *CacheOTPRepository) otpKey(email string, purpose OTPPurpose) string {
	return fmt.Sprintf("otp:%s:%s", email, purpose)
}

func (r *CacheOTPRepository) otpDataKey(email string, purpose OTPPurpose) string {
	return fmt.Sprintf("otp_data:%s:%s", email, purpose)
}

func (r *CacheOTPRepository) otpAttemptKey(email string, purpose OTPPurpose) string {
	return fmt.Sprintf("otp_attempts:%s:%s", email, purpose)
}

func (r *CacheOTPRepository) CreateOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error) {
	if err := r.DeleteOTP(ctx, email, purpose); err != nil {
		return nil, fmt.Errorf("failed to cleanup existing OTP: %w", err)
	}

	code, err := generateOTPCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP code: %w", err)
	}

	now := time.Now()
	otp := &OTP{
		ID:        primitive.NewObjectID(),
		Email:     email,
		Purpose:   purpose,
		Code:      code,
		Attempts:  0,
		ExpiresAt: now.Add(OTPExpiry),
		CreatedAt: now,
		UpdatedAt: now,
	}

	otpData, err := json.Marshal(otp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OTP data: %w", err)
	}

	otpKey := r.otpKey(email, purpose)
	if err := r.cacheService.Set(ctx, otpKey, code, OTPExpiry); err != nil {
		return nil, fmt.Errorf("failed to store OTP code: %w", err)
	}

	otpDataKey := r.otpDataKey(email, purpose)
	if err := r.cacheService.Set(ctx, otpDataKey, otpData, OTPExpiry); err != nil {
		return nil, fmt.Errorf("failed to store OTP data: %w", err)
	}

	otpAttemptKey := r.otpAttemptKey(email, purpose)
	if err := r.cacheService.Set(ctx, otpAttemptKey, "0", OTPExpiry); err != nil {
		return nil, fmt.Errorf("failed to initialize OTP attempts: %w", err)
	}

	return otp, nil
}

func (r *CacheOTPRepository) GetOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error) {
	otpDataKey := r.otpDataKey(email, purpose)

	otpDataStr, err := r.cacheService.Get(ctx, otpDataKey)
	if err != nil {
		if exists, existsErr := r.cacheService.Exists(ctx, otpDataKey); existsErr == nil && exists == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get OTP data: %w", err)
	}

	var otp OTP
	if err := json.Unmarshal([]byte(otpDataStr), &otp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OTP data: %w", err)
	}

	otpAttemptKey := r.otpAttemptKey(email, purpose)
	attemptsStr, err := r.cacheService.Get(ctx, otpAttemptKey)
	if err != nil {
		otp.Attempts = 0
	} else {
		attempts, parseErr := strconv.Atoi(attemptsStr)
		if parseErr != nil {
			otp.Attempts = 0
		} else {
			otp.Attempts = attempts
		}
	}

	return &otp, nil
}

func (r *CacheOTPRepository) ValidateOTP(ctx context.Context, email string, purpose OTPPurpose, code string) (*OTP, error) {
	otp, err := r.GetOTP(ctx, email, purpose)
	if err != nil {
		return nil, err
	}

	if otp == nil {
		return nil, fmt.Errorf("OTP not found")
	}

	if otp.IsExpired() {
		r.DeleteOTP(ctx, email, purpose)
		return nil, fmt.Errorf("OTP has expired")
	}

	if otp.IsMaxAttemptsReached() {
		r.DeleteOTP(ctx, email, purpose)
		return nil, fmt.Errorf("maximum OTP attempts reached")
	}

	otpKey := r.otpKey(email, purpose)
	storedCode, err := r.cacheService.Get(ctx, otpKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get OTP code for validation: %w", err)
	}

	if storedCode != code {
		if err := r.incrementOTPAttemptsByEmailAndPurpose(ctx, email, purpose); err != nil {
			return nil, fmt.Errorf("failed to increment OTP attempts: %w", err)
		}
		return nil, fmt.Errorf("invalid OTP code")
	}

	return otp, nil
}

func (r *CacheOTPRepository) IncrementOTPAttempts(ctx context.Context, id primitive.ObjectID) error {
	return fmt.Errorf("IncrementOTPAttempts by ID not supported in cache implementation - use IncrementOTPAttemptsByEmailAndPurpose")
}

func (r *CacheOTPRepository) incrementOTPAttemptsByEmailAndPurpose(ctx context.Context, email string, purpose OTPPurpose) error {
	otpAttemptKey := r.otpAttemptKey(email, purpose)

	client := r.cacheService.GetClient()
	newAttempts, err := client.Incr(ctx, otpAttemptKey).Result()
	if err != nil {
		return fmt.Errorf("failed to increment OTP attempts: %w", err)
	}

	if err := r.cacheService.Expire(ctx, otpAttemptKey, OTPExpiry); err != nil {
		return fmt.Errorf("failed to set expiration for OTP attempts: %w", err)
	}

	if int(newAttempts) >= MaxOTPAttempts {
		r.DeleteOTP(ctx, email, purpose)
	}

	return nil
}

func (r *CacheOTPRepository) DeleteOTP(ctx context.Context, email string, purpose OTPPurpose) error {
	keys := []string{
		r.otpKey(email, purpose),
		r.otpDataKey(email, purpose),
		r.otpAttemptKey(email, purpose),
	}

	deleted, err := r.cacheService.Delete(ctx, keys...)
	if err != nil {
		return fmt.Errorf("failed to delete OTP keys: %w", err)
	}

	_ = deleted

	return nil
}

func (r *CacheOTPRepository) CleanupExpiredOTPs(ctx context.Context) error {
	return nil
}



