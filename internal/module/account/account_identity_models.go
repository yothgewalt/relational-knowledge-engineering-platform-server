package account

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginResponse struct {
	Token   string           `json:"token"`
	Account *AccountResponse `json:"account"`
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
	Password  string `json:"password" validate:"required,min=8"`
	Avatar    string `json:"avatar"`
}

type RegisterResponse struct {
	Account *AccountResponse `json:"account"`
	Message string           `json:"message"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	OTP         string `json:"otp" validate:"required,len=6"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type ValidateTokenResponse struct {
	Valid   bool               `json:"valid"`
	Claims  *AccountJWTClaims  `json:"claims,omitempty"`
	Account *AccountResponse   `json:"account,omitempty"`
}

type RefreshTokenResponse struct {
	Token string `json:"token"`
}

type MeResponse struct {
	Account *AccountResponse `json:"account"`
	Session *SessionInfo     `json:"session"`
}

type SessionInfo struct {
	ID         string    `json:"id"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
}

type OTPPurpose string

const (
	OTPPurposeEmailVerification OTPPurpose = "email_verification"
	OTPPurposePasswordReset     OTPPurpose = "password_reset"
)

type OTP struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email     string             `json:"email" bson:"email"`
	Purpose   OTPPurpose         `json:"purpose" bson:"purpose"`
	Code      string             `json:"code" bson:"code"`
	Attempts  int                `json:"attempts" bson:"attempts"`
	ExpiresAt time.Time          `json:"expires_at" bson:"expires_at"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

type Session struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AccountID   string             `json:"account_id" bson:"account_id"`
	TokenHash   string             `json:"token_hash" bson:"token_hash"`
	IsActive    bool               `json:"is_active" bson:"is_active"`
	ExpiresAt   time.Time          `json:"expires_at" bson:"expires_at"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	LastUsedAt  time.Time          `json:"last_used_at" bson:"last_used_at"`
	UserAgent   string             `json:"user_agent" bson:"user_agent"`
	IPAddress   string             `json:"ip_address" bson:"ip_address"`
}

type EmailTemplate struct {
	Subject  string
	HtmlBody string
	TextBody string
}

const (
	MaxOTPAttempts = 5
	OTPLength      = 6
	OTPExpiry      = 5 * time.Minute
)

func (otp *OTP) IsExpired() bool {
	return time.Now().After(otp.ExpiresAt)
}

func (otp *OTP) IsMaxAttemptsReached() bool {
	return otp.Attempts >= MaxOTPAttempts
}

func (session *Session) IsExpired() bool {
	return time.Now().After(session.ExpiresAt)
}

func (session *Session) ToSessionInfo() *SessionInfo {
	return &SessionInfo{
		ID:         session.ID.Hex(),
		ExpiresAt:  session.ExpiresAt,
		LastUsedAt: session.LastUsedAt,
		UserAgent:  session.UserAgent,
		IPAddress:  session.IPAddress,
	}
}

func GetWelcomeEmailTemplate(username string) EmailTemplate {
	return EmailTemplate{
		Subject:  "Welcome to Our Platform!",
		HtmlBody: "<h1>Welcome " + username + "!</h1><p>Thank you for joining our platform. Please verify your email to get started.</p>",
		TextBody: "Welcome " + username + "! Thank you for joining our platform. Please verify your email to get started.",
	}
}

func GetEmailVerificationTemplate(otp string) EmailTemplate {
	return EmailTemplate{
		Subject:  "Email Verification Code",
		HtmlBody: "<h1>Email Verification</h1><p>Your verification code is: <strong>" + otp + "</strong></p><p>This code will expire in 5 minutes.</p>",
		TextBody: "Your email verification code is: " + otp + ". This code will expire in 5 minutes.",
	}
}

func GetPasswordResetTemplate(otp string) EmailTemplate {
	return EmailTemplate{
		Subject:  "Password Reset Code",
		HtmlBody: "<h1>Password Reset</h1><p>Your password reset code is: <strong>" + otp + "</strong></p><p>This code will expire in 5 minutes.</p>",
		TextBody: "Password Reset - Your password reset code is: " + otp + ". This code will expire in 5 minutes.",
	}
}

func GetPasswordChangeConfirmationTemplate() EmailTemplate {
	return EmailTemplate{
		Subject:  "Password Changed Successfully",
		HtmlBody: "<h1>Password Changed</h1><p>Your password has been successfully changed. If you did not make this change, please contact support immediately.</p>",
		TextBody: "Password Changed - Your password has been successfully changed. If you did not make this change, please contact support immediately.",
	}
}