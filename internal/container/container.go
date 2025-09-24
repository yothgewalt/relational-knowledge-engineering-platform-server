package container

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/rs/zerolog"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/jwt"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/log"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/minio"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/neo4j"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/redis"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/telemetry"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/vault"
)

type ServiceHealth struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type HealthStatus struct {
	Status   string          `json:"status"`
	Services []ServiceHealth `json:"services"`
	Uptime   time.Duration   `json:"uptime"`
}

type Container struct {
	config *config.Config
	logger zerolog.Logger

	mongoClient     *mongo.MongoService
	neo4jClient     neo4j.Neo4jService
	redisClient     redis.RedisService
	minioClient     minio.MinIOService
	telemetryClient telemetry.TelemetryService
	resendClient    resend.ResendService
	vaultClient     vault.VaultService
	jwtService      *jwt.JWTService

	registry       *ServiceRegistry
	moduleManager  *ModuleManager
	pendingModules []Module

	app *fiber.App

	startTime time.Time
	mu        sync.RWMutex
	running   bool

	shutdownFuncs []func() error
	ctx           context.Context
	cancel        context.CancelFunc
}

type Options struct {
	DisableVault  bool
	DisableResend bool
	Timezone      string
}

func New(opts *Options) *Container {
	if opts == nil {
		opts = &Options{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Container{
		startTime:      time.Now(),
		shutdownFuncs:  make([]func() error, 0),
		pendingModules: make([]Module, 0),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (c *Container) Bootstrap() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("container is already running")
	}

	if err := c.initializeTimezone(); err != nil {
		return fmt.Errorf("failed to initialize timezone: %w", err)
	}

	if err := c.initializeConfig(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	if err := c.initializeLogging(); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	if err := c.initializeVault(); err != nil {
		c.logger.Warn().Err(err).Msg("Vault initialization failed, continuing with environment variable configuration")
	}

	if err := c.reloadConfigWithVault(); err != nil {
		c.logger.Warn().Err(err).Msg("Failed to reload config with Vault, continuing with env vars")
	}

	c.logger.Info().Msg("Starting application bootstrap...")

	if err := c.initializeMongoDB(); err != nil {
		return fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	if err := c.initializeNeo4j(); err != nil {
		c.logger.Warn().Err(err).Msg("Neo4j initialization failed, graph database service will not be available")
	}

	if err := c.initializeRedis(); err != nil {
		c.logger.Warn().Err(err).Msg("Redis initialization failed, cache service will not be available")
	}

	if err := c.initializeMinIO(); err != nil {
		c.logger.Warn().Err(err).Msg("MinIO initialization failed, file storage service will not be available")
	}

	if err := c.initializeTelemetry(); err != nil {
		c.logger.Warn().Err(err).Msg("Telemetry initialization failed, tracing service will not be available")
	}

	if err := c.initializeResend(); err != nil {
		c.logger.Warn().Err(err).Msg("Resend initialization failed, email service will not be available")
	}

	if err := c.initializeJWT(); err != nil {
		return fmt.Errorf("failed to initialize JWT service: %w", err)
	}

	if err := c.initializeRegistry(); err != nil {
		return fmt.Errorf("failed to initialize service registry: %w", err)
	}

	if err := c.initializeServices(); err != nil {
		return fmt.Errorf("failed to initialize application services: %w", err)
	}

	if err := c.initializeRouter(); err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}

	if err := c.startServer(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	c.running = true
	c.logger.Info().Msg("Application bootstrap completed successfully")

	return nil
}

func (c *Container) initializeTimezone() error {
	timezone := os.Getenv("TZ")
	if timezone == "" {
		timezone = "UTC"
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone '%s': %w", timezone, err)
	}

	time.Local = location
	return nil
}

func (c *Container) initializeConfig() error {
	var cfg *config.Config
	var err error

	if c.vaultClient != nil {
		cfg, err = config.LoadWithVault(c.vaultClient)
	} else {
		cfg, err = config.Load()
	}

	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	c.config = cfg
	return nil
}

func (c *Container) initializeLogging() error {
	logger := log.New()
	c.logger = logger
	return nil
}

func (c *Container) initializeMongoDB() error {
	mongoConfig := mongo.MongoConfig{
		Address:  c.config.Mongo.Address,
		Username: c.config.Mongo.Username,
		Password: c.config.Mongo.Password,
		Database: c.config.Mongo.Database,
	}

	c.logger.Info().
		Str("address", mongoConfig.Address).
		Str("username", mongoConfig.Username).
		Str("database", mongoConfig.Database).
		Msg("Attempting MongoDB connection")

	client, err := mongo.NewMongoService(mongoConfig)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("address", mongoConfig.Address).
			Str("database", mongoConfig.Database).
			Msg("Failed to create MongoDB client")
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	c.logger.Debug().Msg("MongoDB client created successfully, performing health check")

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)

	c.logger.Info().
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("database_exists", health.DatabaseExists).
		Dur("latency", health.Latency).
		Str("database", health.Database).
		Str("error", health.Error).
		Msg("MongoDB health check completed")

	if !health.Connected || !health.Authenticated || !health.DatabaseExists {
		c.logger.Error().
			Str("error", health.Error).
			Str("address", mongoConfig.Address).
			Str("database", mongoConfig.Database).
			Bool("connected", health.Connected).
			Bool("authenticated", health.Authenticated).
			Bool("database_exists", health.DatabaseExists).
			Msg("MongoDB health check failed")
		return fmt.Errorf("MongoDB health check failed: %s", health.Error)
	}

	c.mongoClient = client
	c.addShutdownFunc(func() error {
		return client.Close(context.Background())
	})

	c.logger.Info().
		Str("address", mongoConfig.Address).
		Str("database", mongoConfig.Database).
		Str("username", mongoConfig.Username).
		Dur("latency", health.Latency).
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("database_exists", health.DatabaseExists).
		Msg("MongoDB connection fully established - connected, authenticated, and database accessible")

	return nil
}

func (c *Container) initializeNeo4j() error {
	neo4jConfig := neo4j.Neo4jConfig{
		URI:      c.config.Neo4j.URI,
		Username: c.config.Neo4j.Username,
		Password: c.config.Neo4j.Password,
		Database: c.config.Neo4j.Database,
	}

	c.logger.Info().
		Str("uri", neo4jConfig.URI).
		Str("username", neo4jConfig.Username).
		Str("database", neo4jConfig.Database).
		Msg("Attempting Neo4j connection")

	client, err := neo4j.NewNeo4jService(neo4jConfig)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("uri", neo4jConfig.URI).
			Str("database", neo4jConfig.Database).
			Msg("Failed to create Neo4j client")
		return fmt.Errorf("failed to create Neo4j client: %w", err)
	}

	c.logger.Debug().Msg("Neo4j client created successfully, performing health check")

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)

	c.logger.Info().
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("database_exists", health.DatabaseExists).
		Dur("latency", health.Latency).
		Str("uri", health.URI).
		Str("database", health.Database).
		Str("error", health.Error).
		Msg("Neo4j health check completed")

	if !health.Connected || !health.Authenticated || !health.DatabaseExists {
		c.logger.Error().
			Str("error", health.Error).
			Str("uri", neo4jConfig.URI).
			Str("database", neo4jConfig.Database).
			Bool("connected", health.Connected).
			Bool("authenticated", health.Authenticated).
			Bool("database_exists", health.DatabaseExists).
			Msg("Neo4j health check failed")
		return fmt.Errorf("Neo4j health check failed: %s", health.Error)
	}

	c.neo4jClient = client
	c.addShutdownFunc(func() error {
		return client.Close()
	})

	c.logger.Info().
		Str("uri", neo4jConfig.URI).
		Str("database", neo4jConfig.Database).
		Str("username", neo4jConfig.Username).
		Dur("latency", health.Latency).
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("database_exists", health.DatabaseExists).
		Msg("Neo4j connection fully established - connected, authenticated, and database accessible")

	return nil
}

func (c *Container) initializeRedis() error {
	redisConfig := redis.RedisConfig{
		Address:  c.config.Redis.Address,
		Password: c.config.Redis.Password,
		Database: c.config.Redis.Database,
	}

	c.logger.Info().
		Str("address", redisConfig.Address).
		Int("database", redisConfig.Database).
		Msg("Attempting Redis connection")

	client, err := redis.NewRedisService(redisConfig)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("address", redisConfig.Address).
			Int("database", redisConfig.Database).
			Msg("Failed to create Redis client")
		return fmt.Errorf("failed to create Redis client: %w", err)
	}

	c.logger.Debug().Msg("Redis client created successfully, performing health check")

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)

	c.logger.Info().
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("database_exists", health.DatabaseExists).
		Dur("latency", health.Latency).
		Str("address", health.Address).
		Int("database", health.Database).
		Str("error", health.Error).
		Msg("Redis health check completed")

	if !health.Connected || !health.Authenticated || !health.DatabaseExists {
		c.logger.Error().
			Str("error", health.Error).
			Str("address", redisConfig.Address).
			Int("database", redisConfig.Database).
			Bool("connected", health.Connected).
			Bool("authenticated", health.Authenticated).
			Bool("database_exists", health.DatabaseExists).
			Msg("Redis health check failed")

		return fmt.Errorf("redis health check failed: %s", health.Error)
	}

	c.redisClient = client
	c.addShutdownFunc(func() error {
		return client.Close()
	})

	c.logger.Info().
		Str("address", redisConfig.Address).
		Int("database", redisConfig.Database).
		Dur("latency", health.Latency).
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("database_exists", health.DatabaseExists).
		Msg("Redis connection fully established - connected, authenticated, and database accessible")

	return nil
}

