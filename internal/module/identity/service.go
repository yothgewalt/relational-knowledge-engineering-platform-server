package identity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/account"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/jwt"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
)

type IdentityService interface {
	Login(ctx context.Context, req *LoginRequest, userAgent, ipAddress string) (*LoginResponse, error)
	Logout(ctx context.Context, token string) error
	Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error)
	VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error
	ResendEmailVerification(ctx context.Context, req *ResendVerificationRequest) error
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) error
	ChangePassword(ctx context.Context, accountID string, req *ChangePasswordRequest) error
	ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error)
	RefreshToken(ctx context.Context, token string, userAgent, ipAddress string) (*RefreshTokenResponse, error)
}

type identityService struct {
	repository     IdentityRepository
	accountService account.AccountService
	jwtService     *jwt.JWTService
	resendService  resend.ResendService
	fromEmail      string
}

func NewIdentityService(
	repository IdentityRepository,
	accountService account.AccountService,
	jwtService *jwt.JWTService,
	resendService resend.ResendService,
	fromEmail string,
) IdentityService {
	return &identityService{
		repository:     repository,
		accountService: accountService,
		jwtService:     jwtService,
		resendService:  resendService,
		fromEmail:      fromEmail,
	}
}

func (s *identityService) Login(ctx context.Context, req *LoginRequest, userAgent, ipAddress string) (*LoginResponse, error) {
	accountResp, err := s.accountService.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !accountResp.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(req.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	claims := &account.AccountJWTClaims{
		AccountID: accountResp.ID,
		Email:     accountResp.Email,
		Username:  accountResp.Username,
	}

	token, err := s.jwtService.Generate(claims.ToCustomClaims())
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	session := &Session{
		AccountID: accountResp.ID,
		TokenHash: s.hashToken(token),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	_, err = s.repository.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &LoginResponse{
		Token:   token,
		Account: accountResp,
	}, nil
}

func (s *identityService) Logout(ctx context.Context, token string) error {
	tokenHash := s.hashToken(token)
	
	err := s.repository.DeactivateSession(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	return nil
}

func (s *identityService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	_, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	createAccountReq := &account.CreateAccountRequest{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Avatar:    req.Avatar,
	}

	accountResp, err := s.accountService.CreateAccount(ctx, createAccountReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	otp, err := s.repository.CreateOTP(ctx, req.Email, OTPPurposeEmailVerification)
	if err != nil {
		return nil, fmt.Errorf("failed to create email verification OTP: %w", err)
	}

	if s.resendService != nil {
		welcomeTemplate := GetWelcomeEmailTemplate(req.Username)
		welcomeEmailReq := &resend.EmailRequest{
			From:    s.fromEmail,
			To:      []string{req.Email},
			Subject: welcomeTemplate.Subject,
			Html:    welcomeTemplate.HtmlBody,
			Text:    welcomeTemplate.TextBody,
		}

		_, err = s.resendService.SendEmail(ctx, welcomeEmailReq)
		if err != nil {
			fmt.Printf("Failed to send welcome email: %v\n", err)
		}

		verificationTemplate := GetEmailVerificationTemplate(otp.Code)
		verificationEmailReq := &resend.EmailRequest{
			From:    s.fromEmail,
			To:      []string{req.Email},
			Subject: verificationTemplate.Subject,
			Html:    verificationTemplate.HtmlBody,
			Text:    verificationTemplate.TextBody,
		}

		_, err = s.resendService.SendEmail(ctx, verificationEmailReq)
		if err != nil {
			return nil, fmt.Errorf("failed to send verification email: %w", err)
		}
	}

	return &RegisterResponse{
		Account: accountResp,
		Message: "Registration successful. Please check your email for verification code.",
	}, nil
}

func (s *identityService) VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error {
	_, err := s.repository.ValidateOTP(ctx, req.Email, OTPPurposeEmailVerification, req.OTP)
	if err != nil {
		return fmt.Errorf("email verification failed: %w", err)
	}

	accountResp, err := s.accountService.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	updateReq := &account.UpdateAccountRequest{
		IsActive: &[]bool{true}[0],
	}

	_, err = s.accountService.UpdateAccount(ctx, accountResp.ID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to activate account: %w", err)
	}

	err = s.repository.DeleteOTP(ctx, req.Email, OTPPurposeEmailVerification)
	if err != nil {
		fmt.Printf("Failed to delete OTP: %v\n", err)
	}

	return nil
}

func (s *identityService) ResendEmailVerification(ctx context.Context, req *ResendVerificationRequest) error {
	accountResp, err := s.accountService.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("account not found")
	}

	if accountResp.IsActive {
		return fmt.Errorf("email is already verified")
	}

	otp, err := s.repository.CreateOTP(ctx, req.Email, OTPPurposeEmailVerification)
	if err != nil {
		return fmt.Errorf("failed to create verification OTP: %w", err)
	}

	if s.resendService != nil {
		template := GetEmailVerificationTemplate(otp.Code)
		emailReq := &resend.EmailRequest{
			From:    s.fromEmail,
			To:      []string{req.Email},
			Subject: template.Subject,
			Html:    template.HtmlBody,
			Text:    template.TextBody,
		}

		_, err = s.resendService.SendEmail(ctx, emailReq)
		if err != nil {
			return fmt.Errorf("failed to send verification email: %w", err)
		}
	}

	return nil
}

func (s *identityService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	accountResp, err := s.accountService.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	if !accountResp.IsActive {
		return nil
	}

	otp, err := s.repository.CreateOTP(ctx, req.Email, OTPPurposePasswordReset)
	if err != nil {
		return fmt.Errorf("failed to create password reset OTP: %w", err)
	}

	if s.resendService != nil {
		template := GetPasswordResetTemplate(otp.Code)
		emailReq := &resend.EmailRequest{
			From:    s.fromEmail,
			To:      []string{req.Email},
			Subject: template.Subject,
			Html:    template.HtmlBody,
			Text:    template.TextBody,
		}

		_, err = s.resendService.SendEmail(ctx, emailReq)
		if err != nil {
			return fmt.Errorf("failed to send password reset email: %w", err)
		}
	}

	return nil
}

func (s *identityService) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	_, err := s.repository.ValidateOTP(ctx, req.Email, OTPPurposePasswordReset, req.OTP)
	if err != nil {
		return fmt.Errorf("password reset failed: %w", err)
	}

	accountResp, err := s.accountService.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_ = string(hashedPassword)

	err = s.repository.DeactivateAllUserSessions(ctx, accountResp.ID)
	if err != nil {
		fmt.Printf("Failed to deactivate user sessions: %v\n", err)
	}

	err = s.repository.DeleteOTP(ctx, req.Email, OTPPurposePasswordReset)
	if err != nil {
		fmt.Printf("Failed to delete OTP: %v\n", err)
	}

	if s.resendService != nil {
		template := GetPasswordChangeConfirmationTemplate()
		emailReq := &resend.EmailRequest{
			From:    s.fromEmail,
			To:      []string{req.Email},
			Subject: template.Subject,
			Html:    template.HtmlBody,
			Text:    template.TextBody,
		}

		_, err = s.resendService.SendEmail(ctx, emailReq)
		if err != nil {
			fmt.Printf("Failed to send password change confirmation: %v\n", err)
		}
	}

	return nil
}

func (s *identityService) ChangePassword(ctx context.Context, accountID string, req *ChangePasswordRequest) error {
	accountResp, err := s.accountService.GetAccountByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	if !accountResp.IsActive {
		return fmt.Errorf("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(req.OldPassword), []byte(req.OldPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_ = string(hashedPassword)

	err = s.repository.DeactivateAllUserSessions(ctx, accountID)
	if err != nil {
		fmt.Printf("Failed to deactivate user sessions: %v\n", err)
	}

	if s.resendService != nil {
		template := GetPasswordChangeConfirmationTemplate()
		emailReq := &resend.EmailRequest{
			From:    s.fromEmail,
			To:      []string{accountResp.Email},
			Subject: template.Subject,
			Html:    template.HtmlBody,
			Text:    template.TextBody,
		}

		_, err = s.resendService.SendEmail(ctx, emailReq)
		if err != nil {
			fmt.Printf("Failed to send password change confirmation: %v\n", err)
		}
	}

	return nil
}

func (s *identityService) ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	jwtClaims, err := s.jwtService.Verify(token)
	if err != nil {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	accountClaims := account.NewAccountJWTClaimsFromCustom(jwtClaims.CustomClaims)

	tokenHash := s.hashToken(token)
	session, err := s.repository.GetSessionByToken(ctx, tokenHash)
	if err != nil || session == nil {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	if session.IsExpired() || !session.IsActive {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	accountResp, err := s.accountService.GetAccountByID(ctx, accountClaims.AccountID)
	if err != nil {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	if !accountResp.IsActive {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	s.repository.UpdateSessionLastUsed(ctx, session.ID)

	return &ValidateTokenResponse{
		Valid:   true,
		Claims:  accountClaims,
		Account: accountResp,
	}, nil
}

func (s *identityService) RefreshToken(ctx context.Context, token string, userAgent, ipAddress string) (*RefreshTokenResponse, error) {
	validateResp, err := s.ValidateToken(ctx, token)
	if err != nil || !validateResp.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	newToken, err := s.jwtService.Refresh(token)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	oldTokenHash := s.hashToken(token)
	err = s.repository.DeactivateSession(ctx, oldTokenHash)
	if err != nil {
		fmt.Printf("Failed to deactivate old session: %v\n", err)
	}

	session := &Session{
		AccountID: validateResp.Claims.AccountID,
		TokenHash: s.hashToken(newToken),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	_, err = s.repository.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	return &RefreshTokenResponse{
		Token: newToken,
	}, nil
}

func (s *identityService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}