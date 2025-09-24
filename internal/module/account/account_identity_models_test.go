package account

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestOTP_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "expired OTP",
			expiresAt: time.Now().Add(-1 * time.Minute),
			expected:  true,
		},
		{
			name:      "valid OTP",
			expiresAt: time.Now().Add(5 * time.Minute),
			expected:  false,
		},
		{
			name:      "exactly at expiry time",
			expiresAt: time.Now(),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp := &OTP{
				ExpiresAt: tt.expiresAt,
			}
			result := otp.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOTP_IsMaxAttemptsReached(t *testing.T) {
	tests := []struct {
		name     string
		attempts int
		expected bool
	}{
		{
			name:     "no attempts",
			attempts: 0,
			expected: false,
		},
		{
			name:     "few attempts",
			attempts: 3,
			expected: false,
		},
		{
			name:     "exactly at max attempts",
			attempts: MaxOTPAttempts,
			expected: true,
		},
		{
			name:     "exceeded max attempts",
			attempts: MaxOTPAttempts + 1,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp := &OTP{
				Attempts: tt.attempts,
			}
			result := otp.IsMaxAttemptsReached()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "expired session",
			expiresAt: time.Now().Add(-1 * time.Hour),
			expected:  true,
		},
		{
			name:      "valid session",
			expiresAt: time.Now().Add(24 * time.Hour),
			expected:  false,
		},
		{
			name:      "exactly at expiry time",
			expiresAt: time.Now(),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				ExpiresAt: tt.expiresAt,
			}
			result := session.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSession_ToSessionInfo(t *testing.T) {
	objectID := primitive.NewObjectID()
	now := time.Now()

	session := &Session{
		ID:         objectID,
		ExpiresAt:  now.Add(24 * time.Hour),
		LastUsedAt: now,
		UserAgent:  "Mozilla/5.0",
		IPAddress:  "192.168.1.1",
	}

	result := session.ToSessionInfo()

	expected := &SessionInfo{
		ID:         objectID.Hex(),
		ExpiresAt:  session.ExpiresAt,
		LastUsedAt: session.LastUsedAt,
		UserAgent:  "Mozilla/5.0",
		IPAddress:  "192.168.1.1",
	}

	assert.Equal(t, expected, result)
}

func TestGetWelcomeEmailTemplate(t *testing.T) {
	tests := []struct {
		name     string
		username string
	}{
		{
			name:     "normal username",
			username: "johndoe",
		},
		{
			name:     "username with spaces",
			username: "John Doe",
		},
		{
			name:     "empty username",
			username: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWelcomeEmailTemplate(tt.username)

			assert.Equal(t, "Welcome to Our Platform!", result.Subject)
			assert.Contains(t, result.HtmlBody, tt.username)
			assert.Contains(t, result.TextBody, tt.username)
			assert.Contains(t, result.HtmlBody, "Welcome")
			assert.Contains(t, result.TextBody, "Welcome")
		})
	}
}

func TestGetEmailVerificationTemplate(t *testing.T) {
	tests := []struct {
		name string
		otp  string
	}{
		{
			name: "normal OTP",
			otp:  "123456",
		},
		{
			name: "empty OTP",
			otp:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEmailVerificationTemplate(tt.otp)

			assert.Equal(t, "Email Verification Code", result.Subject)
			assert.Contains(t, result.HtmlBody, tt.otp)
			assert.Contains(t, result.TextBody, tt.otp)
			assert.Contains(t, result.HtmlBody, "verification")
			assert.Contains(t, result.TextBody, "verification")
			assert.Contains(t, result.HtmlBody, "5 minutes")
			assert.Contains(t, result.TextBody, "5 minutes")
		})
	}
}

func TestGetPasswordResetTemplate(t *testing.T) {
	tests := []struct {
		name string
		otp  string
	}{
		{
			name: "normal OTP",
			otp:  "654321",
		},
		{
			name: "empty OTP",
			otp:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPasswordResetTemplate(tt.otp)

			assert.Equal(t, "Password Reset Code", result.Subject)
			assert.Contains(t, result.HtmlBody, tt.otp)
			assert.Contains(t, result.TextBody, tt.otp)
			assert.Contains(t, result.HtmlBody, "Password Reset")
			assert.Contains(t, result.TextBody, "Password Reset")
			assert.Contains(t, result.HtmlBody, "5 minutes")
			assert.Contains(t, result.TextBody, "5 minutes")
		})
	}
}

