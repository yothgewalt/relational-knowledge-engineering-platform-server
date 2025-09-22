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
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/consul"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/env"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/log"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
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

	mongoClient *mongo.MongoService

	vaultClient  vault.VaultService
	consulClient consul.ConsulService

	startTime time.Time
	mu        sync.RWMutex
	running   bool

	shutdownFuncs []func() error
	ctx           context.Context
	cancel        context.CancelFunc
}

type ContainerOptions struct {
	DisableVault  bool
	DisableConsul bool
	Timezone      string
}

func NewContainer(opts *ContainerOptions) *Container {
	if opts == nil {
		opts = &ContainerOptions{}
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

	if err := c.initializeVault(); err != nil {
		fmt.Printf("Warning: Vault initialization failed: %v\n", err)
		fmt.Printf("Continuing with environment variable configuration...\n")
	}

	if err := c.initializeConfig(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	if err := c.initializeLogging(); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	c.logger.Info().Msg("Starting application bootstrap...")

	if err := c.initializeMongoDB(); err != nil {
		return fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	if err := c.initializeConsul(); err != nil {
		c.logger.Warn().Err(err).Msg("Consul initialization failed")
		return fmt.Errorf("failed to initialize Consul: %w", err)
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

	client, err := mongo.NewMongoService(mongoConfig)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)
	if !health.Connected {
		return fmt.Errorf("MongoDB health check failed: %s", health.Error)
	}

	c.mongoClient = client
	c.addShutdownFunc(func() error {
		return client.Close(context.Background())
	})

	c.logger.Info().
		Str("address", mongoConfig.Address).
		Str("database", mongoConfig.Database).
		Msg("MongoDB connection established")

	return nil
}

func (c *Container) initializeVault() error {
	vaultAddr, err := env.Get("VAULT_ADDRESS", "localhost:8200")
	if err != nil {
		return fmt.Errorf("failed to get VAULT_ADDRESS: %w", err)
	}

	vaultToken, err := env.Get("VAULT_TOKEN", "token")
	if err != nil {
		return fmt.Errorf("failed to get VAULT_TOKEN: %w", err)
	}

	if vaultAddr == "" || vaultToken == "" {
		fmt.Printf("Vault configuration not provided, skipping Vault initialization\n")
		return nil
	}

	vaultConfig := vault.VaultConfig{
		Address: vaultAddr,
		Token:   vaultToken,
	}

	client, err := vault.NewVaultClient(vaultConfig)
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

	if c.logger.GetLevel() != -1 {
		c.logger.Info().
			Str("address", vaultAddr).
			Bool("authenticated", health.Authenticated).
			Msg("Vault connection established")
	} else {
		fmt.Printf("Vault connection established at %s (authenticated: %v)\n", vaultAddr, health.Authenticated)
	}

	return nil
}

func (c *Container) initializeConsul() error {
	consulAddr := os.Getenv("CONSUL_ADDRESS")
	consulToken := os.Getenv("CONSUL_TOKEN")
	consulDatacenter := os.Getenv("CONSUL_DATACENTER")

	if consulAddr == "" {
		c.logger.Info().Msg("Consul configuration not provided, skipping Consul initialization")
		return nil
	}

	consulConfig := consul.ConsulConfig{
		Address:    consulAddr,
		Token:      consulToken,
		Datacenter: consulDatacenter,
	}

	client, err := consul.NewConsulClient(consulConfig)
	if err != nil {
		return fmt.Errorf("failed to create Consul client: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	health := client.HealthCheck(ctx)
	if !health.Connected {
		return fmt.Errorf("consul health check failed: %s", health.Error)
	}

	c.consulClient = client
	c.addShutdownFunc(client.Close)

	c.logger.Info().
		Str("address", consulAddr).
		Str("datacenter", consulDatacenter).
		Str("leader", health.Leader).
		Msg("Consul connection established")

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

func (c *Container) GetConsulClient() consul.ConsulService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consulClient
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
		if !health.Connected {
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

	if c.consulClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health := c.consulClient.HealthCheck(ctx)
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
			Name:      "consul",
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
