package account

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/jwt"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
)

type AccountService interface {
	// Account management methods
	CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountResponse, error)
	GetAccountByID(ctx context.Context, id string) (*AccountResponse, error)
	GetAccountByEmail(ctx context.Context, email string) (*AccountResponse, error)
	GetAccountByUsername(ctx context.Context, username string) (*AccountResponse, error)
	UpdateAccount(ctx context.Context, id string, req *UpdateAccountRequest) (*AccountResponse, error)
	DeleteAccount(ctx context.Context, id string) error
	ListAccounts(ctx context.Context, req *ListAccountsRequest) (*mongo.PaginatedResult[AccountResponse], error)

	// Authentication methods
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
	GetCurrentUser(ctx context.Context, token string) (*MeResponse, error)
}

type accountService struct {
	repository              AccountRepository
	accountIdentityRepository AccountIdentityRepository
	jwtService              *jwt.JWTService
	resendService           resend.ResendService
	fromEmail               string
}

func NewAccountService(
	mongoService *mongo.MongoService,
	jwtService *jwt.JWTService,
	resendService resend.ResendService,
	fromEmail string,
) AccountService {
	repository := NewAccountRepository(mongoService)
	accountIdentityRepository := NewAccountIdentityRepository(mongoService)
	
	return &accountService{
		repository:                repository,
		accountIdentityRepository: accountIdentityRepository,
		jwtService:                jwtService,
		resendService:             resendService,
		fromEmail:                 fromEmail,
	}
}

func (s *accountService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountResponse, error) {
	exists, err := s.repository.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("account with email %s already exists", req.Email)
	}

	exists, err = s.repository.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("account with username %s already exists", req.Username)
	}

	account := &Account{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Avatar:    req.Avatar,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdAccount, err := s.repository.Create(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return createdAccount.ToResponse(), nil
}

func (s *accountService) GetAccountByID(ctx context.Context, id string) (*AccountResponse, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID format: %w", err)
	}

	account, err := s.repository.GetByID(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return account.ToResponse(), nil
}

func (s *accountService) GetAccountByEmail(ctx context.Context, email string) (*AccountResponse, error) {
	account, err := s.repository.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by email: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return account.ToResponse(), nil
}

func (s *accountService) GetAccountByUsername(ctx context.Context, username string) (*AccountResponse, error) {
	account, err := s.repository.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by username: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return account.ToResponse(), nil
}

func (s *accountService) UpdateAccount(ctx context.Context, id string, req *UpdateAccountRequest) (*AccountResponse, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID format: %w", err)
	}

	updateData := bson.M{
		"updated_at": time.Now(),
	}

	if req.Username != nil && *req.Username != "" {
		exists, err := s.repository.ExistsByUsername(ctx, *req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username existence: %w", err)
		}
		
		if exists {
			existing, err := s.repository.GetByUsername(ctx, *req.Username)
			if err != nil {
				return nil, fmt.Errorf("failed to get existing account: %w", err)
			}
			if existing != nil && existing.ID != objectID {
				return nil, fmt.Errorf("username %s is already taken", *req.Username)
			}
		}
		updateData["username"] = *req.Username
	}

	if req.FirstName != nil {
		updateData["first_name"] = *req.FirstName
	}

	if req.LastName != nil {
		updateData["last_name"] = *req.LastName
	}

	if req.Avatar != nil {
		updateData["avatar"] = *req.Avatar
	}

	if req.IsActive != nil {
		updateData["is_active"] = *req.IsActive
	}

	updatedAccount, err := s.repository.Update(ctx, objectID, updateData)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	if updatedAccount == nil {
		return nil, fmt.Errorf("account not found")
	}

	return updatedAccount.ToResponse(), nil
}

