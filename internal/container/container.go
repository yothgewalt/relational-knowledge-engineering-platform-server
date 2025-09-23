package container

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/log"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/redis"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
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

	mongoClient  *mongo.MongoService
	redisClient  redis.RedisService
	resendClient resend.ResendService

	vaultClient vault.VaultService

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
		startTime:     time.Now(),
		shutdownFuncs: make([]func() error, 0),
		ctx:           ctx,
		cancel:        cancel,
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

	if err := c.initializeRedis(); err != nil {
		c.logger.Warn().Err(err).Msg("Redis initialization failed, cache service will not be available")
	}

	if err := c.initializeResend(); err != nil {
		c.logger.Warn().Err(err).Msg("Resend initialization failed, email service will not be available")
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
		return fmt.Errorf("Redis health check failed: %s", health.Error)
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

	// Add debug logging to understand what API key we're getting
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
