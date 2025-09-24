package account

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OTPRepository interface {
	CreateOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error)
	GetOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error)
	ValidateOTP(ctx context.Context, email string, purpose OTPPurpose, code string) (*OTP, error)
	IncrementOTPAttempts(ctx context.Context, id primitive.ObjectID) error
	DeleteOTP(ctx context.Context, email string, purpose OTPPurpose) error
	CleanupExpiredOTPs(ctx context.Context) error
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error)
	GetSessionsByAccountID(ctx context.Context, accountID string) ([]*Session, error)
	UpdateSessionLastUsed(ctx context.Context, id primitive.ObjectID) error
	DeactivateSession(ctx context.Context, tokenHash string) error
	DeactivateAllUserSessions(ctx context.Context, accountID string) error
	CleanupExpiredSessions(ctx context.Context) error
}

type AccountIdentityRepository interface {
	OTPRepository
	SessionRepository
}