func (c *Container) initializeMinIO() error {
	minioConfig := minio.MinIOConfig{
		Endpoint:        c.config.MinIO.Endpoint,
		AccessKeyID:     c.config.MinIO.AccessKey,
		SecretAccessKey: c.config.MinIO.SecretKey,
		UseSSL:          c.config.MinIO.UseSSL,
		BucketName:      c.config.MinIO.BucketName,
	}

	c.logger.Info().
		Str("endpoint", minioConfig.Endpoint).
		Str("access_key", minioConfig.AccessKeyID).
		Bool("use_ssl", minioConfig.UseSSL).
		Str("bucket_name", minioConfig.BucketName).
		Msg("Attempting MinIO connection")

	client, err := minio.NewMinIOService(minioConfig)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("endpoint", minioConfig.Endpoint).
			Str("bucket_name", minioConfig.BucketName).
			Msg("Failed to create MinIO client")
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	c.logger.Debug().Msg("MinIO client created successfully, performing health check")

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)

	c.logger.Info().
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("bucket_exists", health.BucketExists).
		Dur("latency", health.Latency).
		Str("endpoint", health.Endpoint).
		Str("bucket_name", health.BucketName).
		Str("error", health.Error).
		Msg("MinIO health check completed")

	if !health.Connected || !health.Authenticated || !health.BucketExists {
		c.logger.Error().
			Str("error", health.Error).
			Str("endpoint", minioConfig.Endpoint).
			Str("bucket_name", minioConfig.BucketName).
			Bool("connected", health.Connected).
			Bool("authenticated", health.Authenticated).
			Bool("bucket_exists", health.BucketExists).
			Msg("MinIO health check failed")
		return fmt.Errorf("MinIO health check failed: %s", health.Error)
	}

	c.minioClient = client
	c.addShutdownFunc(func() error {
		return client.Close()
	})

	c.logger.Info().
		Str("endpoint", minioConfig.Endpoint).
		Str("bucket_name", minioConfig.BucketName).
		Str("access_key", minioConfig.AccessKeyID).
		Dur("latency", health.Latency).
		Bool("connected", health.Connected).
		Bool("authenticated", health.Authenticated).
		Bool("bucket_exists", health.BucketExists).
		Msg("MinIO connection fully established - connected, authenticated, and bucket accessible")

	return nil
}

