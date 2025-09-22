package config

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	clearEnvVars()

	resetSingleton()

	config, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if config.Server.Port != "3000" {
		t.Errorf("expected default port 3000, got %s", config.Server.Port)
	}
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", config.Server.Host)
	}
	if config.Server.AppName != "Relational Knowledge Engineering Platform" {
		t.Errorf("expected default app name, got %s", config.Server.AppName)
	}
	if config.Server.ReadTimeout != 10*time.Second {
		t.Errorf("expected default read timeout 10s, got %v", config.Server.ReadTimeout)
	}

	// Test MongoDB defaults
	if config.MongoDB.URL != "mongodb://localhost:27017" {
		t.Errorf("expected default MongoDB URL, got %s", config.MongoDB.URL)
	}
	if config.MongoDB.Database != "knowledge_platform" {
		t.Errorf("expected default MongoDB database, got %s", config.MongoDB.Database)
	}

	// Test Neo4j defaults
	if config.Neo4j.URI != "bolt://localhost:7687" {
		t.Errorf("expected default Neo4j URI, got %s", config.Neo4j.URI)
	}
	if config.Neo4j.Username != "neo4j" {
		t.Errorf("expected default Neo4j username, got %s", config.Neo4j.Username)
	}

	// Test Redis defaults
	if config.Redis.Address != "localhost:6379" {
		t.Errorf("expected default Redis address, got %s", config.Redis.Address)
	}
	if config.Redis.Database != 0 {
		t.Errorf("expected default Redis database 0, got %d", config.Redis.Database)
	}

	// Test MinIO defaults
	if config.MinIO.Endpoint != "localhost:9000" {
		t.Errorf("expected default MinIO endpoint, got %s", config.MinIO.Endpoint)
	}
	if config.MinIO.BucketName != "knowledge-platform" {
		t.Errorf("expected default MinIO bucket name, got %s", config.MinIO.BucketName)
	}

	if !config.Features.APIDocsEnabled {
		t.Error("expected API docs to be enabled by default")
	}
	if !config.Features.HealthCheckEnabled {
		t.Error("expected health check to be enabled by default")
	}
	if config.Features.DebugMode {
		t.Error("expected debug mode to be disabled by default")
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Clear any existing environment variables
	clearEnvVars()

	// Reset the singleton for testing
	resetSingleton()

	// Set test environment variables
	os.Setenv("PORT", "8080")
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("APP_NAME", "Test App")
	os.Setenv("READ_TIMEOUT", "15s")
	os.Setenv("MONGODB_URL", "mongodb://test:27017")
	os.Setenv("MONGODB_DATABASE", "test_db")
	os.Setenv("NEO4J_URI", "bolt://test:7687")
	os.Setenv("NEO4J_USERNAME", "test_user")
	os.Setenv("REDIS_ADDRESS", "test:6379")
	os.Setenv("REDIS_DATABASE", "1")
	os.Setenv("MINIO_ENDPOINT", "test:9000")
	os.Setenv("MINIO_BUCKET_NAME", "test-bucket")
	os.Setenv("DEBUG_MODE", "true")
	os.Setenv("CORS_ENABLED", "false")

	defer func() {
		clearEnvVars()
	}()

	config, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	// Test environment variable overrides
	if config.Server.Port != "8080" {
		t.Errorf("expected port 8080 from env, got %s", config.Server.Port)
	}
	if config.Server.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1 from env, got %s", config.Server.Host)
	}
	if config.Server.AppName != "Test App" {
		t.Errorf("expected app name 'Test App' from env, got %s", config.Server.AppName)
	}
	if config.Server.ReadTimeout != 15*time.Second {
		t.Errorf("expected read timeout 15s from env, got %v", config.Server.ReadTimeout)
	}
	if config.MongoDB.URL != "mongodb://test:27017" {
		t.Errorf("expected MongoDB URL from env, got %s", config.MongoDB.URL)
	}
	if config.MongoDB.Database != "test_db" {
		t.Errorf("expected MongoDB database from env, got %s", config.MongoDB.Database)
	}
	if config.Neo4j.URI != "bolt://test:7687" {
		t.Errorf("expected Neo4j URI from env, got %s", config.Neo4j.URI)
	}
	if config.Neo4j.Username != "test_user" {
		t.Errorf("expected Neo4j username from env, got %s", config.Neo4j.Username)
	}
	if config.Redis.Address != "test:6379" {
		t.Errorf("expected Redis address from env, got %s", config.Redis.Address)
	}
	if config.Redis.Database != 1 {
		t.Errorf("expected Redis database 1 from env, got %d", config.Redis.Database)
	}
	if config.MinIO.Endpoint != "test:9000" {
		t.Errorf("expected MinIO endpoint from env, got %s", config.MinIO.Endpoint)
	}
	if config.MinIO.BucketName != "test-bucket" {
		t.Errorf("expected MinIO bucket name from env, got %s", config.MinIO.BucketName)
	}
	if !config.Features.DebugMode {
		t.Error("expected debug mode to be enabled from env")
	}
	if config.Security.CORSEnabled {
		t.Error("expected CORS to be disabled from env")
	}
}

