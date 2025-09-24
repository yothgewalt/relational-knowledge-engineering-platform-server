package jwt

import (
	"context"
	"testing"
	"time"
)

func TestNewJWTService(t *testing.T) {
	tests := []struct {
		name    string
		config  JWTConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: JWTConfig{
				SecretKey:     "test-secret-key",
				TokenDuration: time.Hour,
				Issuer:        "test-issuer",
			},
			wantErr: false,
		},
		{
			name: "empty secret key",
			config: JWTConfig{
				SecretKey:     "",
				TokenDuration: time.Hour,
				Issuer:        "test-issuer",
			},
			wantErr: true,
		},
		{
			name: "default values applied",
			config: JWTConfig{
				SecretKey: "test-secret-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewJWTService(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJWTService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && service == nil {
				t.Errorf("NewJWTService() returned nil service")
			}
		})
	}
}

func TestJWTService_Generate_And_Verify(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
		Issuer:        "test-issuer",
	}

	service, err := NewJWTService(config)
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	customClaims := map[string]any{
		"user_id": "123",
		"email":   "test@example.com",
		"role":    "user",
	}

	token, err := service.Generate(customClaims)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if token == "" {
		t.Errorf("Generate() returned empty token")
	}

	claims, err := service.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims == nil {
		t.Fatalf("Verify() returned nil claims")
	}

	userID, exists := claims.GetCustomClaim("user_id")
	if !exists {
		t.Errorf("Expected user_id claim to exist")
	}
	if userID != "123" {
		t.Errorf("Expected user_id to be '123', got %v", userID)
	}

	email, exists := claims.GetCustomClaim("email")
	if !exists {
		t.Errorf("Expected email claim to exist")
	}
	if email != "test@example.com" {
		t.Errorf("Expected email to be 'test@example.com', got %v", email)
	}
}

func TestJWTService_Verify_InvalidToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
		Issuer:        "test-issuer",
	}

	service, err := NewJWTService(config)
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	tests := []struct {
		name        string
		token       string
		expectedErr error
	}{
		{
			name:        "invalid token format",
			token:       "invalid-token",
			expectedErr: ErrInvalidToken,
		},
		{
			name:        "empty token",
			token:       "",
			expectedErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Verify(tt.token)
			if err != tt.expectedErr {
				t.Errorf("Verify() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Millisecond * 1,
		Issuer:        "test-issuer",
	}

	service, err := NewJWTService(config)
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	customClaims := map[string]any{
		"user_id": "123",
	}

	token, err := service.Generate(customClaims)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	time.Sleep(time.Millisecond * 10)

	_, err = service.Verify(token)
	if err != ErrExpiredToken {
		t.Errorf("Verify() error = %v, expected %v", err, ErrExpiredToken)
	}
}

func TestJWTService_Refresh(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
		Issuer:        "test-issuer",
	}

	service, err := NewJWTService(config)
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	customClaims := map[string]any{
		"user_id": "123",
		"email":   "test@example.com",
	}

	originalToken, err := service.Generate(customClaims)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	refreshedToken, err := service.Refresh(originalToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if refreshedToken == "" {
		t.Errorf("Refresh() returned empty token")
	}

	claims, err := service.Verify(refreshedToken)
	if err != nil {
		t.Fatalf("Verify() refreshed token error = %v", err)
	}

	userID, exists := claims.GetCustomClaim("user_id")
	if !exists || userID != "123" {
		t.Errorf("Expected user_id claim to be preserved in refreshed token")
	}
}

func TestJWTService_ExtractClaims(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
		Issuer:        "test-issuer",
	}

	service, err := NewJWTService(config)
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	customClaims := map[string]any{
		"user_id": "123",
		"email":   "test@example.com",
	}

	token, err := service.Generate(customClaims)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := service.ExtractClaims(token)
	if err != nil {
		t.Fatalf("ExtractClaims() error = %v", err)
	}

	userID, exists := claims.GetCustomClaim("user_id")
	if !exists || userID != "123" {
		t.Errorf("Expected user_id claim to be extracted")
	}
}

func TestJWTService_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		config         JWTConfig
		expectedStatus bool
	}{
		{
			name: "healthy service",
			config: JWTConfig{
				SecretKey:     "test-secret-key",
				TokenDuration: time.Hour,
				Issuer:        "test-issuer",
			},
			expectedStatus: true,
		},
		{
			name: "unhealthy service - no secret",
			config: JWTConfig{
				SecretKey:     "",
				TokenDuration: time.Hour,
				Issuer:        "test-issuer",
			},
			expectedStatus: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := NewJWTService(tt.config)
			if service == nil {
				return
			}

			ctx := context.Background()
			health := service.HealthCheck(ctx)

			if health.ValidKey != tt.expectedStatus {
				t.Errorf("HealthCheck() ValidKey = %v, expected %v", health.ValidKey, tt.expectedStatus)
			}

			if tt.expectedStatus && health.Error != "" {
				t.Errorf("HealthCheck() should not have error when healthy, got: %s", health.Error)
			}

			if !tt.expectedStatus && health.Error == "" {
				t.Errorf("HealthCheck() should have error when unhealthy")
			}
		})
	}
}

func TestJWTClaims_CustomClaimMethods(t *testing.T) {
	claims := &JWTClaims{}

	value, exists := claims.GetCustomClaim("non-existent")
	if exists {
		t.Errorf("GetCustomClaim() should return false for non-existent claim")
	}
	if value != nil {
		t.Errorf("GetCustomClaim() should return nil value for non-existent claim")
	}

	claims.SetCustomClaim("test_key", "test_value")

	value, exists = claims.GetCustomClaim("test_key")
	if !exists {
		t.Errorf("GetCustomClaim() should return true for existing claim")
	}
	if value != "test_value" {
		t.Errorf("GetCustomClaim() value = %v, expected 'test_value'", value)
	}
}