func (c *Container) initializeTelemetry() error {
	telemetryConfig := telemetry.TelemetryConfig{
		ServiceName:    c.config.Telemetry.ServiceName,
		ServiceVersion: c.config.Telemetry.ServiceVersion,
		Environment:    c.config.Telemetry.Environment,
		Enabled:        c.config.Telemetry.Enabled,
		JaegerEndpoint: c.config.Telemetry.JaegerEndpoint,
		OTLPEndpoint:   c.config.Telemetry.OTLPEndpoint,
		SamplingRatio:  c.config.Telemetry.SamplingRatio,
		ExporterType:   c.config.Telemetry.ExporterType,
	}

	c.logger.Info().
		Str("service_name", telemetryConfig.ServiceName).
		Str("service_version", telemetryConfig.ServiceVersion).
		Str("environment", telemetryConfig.Environment).
		Bool("enabled", telemetryConfig.Enabled).
		Str("exporter_type", telemetryConfig.ExporterType).
		Float64("sampling_ratio", telemetryConfig.SamplingRatio).
		Msg("Attempting telemetry initialization")

	client, err := telemetry.NewTelemetryService(telemetryConfig)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("service_name", telemetryConfig.ServiceName).
			Str("exporter_type", telemetryConfig.ExporterType).
			Msg("Failed to create telemetry client")
		return fmt.Errorf("failed to create telemetry client: %w", err)
	}

	c.logger.Debug().Msg("Telemetry client created successfully, performing health check")

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)

	c.logger.Info().
		Bool("enabled", health.Enabled).
		Str("service_name", health.ServiceName).
		Str("exporter_type", health.ExporterType).
		Float64("sampling_ratio", health.SamplingRatio).
		Dur("latency", health.Latency).
		Str("error", health.Error).
		Msg("Telemetry health check completed")

	if health.Error != "" {
		c.logger.Error().
			Str("error", health.Error).
			Str("service_name", telemetryConfig.ServiceName).
			Str("exporter_type", telemetryConfig.ExporterType).
			Bool("enabled", health.Enabled).
			Msg("Telemetry health check failed")
		return fmt.Errorf("telemetry health check failed: %s", health.Error)
	}

	c.telemetryClient = client
	c.addShutdownFunc(func() error {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		return client.Shutdown(shutdownCtx)
	})

	c.logger.Info().
		Str("service_name", telemetryConfig.ServiceName).
		Str("service_version", telemetryConfig.ServiceVersion).
		Str("environment", telemetryConfig.Environment).
		Str("exporter_type", telemetryConfig.ExporterType).
		Float64("sampling_ratio", telemetryConfig.SamplingRatio).
		Bool("enabled", health.Enabled).
		Dur("latency", health.Latency).
		Msg("Telemetry service fully established and ready for tracing")

	return nil
}