func TestGetPasswordChangeConfirmationTemplate(t *testing.T) {
	result := GetPasswordChangeConfirmationTemplate()

	assert.Equal(t, "Password Changed Successfully", result.Subject)
	assert.Contains(t, result.HtmlBody, "Password Changed")
	assert.Contains(t, result.TextBody, "Password Changed")
	assert.Contains(t, result.HtmlBody, "successfully changed")
	assert.Contains(t, result.TextBody, "successfully changed")
	assert.Contains(t, result.HtmlBody, "contact support")
	assert.Contains(t, result.TextBody, "contact support")
}

func TestOTPJSONMarshalUnmarshal(t *testing.T) {
	objectID := primitive.NewObjectID()
	now := time.Now().Truncate(time.Millisecond)

	original := &OTP{
		ID:        objectID,
		Email:     "test@example.com",
		Purpose:   OTPPurposeEmailVerification,
		Code:      "123456",
		Attempts:  2,
		ExpiresAt: now.Add(5 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled OTP
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original.ID, unmarshaled.ID)
	assert.Equal(t, original.Email, unmarshaled.Email)
	assert.Equal(t, original.Purpose, unmarshaled.Purpose)
	assert.Equal(t, original.Code, unmarshaled.Code)
	assert.Equal(t, original.Attempts, unmarshaled.Attempts)
	assert.True(t, original.ExpiresAt.Equal(unmarshaled.ExpiresAt))
	assert.True(t, original.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, original.UpdatedAt.Equal(unmarshaled.UpdatedAt))
}

func TestSessionJSONMarshalUnmarshal(t *testing.T) {
	objectID := primitive.NewObjectID()
	now := time.Now().Truncate(time.Millisecond)

	original := &Session{
		ID:         objectID,
		AccountID:  "account123",
		TokenHash:  "hashedtoken",
		IsActive:   true,
		ExpiresAt:  now.Add(24 * time.Hour),
		CreatedAt:  now,
		LastUsedAt: now,
		UserAgent:  "Mozilla/5.0",
		IPAddress:  "192.168.1.1",
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled Session
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original.ID, unmarshaled.ID)
	assert.Equal(t, original.AccountID, unmarshaled.AccountID)
	assert.Equal(t, original.TokenHash, unmarshaled.TokenHash)
	assert.Equal(t, original.IsActive, unmarshaled.IsActive)
	assert.True(t, original.ExpiresAt.Equal(unmarshaled.ExpiresAt))
	assert.True(t, original.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, original.LastUsedAt.Equal(unmarshaled.LastUsedAt))
	assert.Equal(t, original.UserAgent, unmarshaled.UserAgent)
	assert.Equal(t, original.IPAddress, unmarshaled.IPAddress)
}

func TestLoginRequestJSONMarshalUnmarshal(t *testing.T) {
	original := &LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled LoginRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original, &unmarshaled)
}

func TestLoginResponseJSONMarshalUnmarshal(t *testing.T) {
	original := &LoginResponse{
		Token:   "jwt.token.here",
		Account: CreateTestAccountResponse(),
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled LoginResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original.Token, unmarshaled.Token)
	assert.NotNil(t, unmarshaled.Account)
}

func TestRegisterRequestJSONMarshalUnmarshal(t *testing.T) {
	original := &RegisterRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Password:  "password123",
		Avatar:    "https://example.com/avatar.jpg",
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled RegisterRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original, &unmarshaled)
}

func TestVerifyEmailRequestJSONMarshalUnmarshal(t *testing.T) {
	original := &VerifyEmailRequest{
		Email: "test@example.com",
		OTP:   "123456",
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled VerifyEmailRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, original, &unmarshaled)
}

func TestMeResponseJSONMarshalUnmarshal(t *testing.T) {
	original := &MeResponse{
		Account: CreateTestAccountResponse(),
		Session: &SessionInfo{
			ID:         primitive.NewObjectID().Hex(),
			ExpiresAt:  time.Now().Add(24 * time.Hour),
			LastUsedAt: time.Now(),
			UserAgent:  "Mozilla/5.0",
			IPAddress:  "192.168.1.1",
		},
	}

	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled MeResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.NotNil(t, unmarshaled.Account)
	assert.NotNil(t, unmarshaled.Session)
	assert.Equal(t, original.Session.ID, unmarshaled.Session.ID)
}

func TestOTPPurposeConstants(t *testing.T) {
	assert.Equal(t, OTPPurpose("email_verification"), OTPPurposeEmailVerification)
	assert.Equal(t, OTPPurpose("password_reset"), OTPPurposePasswordReset)
}

func TestOTPConstants(t *testing.T) {
	assert.Equal(t, 5, MaxOTPAttempts)
	assert.Equal(t, 6, OTPLength)
	assert.Equal(t, 5*time.Minute, OTPExpiry)
}
