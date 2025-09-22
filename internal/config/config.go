package config

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/env"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	MongoDB  MongoDBConfig  `json:"mongodb"`
	Neo4j    Neo4jConfig    `json:"neo4j"`
	Redis    RedisConfig    `json:"redis"`
	MinIO    MinIOConfig    `json:"minio"`
	Logging  LoggingConfig  `json:"logging"`
	Security SecurityConfig `json:"security"`
	Features FeatureConfig  `json:"features"`
}

type ServerConfig struct {
	Port         string        `json:"port"`
	Host         string        `json:"host"`
	AppName      string        `json:"app_name"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	TLSEnabled   bool          `json:"tls_enabled"`
	TLSCertPath  string        `json:"tls_cert_path"`
	TLSKeyPath   string        `json:"tls_key_path"`
}

type MongoDBConfig struct {
	URL             string        `json:"url"`
	Database        string        `json:"database"`
	MaxPoolSize     int           `json:"max_pool_size"`
	MinPoolSize     int           `json:"min_pool_size"`
	MaxConnIdleTime time.Duration `json:"max_conn_idle_time"`
	ConnectTimeout  time.Duration `json:"connect_timeout"`
	ServerTimeout   time.Duration `json:"server_timeout"`
	ReplicaSet      string        `json:"replica_set"`
	AuthSource      string        `json:"auth_source"`
	TLSEnabled      bool          `json:"tls_enabled"`
}

type Neo4jConfig struct {
	URI                          string        `json:"uri"`
	Username                     string        `json:"username"`
	Password                     string        `json:"password"`
	Database                     string        `json:"database"`
	MaxConnectionPoolSize        int           `json:"max_connection_pool_size"`
	MaxConnectionLifetime        time.Duration `json:"max_connection_lifetime"`
	ConnectionAcquisitionTimeout time.Duration `json:"connection_acquisition_timeout"`
	ConnectionTimeout            time.Duration `json:"connection_timeout"`
	MaxTransactionRetryTime      time.Duration `json:"max_transaction_retry_time"`
	EncryptionEnabled            bool          `json:"encryption_enabled"`
	TrustStrategy                string        `json:"trust_strategy"`
}

type RedisConfig struct {
	Address            string        `json:"address"`
	Password           string        `json:"password"`
	Database           int           `json:"database"`
	PoolSize           int           `json:"pool_size"`
	MinIdleConns       int           `json:"min_idle_conns"`
	MaxConnAge         time.Duration `json:"max_conn_age"`
	PoolTimeout        time.Duration `json:"pool_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	IdleCheckFrequency time.Duration `json:"idle_check_frequency"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout"`
	ClusterMode        bool          `json:"cluster_mode"`
	ClusterAddresses   []string      `json:"cluster_addresses"`
	TLSEnabled         bool          `json:"tls_enabled"`
}

type MinIOConfig struct {
	Endpoint         string        `json:"endpoint"`
	AccessKey        string        `json:"access_key"`
	SecretKey        string        `json:"secret_key"`
	BucketName       string        `json:"bucket_name"`
	Region           string        `json:"region"`
	UseSSL           bool          `json:"use_ssl"`
	AutoCreateBucket bool          `json:"auto_create_bucket"`
	PresignedExpiry  time.Duration `json:"presigned_expiry"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

type SecurityConfig struct {
	JWTSecret         string        `json:"jwt_secret"`
	JWTExpiration     time.Duration `json:"jwt_expiration"`
	CORSEnabled       bool          `json:"cors_enabled"`
	CORSOrigins       string        `json:"cors_origins"`
	RateLimitEnabled  bool          `json:"rate_limit_enabled"`
	RateLimitRequests int           `json:"rate_limit_requests"`
	RateLimitWindow   time.Duration `json:"rate_limit_window"`
}

type FeatureConfig struct {
	APIDocsEnabled     bool `json:"api_docs_enabled"`
	MetricsEnabled     bool `json:"metrics_enabled"`
	HealthCheckEnabled bool `json:"health_check_enabled"`
	DebugMode          bool `json:"debug_mode"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
)

func Load() (*Config, error) {
	var err error
	once.Do(func() {
		instance, err = loadConfig()
	})
	return instance, err
}

func MustLoad() *Config {
	config, err := Load()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}
	return config
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	if instance == nil {
		panic("configuration not loaded - call Load() or MustLoad() first")
	}
	return instance
}

func Reload() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	newConfig, err := loadConfig()
	if err != nil {
		return nil, err
	}

	instance = newConfig
	return instance, nil
}

func loadConfig() (*Config, error) {
	config := &Config{}
	var err error

	config.Server.Port, err = env.Get("PORT", "3000")
	if err != nil {
		return nil, err
	}

	config.Server.Host, err = env.Get("HOST", "0.0.0.0")
	if err != nil {
		return nil, err
	}

	config.Server.AppName, err = env.Get("APP_NAME", "Relational Knowledge Engineering Platform")
	if err != nil {
		return nil, err
	}

	config.Server.ReadTimeout, err = env.Get("READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	config.Server.WriteTimeout, err = env.Get("WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	config.Server.IdleTimeout, err = env.Get("IDLE_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}

	config.Server.TLSEnabled, err = env.Get("TLS_ENABLED", false)
	if err != nil {
		return nil, err
	}

	config.Server.TLSCertPath, err = env.Get("TLS_CERT_PATH", "")
	if err != nil {
		return nil, err
	}

	config.Server.TLSKeyPath, err = env.Get("TLS_KEY_PATH", "")
	if err != nil {
		return nil, err
	}

	config.MongoDB.URL, err = env.Get("MONGODB_URL", "mongodb://localhost:27017")
	if err != nil {
		return nil, err
	}

	config.MongoDB.Database, err = env.Get("MONGODB_DATABASE", "knowledge_platform")
	if err != nil {
		return nil, err
	}

	config.MongoDB.MaxPoolSize, err = env.Get("MONGODB_MAX_POOL_SIZE", 100)
	if err != nil {
		return nil, err
	}

	config.MongoDB.MinPoolSize, err = env.Get("MONGODB_MIN_POOL_SIZE", 0)
	if err != nil {
		return nil, err
	}

	config.MongoDB.MaxConnIdleTime, err = env.Get("MONGODB_MAX_CONN_IDLE_TIME", 10*time.Minute)
	if err != nil {
		return nil, err
	}

	config.MongoDB.ConnectTimeout, err = env.Get("MONGODB_CONNECT_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	config.MongoDB.ServerTimeout, err = env.Get("MONGODB_SERVER_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}

	config.MongoDB.ReplicaSet, err = env.Get("MONGODB_REPLICA_SET", "")
	if err != nil {
		return nil, err
	}

	config.MongoDB.AuthSource, err = env.Get("MONGODB_AUTH_SOURCE", "admin")
	if err != nil {
		return nil, err
	}

	config.MongoDB.TLSEnabled, err = env.Get("MONGODB_TLS_ENABLED", false)
	if err != nil {
		return nil, err
	}

	config.Neo4j.URI, err = env.Get("NEO4J_URI", "bolt://localhost:7687")
	if err != nil {
		return nil, err
	}

	config.Neo4j.Username, err = env.Get("NEO4J_USERNAME", "neo4j")
	if err != nil {
		return nil, err
	}

	config.Neo4j.Password, err = env.Get("NEO4J_PASSWORD", "password")
	if err != nil {
		return nil, err
	}

	config.Neo4j.Database, err = env.Get("NEO4J_DATABASE", "neo4j")
	if err != nil {
		return nil, err
	}

	config.Neo4j.MaxConnectionPoolSize, err = env.Get("NEO4J_MAX_CONNECTION_POOL_SIZE", 100)
	if err != nil {
		return nil, err
	}

	config.Neo4j.MaxConnectionLifetime, err = env.Get("NEO4J_MAX_CONNECTION_LIFETIME", 1*time.Hour)
	if err != nil {
		return nil, err
	}

	config.Neo4j.ConnectionAcquisitionTimeout, err = env.Get("NEO4J_CONNECTION_ACQUISITION_TIMEOUT", 60*time.Second)
	if err != nil {
		return nil, err
	}

	config.Neo4j.ConnectionTimeout, err = env.Get("NEO4J_CONNECTION_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}

	config.Neo4j.MaxTransactionRetryTime, err = env.Get("NEO4J_MAX_TRANSACTION_RETRY_TIME", 30*time.Second)
	if err != nil {
		return nil, err
	}

	config.Neo4j.EncryptionEnabled, err = env.Get("NEO4J_ENCRYPTION_ENABLED", false)
	if err != nil {
		return nil, err
	}

	config.Neo4j.TrustStrategy, err = env.Get("NEO4J_TRUST_STRATEGY", "TRUST_ALL_CERTIFICATES")
	if err != nil {
		return nil, err
	}

	config.Redis.Address, err = env.Get("REDIS_ADDRESS", "localhost:6379")
	if err != nil {
		return nil, err
	}

	config.Redis.Password, err = env.Get("REDIS_PASSWORD", "")
	if err != nil {
		return nil, err
	}

	config.Redis.Database, err = env.Get("REDIS_DATABASE", 0)
	if err != nil {
		return nil, err
	}

	config.Redis.PoolSize, err = env.Get("REDIS_POOL_SIZE", 10)
	if err != nil {
		return nil, err
	}

	config.Redis.MinIdleConns, err = env.Get("REDIS_MIN_IDLE_CONNS", 5)
	if err != nil {
		return nil, err
	}

	config.Redis.MaxConnAge, err = env.Get("REDIS_MAX_CONN_AGE", 30*time.Minute)
	if err != nil {
		return nil, err
	}

	config.Redis.PoolTimeout, err = env.Get("REDIS_POOL_TIMEOUT", 4*time.Second)
	if err != nil {
		return nil, err
	}

	config.Redis.IdleTimeout, err = env.Get("REDIS_IDLE_TIMEOUT", 5*time.Minute)
	if err != nil {
		return nil, err
	}

	config.Redis.IdleCheckFrequency, err = env.Get("REDIS_IDLE_CHECK_FREQUENCY", 1*time.Minute)
	if err != nil {
		return nil, err
	}

	config.Redis.ReadTimeout, err = env.Get("REDIS_READ_TIMEOUT", 3*time.Second)
	if err != nil {
		return nil, err
	}

	config.Redis.WriteTimeout, err = env.Get("REDIS_WRITE_TIMEOUT", 3*time.Second)
	if err != nil {
		return nil, err
	}

	config.Redis.ClusterMode, err = env.Get("REDIS_CLUSTER_MODE", false)
	if err != nil {
		return nil, err
	}

	config.Redis.TLSEnabled, err = env.Get("REDIS_TLS_ENABLED", false)
	if err != nil {
		return nil, err
	}

	config.MinIO.Endpoint, err = env.Get("MINIO_ENDPOINT", "localhost:9000")
	if err != nil {
		return nil, err
	}

	config.MinIO.AccessKey, err = env.Get("MINIO_ACCESS_KEY", "minioadmin")
	if err != nil {
		return nil, err
	}

	config.MinIO.SecretKey, err = env.Get("MINIO_SECRET_KEY", "minioadmin")
	if err != nil {
		return nil, err
	}

	config.MinIO.BucketName, err = env.Get("MINIO_BUCKET_NAME", "knowledge-platform")
	if err != nil {
		return nil, err
	}

	config.MinIO.Region, err = env.Get("MINIO_REGION", "us-east-1")
	if err != nil {
		return nil, err
	}

	config.MinIO.UseSSL, err = env.Get("MINIO_USE_SSL", false)
	if err != nil {
		return nil, err
	}

	config.MinIO.AutoCreateBucket, err = env.Get("MINIO_AUTO_CREATE_BUCKET", true)
	if err != nil {
		return nil, err
	}

	config.MinIO.PresignedExpiry, err = env.Get("MINIO_PRESIGNED_EXPIRY", 24*time.Hour)
	if err != nil {
		return nil, err
	}

	config.Logging.Level, err = env.Get("LOG_LEVEL", "info")
	if err != nil {
		return nil, err
	}

	config.Logging.Format, err = env.Get("LOG_FORMAT", "console")
	if err != nil {
		return nil, err
	}

	config.Logging.Output, err = env.Get("LOG_OUTPUT", "stderr")
	if err != nil {
		return nil, err
	}

	config.Security.JWTSecret, err = env.Get("JWT_SECRET", "your-secret-key-change-in-production")
	if err != nil {
		return nil, err
	}

	config.Security.JWTExpiration, err = env.Get("JWT_EXPIRATION", 24*time.Hour)
	if err != nil {
		return nil, err
	}

	config.Security.CORSEnabled, err = env.Get("CORS_ENABLED", true)
	if err != nil {
		return nil, err
	}

	config.Security.CORSOrigins, err = env.Get("CORS_ORIGINS", "*")
	if err != nil {
		return nil, err
	}

	config.Security.RateLimitEnabled, err = env.Get("RATE_LIMIT_ENABLED", false)
	if err != nil {
		return nil, err
	}

	config.Security.RateLimitRequests, err = env.Get("RATE_LIMIT_REQUESTS", 100)
	if err != nil {
		return nil, err
	}

	config.Security.RateLimitWindow, err = env.Get("RATE_LIMIT_WINDOW", 1*time.Minute)
	if err != nil {
		return nil, err
	}

	config.Features.APIDocsEnabled, err = env.Get("API_DOCS_ENABLED", true)
	if err != nil {
		return nil, err
	}

	config.Features.MetricsEnabled, err = env.Get("METRICS_ENABLED", false)
	if err != nil {
		return nil, err
	}

	config.Features.HealthCheckEnabled, err = env.Get("HEALTH_CHECK_ENABLED", true)
	if err != nil {
		return nil, err
	}

	config.Features.DebugMode, err = env.Get("DEBUG_MODE", false)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if err := c.MongoDB.Validate(); err != nil {
		return fmt.Errorf("mongoDB config validation failed: %w", err)
	}

	if err := c.Neo4j.Validate(); err != nil {
		return fmt.Errorf("neo4j config validation failed: %w", err)
	}

	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis config validation failed: %w", err)
	}

	if err := c.MinIO.Validate(); err != nil {
		return fmt.Errorf("minIO config validation failed: %w", err)
	}

	return nil
}

func (m *MongoDBConfig) Validate() error {
	if m.URL == "" {
		return fmt.Errorf("mongoDB URL is required")
	}

	if m.Database == "" {
		return fmt.Errorf("mongoDB database name is required")
	}

	if m.MaxPoolSize <= 0 {
		return fmt.Errorf("mongoDB max pool size must be positive")
	}

	if m.ConnectTimeout <= 0 {
		return fmt.Errorf("mongoDB connect timeout must be positive")
	}

	return nil
}

func (n *Neo4jConfig) Validate() error {
	if n.URI == "" {
		return fmt.Errorf("neo4j URI is required")
	}

	if n.Username == "" {
		return fmt.Errorf("neo4j username is required")
	}

	if n.Password == "" {
		return fmt.Errorf("neo4j password is required")
	}

	if n.Database == "" {
		return fmt.Errorf("neo4j database name is required")
	}

	if n.MaxConnectionPoolSize <= 0 {
		return fmt.Errorf("neo4j max connection pool size must be positive")
	}

	if n.ConnectionTimeout <= 0 {
		return fmt.Errorf("neo4j connection timeout must be positive")
	}

	validTrustStrategies := []string{"TRUST_ALL_CERTIFICATES", "TRUST_SYSTEM_CA_SIGNED_CERTIFICATES", "TRUST_CUSTOM_CA_SIGNED_CERTIFICATES"}
	if !slices.Contains(validTrustStrategies, n.TrustStrategy) {
		return fmt.Errorf("neo4j trust strategy must be one of: %v", validTrustStrategies)
	}

	return nil
}

func (r *RedisConfig) Validate() error {
	if r.Address == "" {
		return fmt.Errorf("redis address is required")
	}

	if r.Database < 0 {
		return fmt.Errorf("redis database number must be non-negative")
	}

	if r.PoolSize <= 0 {
		return fmt.Errorf("redis pool size must be positive")
	}

	if r.ReadTimeout <= 0 {
		return fmt.Errorf("redis read timeout must be positive")
	}

	if r.WriteTimeout <= 0 {
		return fmt.Errorf("redis write timeout must be positive")
	}

	if r.ClusterMode && len(r.ClusterAddresses) == 0 {
		return fmt.Errorf("redis cluster addresses must be provided when cluster mode is enabled")
	}

	return nil
}

func (m *MinIOConfig) Validate() error {
	if m.Endpoint == "" {
		return fmt.Errorf("minIO endpoint is required")
	}

	if m.AccessKey == "" {
		return fmt.Errorf("minIO access key is required")
	}

	if m.SecretKey == "" {
		return fmt.Errorf("minIO secret key is required")
	}

	if m.BucketName == "" {
		return fmt.Errorf("minIO bucket name is required")
	}

	if m.PresignedExpiry <= 0 {
		return fmt.Errorf("minIO presigned expiry must be positive")
	}

	return nil
}