func (c *Container) initializeVault() error {
	if c.config == nil {
		return fmt.Errorf("config is not loaded")
	}

	vaultConfig := c.config.Vault
	if vaultConfig.Address == "" || vaultConfig.Token == "" {
		c.logger.Info().Msg("Vault configuration not provided, skipping Vault initialization")
		return nil
	}

	clientConfig := vault.VaultConfig{
		Address: vaultConfig.Address,
		Token:   vaultConfig.Token,
	}

	client, err := vault.NewVaultClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create Vault client: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)
	if !health.Connected {
		return fmt.Errorf("vault health check failed: %s", health.Error)
	}

	c.vaultClient = client
	c.addShutdownFunc(client.Close)

	c.logger.Info().
		Str("address", vaultConfig.Address).
		Bool("authenticated", health.Authenticated).
		Msg("Vault connection established")

	return nil
}

func (c *Container) initializeResend() error {
	if c.config == nil {
		return fmt.Errorf("config is not loaded")
	}

	resendConfig := c.config.Resend

	apiKeyMasked := ""
	if resendConfig.ApiKey != "" {
		if len(resendConfig.ApiKey) > 8 {
			apiKeyMasked = resendConfig.ApiKey[:4] + "***" + resendConfig.ApiKey[len(resendConfig.ApiKey)-4:]
		} else {
			apiKeyMasked = "***"
		}
	}

	c.logger.Debug().
		Str("api_key_masked", apiKeyMasked).
		Bool("api_key_empty", resendConfig.ApiKey == "").
		Msg("Resend configuration loaded from config")

	if resendConfig.ApiKey == "" {
		c.logger.Info().Msg("Resend API key not configured, skipping Resend initialization")
		return nil
	}

	clientConfig := resend.ResendConfig{
		ApiKey: resendConfig.ApiKey,
	}

	client, err := resend.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create Resend client: %w", err)
	}

	healthCtx, healthCancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer healthCancel()

	health := client.HealthCheck(healthCtx)
	if !health.Connected {
		return fmt.Errorf("resend health check failed: %s", health.Error)
	}

	c.resendClient = client
	c.addShutdownFunc(client.Close)

	c.logger.Info().
		Str("api_key", health.ApiKey).
		Msg("Resend client initialized successfully")

	return nil
}

