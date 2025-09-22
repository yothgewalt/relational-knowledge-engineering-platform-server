package config

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/vault"
)

type MockVaultService struct {
	secrets map[string]map[string]interface{}
}

func (m *MockVaultService) HealthCheck(ctx context.Context) vault.HealthStatus {
	return vault.HealthStatus{
		Connected:     true,
		Authenticated: true,
	}
}

func (m *MockVaultService) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	if secret, exists := m.secrets[path]; exists {
		return secret, nil
	}
	return nil, nil
}

func (m *MockVaultService) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	return nil
}

func (m *MockVaultService) ListSecrets(ctx context.Context, path string) ([]string, error) {
	return []string{}, nil
}

func (m *MockVaultService) DeleteSecret(ctx context.Context, path string) error {
	return nil
}

func (m *MockVaultService) Close() error {
	return nil
}

func TestConfigLoad(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config == nil {
		t.Fatal("Config is nil")
	}

	if config.Server.Port == "" {
		t.Error("Server port should not be empty")
	}
	if config.Mongo.Address == "" {
		t.Error("Mongo address should not be empty")
	}
	if config.Redis.Address == "" {
		t.Error("Redis address should not be empty")
	}
}

func TestConfigLoadWithVault(t *testing.T) {
	mockVault := &MockVaultService{
		secrets: map[string]map[string]interface{}{
			"secret/database/mongodb": {
				"address":  "vault-mongo://test:27017",
				"username": "vault-admin",
				"password": "vault-password",
				"database": "vault-db",
			},
			"secret/database/redis": {
				"address":  "vault-redis://test:6379",
				"password": "vault-redis-pass",
				"database": "1",
			},
		},
	}

	instance = nil
	once = sync.Once{}
	vaultClient = mockVault

	config, err := LoadWithVault(mockVault)
	if err != nil {
		t.Fatalf("Failed to load config with Vault: %v", err)
	}

	if config == nil {
		t.Fatal("Config is nil")
	}

	if config.Mongo.Address != "vault-mongo://test:27017" {
		t.Errorf("Expected Mongo address from Vault, got: %s", config.Mongo.Address)
	}
	if config.Mongo.Username != "vault-admin" {
		t.Errorf("Expected Mongo username from Vault, got: %s", config.Mongo.Username)
	}
	if config.Mongo.Password != "vault-password" {
		t.Errorf("Expected Mongo password from Vault, got: %s", config.Mongo.Password)
	}
	if config.Mongo.Database != "vault-db" {
		t.Errorf("Expected Mongo database from Vault, got: %s", config.Mongo.Database)
	}

	if config.Redis.Address != "vault-redis://test:6379" {
		t.Errorf("Expected Redis address from Vault, got: %s", config.Redis.Address)
	}
	if config.Redis.Password != "vault-redis-pass" {
		t.Errorf("Expected Redis password from Vault, got: %s", config.Redis.Password)
	}
	if config.Redis.Database != 1 {
		t.Errorf("Expected Redis database from Vault, got: %d", config.Redis.Database)
	}
}

func TestVaultSecretPaths(t *testing.T) {
	os.Setenv("VAULT_MONGO_SECRET_PATH", "custom/mongo/path")
	os.Setenv("VAULT_REDIS_SECRET_PATH", "custom/redis/path")
	os.Setenv("VAULT_RESEND_SECRET_PATH", "custom/resend/path")

	defer func() {
		os.Unsetenv("VAULT_MONGO_SECRET_PATH")
		os.Unsetenv("VAULT_REDIS_SECRET_PATH")
		os.Unsetenv("VAULT_RESEND_SECRET_PATH")
	}()

	vaultConfig, err := loadVaultSecretsConfig()
	if err != nil {
		t.Fatalf("Failed to load vault secrets config: %v", err)
	}

	if vaultConfig.MongoSecretPath != "custom/mongo/path" {
		t.Errorf("Expected custom mongo path, got: %s", vaultConfig.MongoSecretPath)
	}
	if vaultConfig.RedisSecretPath != "custom/redis/path" {
		t.Errorf("Expected custom redis path, got: %s", vaultConfig.RedisSecretPath)
	}
	if vaultConfig.ResendSecretPath != "custom/resend/path" {
		t.Errorf("Expected custom resend path, got: %s", vaultConfig.ResendSecretPath)
	}
}

func TestFallbackToEnvVars(t *testing.T) {
	os.Setenv("MONGO_ADDRESS", "env-mongo://test:27017")
	os.Setenv("REDIS_ADDRESS", "env-redis://test:6379")

	defer func() {
		os.Unsetenv("MONGO_ADDRESS")
		os.Unsetenv("REDIS_ADDRESS")
	}()

	instance = nil
	once = sync.Once{}
	vaultClient = nil

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Mongo.Address != "env-mongo://test:27017" {
		t.Errorf("Expected Mongo address from env, got: %s", config.Mongo.Address)
	}
	if config.Redis.Address != "env-redis://test:6379" {
		t.Errorf("Expected Redis address from env, got: %s", config.Redis.Address)
	}
}
