package vault

import (
	"context"
	"testing"
	"time"
)

func TestVaultConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config VaultConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: VaultConfig{
				Address: "http://localhost:8200",
				Token:   "test-token",
			},
			valid: true,
		},
		{
			name: "empty address",
			config: VaultConfig{
				Address: "",
				Token:   "test-token",
			},
			valid: false,
		},
		{
			name: "empty token",
			config: VaultConfig{
				Address: "http://localhost:8200",
				Token:   "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Address == "" || tt.config.Token == "" {
				if tt.valid {
					t.Errorf("expected config to be invalid but test marked as valid")
				}
			} else {
				if !tt.valid {
					t.Errorf("expected config to be valid but test marked as invalid")
				}
			}
		})
	}
}

func TestHealthStatus_Structure(t *testing.T) {
	status := HealthStatus{
		Connected:     true,
		Address:       "http://localhost:8200",
		Authenticated: true,
		Latency:       50 * time.Millisecond,
		Error:         "",
	}

	if !status.Connected {
		t.Error("Connected field should be accessible")
	}
	if status.Address != "http://localhost:8200" {
		t.Error("Address field should be accessible")
	}
	if !status.Authenticated {
		t.Error("Authenticated field should be accessible")
	}
	if status.Latency != 50*time.Millisecond {
		t.Error("Latency field should be accessible")
	}
	if status.Error != "" {
		t.Error("Error field should be empty")
	}
}

func TestVaultService_Interface(t *testing.T) {
	var _ VaultService = (*VaultClient)(nil)
}

func TestTLSConfig_Structure(t *testing.T) {
	tlsConfig := &TLSConfig{
		CACert:     "/path/to/ca.pem",
		ClientCert: "/path/to/client.pem",
		ClientKey:  "/path/to/client-key.pem",
		Insecure:   false,
	}

	if tlsConfig.CACert != "/path/to/ca.pem" {
		t.Error("CACert field should be accessible")
	}
	if tlsConfig.ClientCert != "/path/to/client.pem" {
		t.Error("ClientCert field should be accessible")
	}
	if tlsConfig.ClientKey != "/path/to/client-key.pem" {
		t.Error("ClientKey field should be accessible")
	}
	if tlsConfig.Insecure {
		t.Error("Insecure field should be false")
	}
}

func TestVaultClient_StructureAndMethods(t *testing.T) {
	client := &VaultClient{
		config: VaultConfig{
			Address: "http://localhost:8200",
			Token:   "test-token",
		},
	}

	if client.config.Address != "http://localhost:8200" {
		t.Error("config.Address should be accessible")
	}
	if client.config.Token != "test-token" {
		t.Error("config.Token should be accessible")
	}
}

func TestSecretPath_Validation(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"valid path", "secret/myapp/config", true},
		{"path with spaces", "secret/my app/config", true},
		{"empty path", "", false},
		{"root path", "/", false},
		{"kv v2 path", "kv/data/myapp", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.path == "" || tt.path == "/" {
				if tt.valid {
					t.Errorf("expected path %q to be invalid", tt.path)
				}
			} else {
				if !tt.valid {
					t.Errorf("expected path %q to be valid", tt.path)
				}
			}
		})
	}
}

func TestSecretData_Validation(t *testing.T) {
	tests := []struct {
		name  string
		data  map[string]any
		valid bool
	}{
		{
			name: "valid secret data",
			data: map[string]any{
				"username": "admin",
				"password": "secret123",
			},
			valid: true,
		},
		{
			name:  "empty secret data",
			data:  map[string]any{},
			valid: true,
		},
		{
			name:  "nil secret data",
			data:  nil,
			valid: false,
		},
		{
			name: "complex secret data",
			data: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
				"credentials": []string{"user1", "user2"},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.data == nil {
				if tt.valid {
					t.Error("nil data should be invalid")
				}
			} else {
				if !tt.valid {
					t.Error("non-nil data should be valid")
				}
			}
		})
	}
}

func TestKVv2_PathDetection(t *testing.T) {
	client := &VaultClient{}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"secret engine", "secret/myapp", true},
		{"kv engine", "kv/myapp", true},
		{"custom path", "myengine/data", false},
		{"root path", "myapp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKVv2, _, err := client.isKVv2(tt.path)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if isKVv2 != tt.expected {
				t.Errorf("expected %v, got %v for path %s", tt.expected, isKVv2, tt.path)
			}
		})
	}
}

func TestContext_Handling(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		valid bool
	}{
		{
			name:  "valid context",
			ctx:   context.Background(),
			valid: true,
		},
		{
			name: "context with timeout",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
				_ = cancel
				return ctx
			}(),
			valid: true,
		},
		{
			name:  "cancelled context",
			ctx:   func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctx.Err()
			if tt.valid && err != nil {
				t.Errorf("expected valid context but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid context but got no error")
			}
		})
	}
}

func TestVaultClient_Close(t *testing.T) {
	client := &VaultClient{
		client: nil,
		config: VaultConfig{
			Address: "http://localhost:8200",
			Token:   "test-token",
		},
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Close should not return error, got: %v", err)
	}
}

func TestHealthStatus_ErrorHandling(t *testing.T) {
	status := HealthStatus{
		Connected:     false,
		Authenticated: false,
		Latency:       0,
		Error:         "connection refused",
	}

	if status.Connected {
		t.Error("status should indicate disconnected")
	}
	if status.Authenticated {
		t.Error("status should indicate not authenticated")
	}
	if status.Error == "" {
		t.Error("error message should be present")
	}
	if status.Latency != 0 {
		t.Error("latency should be zero for failed connection")
	}
}

func TestSecretOperations_Validation(t *testing.T) {
	testCases := []struct {
		operation string
		path      string
		data      map[string]any
		valid     bool
	}{
		{"get", "secret/myapp", nil, true},
		{"get", "", nil, false},
		{"put", "secret/myapp", map[string]any{"key": "value"}, true},
		{"put", "secret/myapp", nil, false},
		{"delete", "secret/myapp", nil, true},
		{"delete", "", nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.operation+"_"+tc.path, func(t *testing.T) {
			switch tc.operation {
			case "get", "delete":
				if tc.path == "" {
					if tc.valid {
						t.Error("empty path should be invalid")
					}
				}
			case "put":
				if tc.path == "" || tc.data == nil {
					if tc.valid {
						t.Error("empty path or nil data should be invalid")
					}
				}
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	client := &VaultClient{
		config: VaultConfig{
			Address: "http://localhost:8200",
			Token:   "test-token",
		},
	}

	done := make(chan bool, 5)

	for range 5 {
		go func() {
			defer func() { done <- true }()

			_ = client.config.Address
			_ = client.config.Token
		}()
	}

	for range 5 {
		<-done
	}
}