func (c *Container) reloadConfigWithVault() error {
	if c.vaultClient == nil {
		c.logger.Info().Msg("Vault client not available, skipping config reload with Vault")
		return nil
	}

	updatedConfig, err := config.ReloadWithVault(c.vaultClient)
	if err != nil {
		return fmt.Errorf("failed to reload config with Vault: %w", err)
	}

	c.config = updatedConfig
	c.logger.Info().Msg("Config reloaded with Vault secrets")

	return nil
}

func (c *Container) addShutdownFunc(fn func() error) {
	c.shutdownFuncs = append(c.shutdownFuncs, fn)
}

func (c *Container) GetConfig() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *Container) GetLogger() zerolog.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logger
}

func (c *Container) GetMongoClient() *mongo.MongoService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mongoClient
}

func (c *Container) GetVaultClient() vault.VaultService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.vaultClient
}

func (c *Container) GetRedisClient() redis.RedisService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.redisClient
}

func (c *Container) GetNeo4jClient() neo4j.Neo4jService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.neo4jClient
}

func (c *Container) GetMinIOClient() minio.MinIOService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.minioClient
}

func (c *Container) GetTelemetryClient() telemetry.TelemetryService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.telemetryClient
}

func (c *Container) GetResendClient() resend.ResendService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.resendClient
}

func (c *Container) HealthCheck() HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	services := make([]ServiceHealth, 0)
	overallStatus := "healthy"

	if c.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.mongoClient.HealthCheck(ctx)
		cancel()

		status := "healthy"
		message := ""
		if !health.Connected || !health.Authenticated || !health.DatabaseExists {
			status = "unhealthy"
			message = health.Error
			overallStatus = "unhealthy"
		}

		services = append(services, ServiceHealth{
			Name:      "mongodb",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		})
	}

	if c.neo4jClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.neo4jClient.HealthCheck(ctx)
		cancel()

		status := "healthy"
		message := ""
		if !health.Connected || !health.Authenticated || !health.DatabaseExists {
			status = "unhealthy"
			message = health.Error
			overallStatus = "unhealthy"
		}

		services = append(services, ServiceHealth{
			Name:      "neo4j",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		})
	}

	if c.redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.redisClient.HealthCheck(ctx)
		cancel()

		status := "healthy"
		message := ""
		if !health.Connected || !health.Authenticated || !health.DatabaseExists {
			status = "unhealthy"
			message = health.Error
			overallStatus = "unhealthy"
		}

		services = append(services, ServiceHealth{
			Name:      "redis",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		})
	}

	if c.minioClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.minioClient.HealthCheck(ctx)
		cancel()

		status := "healthy"
		message := ""
		if !health.Connected || !health.Authenticated || !health.BucketExists {
			status = "unhealthy"
			message = health.Error
			if overallStatus != "unhealthy" {
				overallStatus = "degraded"
			}
		}

		services = append(services, ServiceHealth{
			Name:      "minio",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		})
	}

	if c.telemetryClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.telemetryClient.HealthCheck(ctx)
		cancel()

		status := "healthy"
		message := ""
		if health.Error != "" {
			status = "unhealthy"
			message = health.Error
			if overallStatus != "unhealthy" {
				overallStatus = "degraded"
			}
		}

		services = append(services, ServiceHealth{
			Name:      "telemetry",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		})
	}

	if c.vaultClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.vaultClient.HealthCheck(ctx)
		cancel()

		status := "healthy"
		message := ""
		if !health.Connected {
			status = "unhealthy"
			message = health.Error
			if overallStatus != "unhealthy" {
				overallStatus = "degraded"
			}
		}

		services = append(services, ServiceHealth{
			Name:      "vault",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		})
	}

	if c.resendClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.resendClient.HealthCheck(ctx)
		cancel()

		status := "healthy"
		message := ""
		if !health.Connected {
			status = "unhealthy"
			message = health.Error
			if overallStatus != "unhealthy" {
				overallStatus = "degraded"
			}
		}

		services = append(services, ServiceHealth{
			Name:      "resend",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
		})
	}

	return HealthStatus{
		Status:   overallStatus,
		Services: services,
		Uptime:   time.Since(c.startTime),
	}
}