func TestMustLoad_Success(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	config := MustLoad()
	if config.Server.Port != "3000" {
		t.Errorf("expected default port 3000, got %s", config.Server.Port)
	}
}

func TestGet_WithoutLoad(t *testing.T) {
	resetSingleton()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when calling Get() without Load()")
		}
	}()

	Get()
}

func TestGet_AfterLoad(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	_, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	config := Get()
	if config == nil {
		t.Error("expected config to be available after Load()")
	}
}

func TestReload(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	config1, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if config1.Server.Port != "3000" {
		t.Errorf("expected initial port 3000, got %s", config1.Server.Port)
	}

	os.Setenv("PORT", "9000")
	defer os.Unsetenv("PORT")

	config2, err := Reload()
	if err != nil {
		t.Fatalf("unexpected error reloading config: %v", err)
	}

	if config2.Server.Port != "9000" {
		t.Errorf("expected reloaded port 9000, got %s", config2.Server.Port)
	}

	config3 := Get()
	if config3.Server.Port != "9000" {
		t.Errorf("expected Get() to return reloaded port 9000, got %s", config3.Server.Port)
	}
}

func TestConcurrentAccess(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	_, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config := Get()
			if config == nil {
				t.Error("config should not be nil in concurrent access")
			}
		}()
	}

	wg.Wait()
}

func TestLoad_Singleton(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	config1, err1 := Load()
	config2, err2 := Load()

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}

	if config1 != config2 {
		t.Error("Load() should return the same instance (singleton pattern)")
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	os.Setenv("READ_TIMEOUT", "invalid_duration")
	defer os.Unsetenv("READ_TIMEOUT")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid duration, got nil")
	}
}

func TestLoad_InvalidBool(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	os.Setenv("DEBUG_MODE", "maybe")
	defer os.Unsetenv("DEBUG_MODE")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid boolean, got nil")
	}
}

func TestLoad_InvalidInt(t *testing.T) {
	clearEnvVars()
	resetSingleton()

	os.Setenv("MONGODB_MAX_POOL_SIZE", "not_a_number")
	defer os.Unsetenv("MONGODB_MAX_POOL_SIZE")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid integer, got nil")
	}
}

func clearEnvVars() {
	envVars := []string{
		// Server config
		"PORT", "HOST", "APP_NAME", "READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT",
		"TLS_ENABLED", "TLS_CERT_PATH", "TLS_KEY_PATH",

		// MongoDB config
		"MONGODB_URL", "MONGODB_DATABASE", "MONGODB_MAX_POOL_SIZE", "MONGODB_MIN_POOL_SIZE",
		"MONGODB_MAX_CONN_IDLE_TIME", "MONGODB_CONNECT_TIMEOUT", "MONGODB_SERVER_TIMEOUT",
		"MONGODB_REPLICA_SET", "MONGODB_AUTH_SOURCE", "MONGODB_TLS_ENABLED",

		// Neo4j config
		"NEO4J_URI", "NEO4J_USERNAME", "NEO4J_PASSWORD", "NEO4J_DATABASE",
		"NEO4J_MAX_CONNECTION_POOL_SIZE", "NEO4J_MAX_CONNECTION_LIFETIME",
		"NEO4J_CONNECTION_ACQUISITION_TIMEOUT", "NEO4J_CONNECTION_TIMEOUT",
		"NEO4J_MAX_TRANSACTION_RETRY_TIME", "NEO4J_ENCRYPTION_ENABLED", "NEO4J_TRUST_STRATEGY",

		// Redis config
		"REDIS_ADDRESS", "REDIS_PASSWORD", "REDIS_DATABASE", "REDIS_POOL_SIZE",
		"REDIS_MIN_IDLE_CONNS", "REDIS_MAX_CONN_AGE", "REDIS_POOL_TIMEOUT",
		"REDIS_IDLE_TIMEOUT", "REDIS_IDLE_CHECK_FREQUENCY", "REDIS_READ_TIMEOUT",
		"REDIS_WRITE_TIMEOUT", "REDIS_CLUSTER_MODE", "REDIS_TLS_ENABLED",

		// MinIO config
		"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_BUCKET_NAME",
		"MINIO_REGION", "MINIO_USE_SSL", "MINIO_AUTO_CREATE_BUCKET", "MINIO_PRESIGNED_EXPIRY",

		// Other configs
		"LOG_LEVEL", "LOG_FORMAT", "LOG_OUTPUT",
		"JWT_SECRET", "JWT_EXPIRATION", "CORS_ENABLED", "CORS_ORIGINS",
		"RATE_LIMIT_ENABLED", "RATE_LIMIT_REQUESTS", "RATE_LIMIT_WINDOW",
		"API_DOCS_ENABLED", "METRICS_ENABLED", "HEALTH_CHECK_ENABLED", "DEBUG_MODE",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

func resetSingleton() {
	mu.Lock()
	defer mu.Unlock()
	instance = nil
	once = sync.Once{}
}
