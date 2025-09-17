package config

import (
	"os"
	"testing"
)

func TestConfig_Load(t *testing.T) {
	// Test default configuration
	cfg := Load()
	
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}
	
	// Test default values
	if cfg.Database.MongoDB.Host != "localhost" {
		t.Errorf("MongoDB.Host = %s, want localhost", cfg.Database.MongoDB.Host)
	}
	
	if cfg.Database.MongoDB.Port != 27017 {
		t.Errorf("MongoDB.Port = %d, want 27017", cfg.Database.MongoDB.Port)
	}
	
	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want 3000", cfg.Server.Port)
	}
}

func TestConfig_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("MONGODB_HOST", "test-mongo")
	os.Setenv("MONGODB_PORT", "27018")
	os.Setenv("SERVER_PORT", "8080")
	
	// Cleanup after test
	defer func() {
		os.Unsetenv("MONGODB_HOST")
		os.Unsetenv("MONGODB_PORT")
		os.Unsetenv("SERVER_PORT")
	}()
	
	cfg := Load()
	
	if cfg.Database.MongoDB.Host != "test-mongo" {
		t.Errorf("MongoDB.Host = %s, want test-mongo", cfg.Database.MongoDB.Host)
	}
	
	if cfg.Database.MongoDB.Port != 27018 {
		t.Errorf("MongoDB.Port = %d, want 27018", cfg.Database.MongoDB.Port)
	}
	
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "Valid integer",
			envValue:     "123",
			defaultValue: 456,
			expected:     123,
		},
		{
			name:         "Invalid integer",
			envValue:     "invalid",
			defaultValue: 456,
			expected:     456,
		},
		{
			name:         "Empty value",
			envValue:     "",
			defaultValue: 456,
			expected:     456,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
				defer os.Unsetenv(key)
			}
			
			result := getEnvAsInt(key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsInt() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "Environment variable exists",
			envValue:     "test-value",
			defaultValue: "default-value",
			expected:     "test-value",
		},
		{
			name:         "Environment variable does not exist",
			envValue:     "",
			defaultValue: "default-value",
			expected:     "default-value",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_STRING_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
				defer os.Unsetenv(key)
			}
			
			result := getEnv(key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv() = %s, want %s", result, tt.expected)
			}
		})
	}
}