func (c *Container) WaitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	c.logger.Info().Str("signal", sig.String()).Msg("Shutdown signal received")

	if err := c.Shutdown(); err != nil {
		c.logger.Error().Err(err).Msg("Error during shutdown")
	}
}

func (c *Container) Shutdown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.logger.Info().Msg("Starting graceful shutdown...")

	c.cancel()

	var errors []error

	for i := len(c.shutdownFuncs) - 1; i >= 0; i-- {
		if err := c.shutdownFuncs[i](); err != nil {
			errors = append(errors, err)
			c.logger.Error().Err(err).Msg("Error during service shutdown")
		}
	}

	c.running = false
	c.logger.Info().Msg("Graceful shutdown completed")

	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors", len(errors))
	}

	return nil
}

func (c *Container) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *Container) initializeJWT() error {
	jwtConfig := jwt.JWTConfig{
		SecretKey:     os.Getenv("JWT_SECRET_KEY"),
		TokenDuration: 24 * time.Hour,
		Issuer:        "relational-knowledge-platform",
	}

	if jwtConfig.SecretKey == "" {
		jwtConfig.SecretKey = "default-secret-key-change-in-production"
		c.logger.Warn().Msg("JWT secret key not set, using default (change in production)")
	}

	jwtService, err := jwt.NewJWTService(jwtConfig)
	if err != nil {
		return fmt.Errorf("failed to create JWT service: %w", err)
	}

	c.jwtService = jwtService
	c.logger.Info().Msg("JWT service initialized")

	return nil
}

func (c *Container) initializeRegistry() error {
	c.registry = NewServiceRegistry(c.logger)
	c.moduleManager = NewModuleManager(c.registry, c.logger)

	c.registry.RegisterInfrastructure(
		c.mongoClient,
		c.redisClient,
		c.neo4jClient,
		c.minioClient,
		c.telemetryClient,
		c.resendClient,
		c.vaultClient,
		c.jwtService,
	)

	for _, module := range c.pendingModules {
		if err := c.moduleManager.RegisterModule(module); err != nil {
			return fmt.Errorf("failed to register pending module: %w", err)
		}
	}
	c.pendingModules = nil

	c.logger.Info().Msg("Service registry and module manager initialized")
	return nil
}

func (c *Container) RegisterModule(module Module) error {
	if c.moduleManager == nil {
		c.pendingModules = append(c.pendingModules, module)
		return nil
	}
	return c.moduleManager.RegisterModule(module)
}

func (c *Container) initializeServices() error {
	if err := c.moduleManager.InitializeServices(); err != nil {
		return fmt.Errorf("failed to initialize module services: %w", err)
	}

	if err := c.moduleManager.InitializeMiddleware(); err != nil {
		return fmt.Errorf("failed to initialize module middleware: %w", err)
	}

	c.logger.Info().Msg("All services initialized via module system")
	return nil
}

func (c *Container) initializeRouter() error {
	c.app = fiber.New(fiber.Config{
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return ctx.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
		DisableStartupMessage: true,
	})

	apiV1 := c.app.Group("/api/v1")

	if err := c.moduleManager.InitializeRoutes(apiV1); err != nil {
		return fmt.Errorf("failed to initialize routes: %w", err)
	}

	c.app.Get("/swagger/*", swagger.HandlerDefault)
	c.logger.Info().Msg("API documentation available at: http://localhost:3000/swagger/")

	c.logger.Info().Msg("Router initialized via module system")
	return nil
}

func (c *Container) startServer() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	c.logger.Info().Str("port", port).Msg("Starting HTTP server")

	go func() {
		if err := c.app.Listen(":" + port); err != nil {
			c.logger.Error().Err(err).Msg("Failed to start server")
		}
	}()

	c.addShutdownFunc(func() error {
		return c.app.Shutdown()
	})

	return nil
}
