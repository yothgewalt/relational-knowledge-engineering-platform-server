package jwt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	ErrTokenClaims  = errors.New("invalid token claims")
)

type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
	Issuer        string
}

type HealthStatus struct {
	Configured bool          `json:"configured"`
	ValidKey   bool          `json:"valid_key"`
	Issuer     string        `json:"issuer"`
	Duration   time.Duration `json:"token_duration"`
	Error      string        `json:"error,omitempty"`
}

type JWTClaims struct {
	jwt.RegisteredClaims
	CustomClaims map[string]any `json:"custom_claims,omitempty"`
}

type JWTService struct {
	config JWTConfig
	mu     sync.RWMutex
}

func NewJWTService(config JWTConfig) (*JWTService, error) {
	if config.SecretKey == "" {
		return nil, fmt.Errorf("JWT secret key is required")
	}
	
	if config.TokenDuration <= 0 {
		config.TokenDuration = 24 * time.Hour
	}
	
	if config.Issuer == "" {
		config.Issuer = "jwt-service"
	}

	return &JWTService{
		config: config,
	}, nil
}

func (s *JWTService) Generate(customClaims map[string]any) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.Issuer,
		},
		CustomClaims: customClaims,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.SecretKey))
}

func (s *JWTService) Verify(tokenString string) (*JWTClaims, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.SecretKey), nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrTokenClaims
	}

	return claims, nil
}

func (s *JWTService) Refresh(tokenString string) (string, error) {
	claims, err := s.Verify(tokenString)
	if err != nil {
		if !errors.Is(err, ErrExpiredToken) {
			return "", err
		}
	}

	return s.Generate(claims.CustomClaims)
}

func (s *JWTService) ExtractClaims(tokenString string) (*JWTClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrTokenClaims
	}

	return claims, nil
}

func (s *JWTService) GetConfig() JWTConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *JWTService) HealthCheck(ctx context.Context) HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := HealthStatus{
		Configured: s.config.SecretKey != "",
		Issuer:     s.config.Issuer,
		Duration:   s.config.TokenDuration,
	}

	if !status.Configured {
		status.Error = "JWT secret key not configured"
		return status
	}

	testClaims := map[string]any{
		"test": "health_check",
	}

	token, err := s.Generate(testClaims)
	if err != nil {
		status.ValidKey = false
		status.Error = fmt.Sprintf("failed to generate test token: %v", err)
		return status
	}

	_, err = s.Verify(token)
	if err != nil {
		status.ValidKey = false
		status.Error = fmt.Sprintf("failed to verify test token: %v", err)
		return status
	}

	status.ValidKey = true
	return status
}

func (s *JWTService) Close() error {
	return nil
}

func (c *JWTClaims) GetCustomClaim(key string) (any, bool) {
	if c.CustomClaims == nil {
		return nil, false
	}
	value, exists := c.CustomClaims[key]
	return value, exists
}

func (c *JWTClaims) SetCustomClaim(key string, value any) {
	if c.CustomClaims == nil {
		c.CustomClaims = make(map[string]interface{})
	}
	c.CustomClaims[key] = value
}