func (s *accountService) DeleteAccount(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid account ID format: %w", err)
	}

	account, err := s.repository.GetByID(ctx, objectID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account not found")
	}

	err = s.repository.Delete(ctx, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

func (s *accountService) ListAccounts(ctx context.Context, req *ListAccountsRequest) (*mongo.PaginatedResult[AccountResponse], error) {
	filter := bson.M{}

	if req.Email != "" {
		filter["email"] = bson.M{"$regex": req.Email, "$options": "i"}
	}

	if req.Username != "" {
		filter["username"] = bson.M{"$regex": req.Username, "$options": "i"}
	}

	if req.IsActive != nil {
		filter["is_active"] = *req.IsActive
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	pagination := mongo.PaginationOptions{
		Page:  req.Page,
		Limit: req.Limit,
	}

	result, err := s.repository.List(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	responses := make([]AccountResponse, len(result.Data))
	for i, account := range result.Data {
		responses[i] = *account.ToResponse()
	}

	return &mongo.PaginatedResult[AccountResponse]{
		Data:       responses,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
		HasNext:    result.HasNext,
		HasPrev:    result.HasPrev,
	}, nil
}

func (s *accountService) Login(ctx context.Context, req *LoginRequest, userAgent, ipAddress string) (*LoginResponse, error) {
	accountResp, err := s.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !accountResp.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(req.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	claims := &AccountJWTClaims{
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

	_, err = s.accountIdentityRepository.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &LoginResponse{
		Token:   token,
		Account: accountResp,
	}, nil
}

func (s *accountService) Logout(ctx context.Context, token string) error {
	tokenHash := s.hashToken(token)
	
	err := s.accountIdentityRepository.DeactivateSession(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	return nil
}

func (s *accountService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	_, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	createAccountReq := &CreateAccountRequest{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Avatar:    req.Avatar,
	}

	accountResp, err := s.CreateAccount(ctx, createAccountReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	otp, err := s.accountIdentityRepository.CreateOTP(ctx, req.Email, OTPPurposeEmailVerification)
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

func (s *accountService) VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error {
	_, err := s.accountIdentityRepository.ValidateOTP(ctx, req.Email, OTPPurposeEmailVerification, req.OTP)
	if err != nil {
		return fmt.Errorf("email verification failed: %w", err)
	}

	accountResp, err := s.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	updateReq := &UpdateAccountRequest{
		IsActive: &[]bool{true}[0],
	}

	_, err = s.UpdateAccount(ctx, accountResp.ID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to activate account: %w", err)
	}

	err = s.accountIdentityRepository.DeleteOTP(ctx, req.Email, OTPPurposeEmailVerification)
	if err != nil {
		fmt.Printf("Failed to delete OTP: %v\n", err)
	}

	return nil
}

func (s *accountService) ResendEmailVerification(ctx context.Context, req *ResendVerificationRequest) error {
	accountResp, err := s.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("account not found")
	}

	if accountResp.IsActive {
		return fmt.Errorf("email is already verified")
	}

	otp, err := s.accountIdentityRepository.CreateOTP(ctx, req.Email, OTPPurposeEmailVerification)
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

func (s *accountService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	accountResp, err := s.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	if !accountResp.IsActive {
		return nil
	}

	otp, err := s.accountIdentityRepository.CreateOTP(ctx, req.Email, OTPPurposePasswordReset)
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

func (s *accountService) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	_, err := s.accountIdentityRepository.ValidateOTP(ctx, req.Email, OTPPurposePasswordReset, req.OTP)
	if err != nil {
		return fmt.Errorf("password reset failed: %w", err)
	}

	accountResp, err := s.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_ = string(hashedPassword)

	err = s.accountIdentityRepository.DeactivateAllUserSessions(ctx, accountResp.ID)
	if err != nil {
		fmt.Printf("Failed to deactivate user sessions: %v\n", err)
	}

	err = s.accountIdentityRepository.DeleteOTP(ctx, req.Email, OTPPurposePasswordReset)
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

func (s *accountService) ChangePassword(ctx context.Context, accountID string, req *ChangePasswordRequest) error {
	accountResp, err := s.GetAccountByID(ctx, accountID)
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

	err = s.accountIdentityRepository.DeactivateAllUserSessions(ctx, accountID)
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

func (s *accountService) ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	jwtClaims, err := s.jwtService.Verify(token)
	if err != nil {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	accountClaims := NewAccountJWTClaimsFromCustom(jwtClaims.CustomClaims)

	tokenHash := s.hashToken(token)
	session, err := s.accountIdentityRepository.GetSessionByToken(ctx, tokenHash)
	if err != nil || session == nil {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	if session.IsExpired() || !session.IsActive {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	accountResp, err := s.GetAccountByID(ctx, accountClaims.AccountID)
	if err != nil {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	if !accountResp.IsActive {
		return &ValidateTokenResponse{Valid: false}, nil
	}

	s.accountIdentityRepository.UpdateSessionLastUsed(ctx, session.ID)

	return &ValidateTokenResponse{
		Valid:   true,
		Claims:  accountClaims,
		Account: accountResp,
	}, nil
}

func (s *accountService) RefreshToken(ctx context.Context, token string, userAgent, ipAddress string) (*RefreshTokenResponse, error) {
	validateResp, err := s.ValidateToken(ctx, token)
	if err != nil || !validateResp.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	newToken, err := s.jwtService.Refresh(token)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	oldTokenHash := s.hashToken(token)
	err = s.accountIdentityRepository.DeactivateSession(ctx, oldTokenHash)
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

	_, err = s.accountIdentityRepository.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	return &RefreshTokenResponse{
		Token: newToken,
	}, nil
}

func (s *accountService) GetCurrentUser(ctx context.Context, token string) (*MeResponse, error) {
	validateResp, err := s.ValidateToken(ctx, token)
	if err != nil || !validateResp.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	tokenHash := s.hashToken(token)
	session, err := s.accountIdentityRepository.GetSessionByToken(ctx, tokenHash)
	if err != nil || session == nil {
		return nil, fmt.Errorf("session not found")
	}

	return &MeResponse{
		Account: validateResp.Account,
		Session: session.ToSessionInfo(),
	}, nil
}

func (s *accountService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

