package account

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
)

const (
	OTPCollectionName     = "otps"
	SessionCollectionName = "sessions"
)


type accountIdentityRepository struct {
	otpRepo     mongo.Repository[OTP]
	sessionRepo mongo.Repository[Session]
}

var _ AccountIdentityRepository = (*accountIdentityRepository)(nil)

func NewAccountIdentityRepository(mongoService *mongo.MongoService) AccountIdentityRepository {
	return &accountIdentityRepository{
		otpRepo:     mongo.NewRepository[OTP](mongoService, OTPCollectionName),
		sessionRepo: mongo.NewRepository[Session](mongoService, SessionCollectionName),
	}
}

func (r *accountIdentityRepository) CreateOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error) {
	r.DeleteOTP(ctx, email, purpose)

	code, err := generateOTPCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP code: %w", err)
	}

	otp := &OTP{
		ID:        primitive.NewObjectID(),
		Email:     email,
		Purpose:   purpose,
		Code:      code,
		Attempts:  0,
		ExpiresAt: time.Now().Add(OTPExpiry),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := r.otpRepo.Create(ctx, *otp)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTP: %w", err)
	}

	return result, nil
}

func (r *accountIdentityRepository) GetOTP(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error) {
	filter := bson.M{
		"email":   email,
		"purpose": purpose,
	}

	otp, err := r.otpRepo.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get OTP: %w", err)
	}

	if otp == nil {
		return nil, nil
	}

	return otp, nil
}

func (r *accountIdentityRepository) ValidateOTP(ctx context.Context, email string, purpose OTPPurpose, code string) (*OTP, error) {
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

	if otp.Code != code {
		r.IncrementOTPAttempts(ctx, otp.ID)
		return nil, fmt.Errorf("invalid OTP code")
	}

	return otp, nil
}

func (r *accountIdentityRepository) IncrementOTPAttempts(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$inc": bson.M{"attempts": 1},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := r.otpRepo.Update(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment OTP attempts: %w", err)
	}

	return nil
}

func (r *accountIdentityRepository) DeleteOTP(ctx context.Context, email string, purpose OTPPurpose) error {
	filter := bson.M{
		"email":   email,
		"purpose": purpose,
	}

	err := r.otpRepo.Delete(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete OTP: %w", err)
	}

	return nil
}

func (r *accountIdentityRepository) CleanupExpiredOTPs(ctx context.Context) error {
	filter := bson.M{
		"expires_at": bson.M{"$lte": time.Now()},
	}

	opts := options.Find()
	expiredOTPs, err := r.otpRepo.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("failed to find expired OTPs: %w", err)
	}

	for _, otp := range expiredOTPs {
		r.DeleteOTP(ctx, otp.Email, otp.Purpose)
	}

	return nil
}

func (r *accountIdentityRepository) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	session.ID = primitive.NewObjectID()
	session.IsActive = true
	session.CreatedAt = time.Now()
	session.LastUsedAt = time.Now()

	result, err := r.sessionRepo.Create(ctx, *session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return result, nil
}

func (r *accountIdentityRepository) GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error) {
	filter := bson.M{
		"token_hash": tokenHash,
		"is_active":  true,
	}

	session, err := r.sessionRepo.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, nil
	}

	return session, nil
}

func (r *accountIdentityRepository) GetSessionsByAccountID(ctx context.Context, accountID string) ([]*Session, error) {
	filter := bson.M{
		"account_id": accountID,
		"is_active":  true,
	}

	opts := options.Find().SetSort(bson.D{{Key: "last_used_at", Value: -1}})
	sessions, err := r.sessionRepo.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	result := make([]*Session, len(sessions))
	for i, session := range sessions {
		s := session
		result[i] = &s
	}

	return result, nil
}

func (r *accountIdentityRepository) UpdateSessionLastUsed(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{"last_used_at": time.Now()},
	}

	_, err := r.sessionRepo.Update(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update session last used: %w", err)
	}

	return nil
}

func (r *accountIdentityRepository) DeactivateSession(ctx context.Context, tokenHash string) error {
	filter := bson.M{"token_hash": tokenHash}
	update := bson.M{
		"$set": bson.M{
			"is_active": false,
			"last_used_at": time.Now(),
		},
	}

	_, err := r.sessionRepo.Update(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate session: %w", err)
	}

	return nil
}

func (r *accountIdentityRepository) DeactivateAllUserSessions(ctx context.Context, accountID string) error {
	filter := bson.M{"account_id": accountID}
	update := bson.M{
		"$set": bson.M{
			"is_active": false,
			"last_used_at": time.Now(),
		},
	}

	sessions, err := r.sessionRepo.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find user sessions: %w", err)
	}

	for _, session := range sessions {
		sessionFilter := bson.M{"_id": session.ID}
		_, err := r.sessionRepo.Update(ctx, sessionFilter, update)
		if err != nil {
			continue
		}
	}

	return nil
}

func (r *accountIdentityRepository) CleanupExpiredSessions(ctx context.Context) error {
	filter := bson.M{
		"$or": []bson.M{
			{"expires_at": bson.M{"$lte": time.Now()}},
			{"is_active": false},
		},
	}

	sessions, err := r.sessionRepo.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find expired sessions: %w", err)
	}

	for _, session := range sessions {
		sessionFilter := bson.M{"_id": session.ID}
		r.sessionRepo.Delete(ctx, sessionFilter)
	}

	return nil
}

func generateOTPCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, OTPLength)
	
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}
	
	return string(code), nil
}

