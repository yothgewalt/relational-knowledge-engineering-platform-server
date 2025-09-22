package vault

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

type VaultConfig struct {
	Address   string
	Token     string
	TLSConfig *TLSConfig
}

type TLSConfig struct {
	CACert     string
	ClientCert string
	ClientKey  string
	Insecure   bool
}

type HealthStatus struct {
	Connected    bool          `json:"connected"`
	Address      string        `json:"address"`
	Authenticated bool         `json:"authenticated"`
	Latency      time.Duration `json:"latency"`
	Error        string        `json:"error,omitempty"`
}

type VaultService interface {
	HealthCheck(ctx context.Context) HealthStatus
	GetSecret(ctx context.Context, path string) (map[string]interface{}, error)
	PutSecret(ctx context.Context, path string, data map[string]interface{}) error
	ListSecrets(ctx context.Context, path string) ([]string, error)
	DeleteSecret(ctx context.Context, path string) error
	Close() error
}

type VaultClient struct {
	client *api.Client
	config VaultConfig
	mu     sync.RWMutex
}

func NewVaultClient(config VaultConfig) (*VaultClient, error) {
	// Create Vault API config
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = config.Address

	// Configure TLS if provided
	if config.TLSConfig != nil {
		tlsConfig := &api.TLSConfig{
			CACert:     config.TLSConfig.CACert,
			ClientCert: config.TLSConfig.ClientCert,
			ClientKey:  config.TLSConfig.ClientKey,
			Insecure:   config.TLSConfig.Insecure,
		}
		if err := vaultConfig.ConfigureTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
	}

	// Create Vault client
	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set authentication token
	client.SetToken(config.Token)

	// Verify connection and authentication
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test authentication by checking token info
	if _, err := client.Auth().Token().LookupSelfWithContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to authenticate with Vault: %w", err)
	}

	return &VaultClient{
		client: client,
		config: config,
	}, nil
}

func (v *VaultClient) HealthCheck(ctx context.Context) HealthStatus {
	v.mu.RLock()
	defer v.mu.RUnlock()

	start := time.Now()
	status := HealthStatus{
		Address: v.config.Address,
	}

	// Check Vault health endpoint
	_, err := v.client.Sys().HealthWithContext(ctx)
	if err != nil {
		status.Connected = false
		status.Authenticated = false
		status.Error = err.Error()
		return status
	}

	status.Connected = true
	status.Latency = time.Since(start)

	// Verify authentication
	if _, err := v.client.Auth().Token().LookupSelfWithContext(ctx); err != nil {
		status.Authenticated = false
		status.Error = fmt.Sprintf("authentication failed: %v", err)
	} else {
		status.Authenticated = true
	}

	return status
}

func (v *VaultClient) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	secret, err := v.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at path %s: %w", path, err)
	}

	if secret == nil {
		return nil, fmt.Errorf("secret not found at path: %s", path)
	}

	// Handle KV v2 format (data nested under "data" key)
	if data, ok := secret.Data["data"]; ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			return dataMap, nil
		}
	}

	// Return raw data for KV v1 or other secret engines
	return secret.Data, nil
}

func (v *VaultClient) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// For KV v2, wrap data under "data" key
	secretData := map[string]interface{}{
		"data": data,
	}

	_, err := v.client.Logical().WriteWithContext(ctx, path, secretData)
	if err != nil {
		return fmt.Errorf("failed to write secret at path %s: %w", path, err)
	}

	return nil
}

func (v *VaultClient) ListSecrets(ctx context.Context, path string) ([]string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Ensure path ends with / for listing
	if path[len(path)-1] != '/' {
		path += "/"
	}

	secret, err := v.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at path %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	// Extract keys from response
	if keys, ok := secret.Data["keys"]; ok {
		if keysList, ok := keys.([]interface{}); ok {
			result := make([]string, len(keysList))
			for i, key := range keysList {
				if keyStr, ok := key.(string); ok {
					result[i] = keyStr
				}
			}
			return result, nil
		}
	}

	return []string{}, nil
}

func (v *VaultClient) DeleteSecret(ctx context.Context, path string) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	_, err := v.client.Logical().DeleteWithContext(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete secret at path %s: %w", path, err)
	}

	return nil
}

func (v *VaultClient) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Vault client doesn't require explicit closing
	// Just clear the token for security
	if v.client != nil {
		v.client.ClearToken()
	}

	return nil
}

// Helper function to check if path is KV v2
func (v *VaultClient) isKVv2(path string) (bool, string, error) {
	// This is a simplified check - in production you might want to
	// query the sys/mounts endpoint to determine the engine type
	// For now, assume KV v2 if path starts with common patterns
	if len(path) >= 7 && path[:7] == "secret/" {
		return true, path, nil
	}
	if len(path) >= 3 && path[:3] == "kv/" {
		return true, path, nil
	}
	return false, path, nil
}