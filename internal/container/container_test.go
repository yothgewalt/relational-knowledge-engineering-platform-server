package container

import (
	"fmt"
	"testing"
	"time"
)

func TestNewContainer(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	if container == nil {
		t.Fatal("NewContainer should return a non-nil container")
	}

	if container.ctx == nil {
		t.Error("Container context should be initialized")
	}

	if container.cancel == nil {
		t.Error("Container cancel function should be initialized")
	}

	if len(container.shutdownFuncs) != 0 {
		t.Error("Container should start with empty shutdown functions")
	}

	if container.running {
		t.Error("Container should not be running initially")
	}
}

func TestNewContainerWithOptions(t *testing.T) {
	t.Parallel()
	opts := &ContainerOptions{
		DisableVault:  true,
		DisableConsul: true,
		Timezone:      "America/New_York",
	}

	container := NewContainer(opts)

	if container == nil {
		t.Fatal("NewContainer should return a non-nil container")
	}

	if container.ctx == nil {
		t.Error("Container context should be initialized")
	}
}

func TestInitializeTimezone(t *testing.T) {
	t.Run("default timezone", func(t *testing.T) {
		container := NewContainer(nil)

		t.Setenv("TZ", "")

		err := container.initializeTimezone()
		if err != nil {
			t.Errorf("initializeTimezone should not return error for default timezone: %v", err)
		}

		if time.Local.String() != "UTC" {
			t.Errorf("Expected timezone to be UTC, got %s", time.Local.String())
		}
	})

	t.Run("custom valid timezone", func(t *testing.T) {
		container := NewContainer(nil)

		t.Setenv("TZ", "America/New_York")

		err := container.initializeTimezone()
		if err != nil {
			t.Errorf("initializeTimezone should not return error for valid timezone: %v", err)
		}

		if time.Local.String() != "America/New_York" {
			t.Errorf("Expected timezone to be America/New_York, got %s", time.Local.String())
		}
	})
}

func TestInitializeTimezoneInvalid(t *testing.T) {
	container := NewContainer(nil)

	t.Setenv("TZ", "Invalid/Timezone")

	err := container.initializeTimezone()
	if err == nil {
		t.Error("initializeTimezone should return error for invalid timezone")
	}
}

func TestInitializeLogging(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	err := container.initializeLogging()
	if err != nil {
		t.Errorf("initializeLogging should not return error: %v", err)
	}

	logger := container.GetLogger()

	logger.Info().Msg("Test log message")
}

func TestContainerLifecycle(t *testing.T) {
	container := NewContainer(nil)

	if container.IsRunning() {
		t.Error("Container should not be running initially")
	}

	err := container.Bootstrap()
	if err == nil {
		t.Error("Bootstrap should fail without proper configuration")
	}

	if container.IsRunning() {
		t.Error("Container should not be running after failed bootstrap")
	}
}

func TestAddShutdownFunc(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	called := false
	shutdownFunc := func() error {
		called = true
		return nil
	}

	container.addShutdownFunc(shutdownFunc)

	if len(container.shutdownFuncs) != 1 {
		t.Error("Shutdown function should be added")
	}

	err := container.shutdownFuncs[0]()
	if err != nil {
		t.Errorf("Shutdown function should not return error: %v", err)
	}

	if !called {
		t.Error("Shutdown function should be called")
	}
}

func TestHealthCheckEmpty(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	health := container.HealthCheck()

	if health.Status != "healthy" {
		t.Errorf("Expected status to be healthy, got %s", health.Status)
	}

	if len(health.Services) != 0 {
		t.Errorf("Expected no services, got %d", len(health.Services))
	}

	if health.Uptime <= 0 {
		t.Error("Uptime should be positive")
	}
}

func TestServiceHealth(t *testing.T) {
	t.Parallel()
	health := ServiceHealth{
		Name:      "test-service",
		Status:    "healthy",
		Message:   "All good",
		Timestamp: time.Now(),
	}

	if health.Name != "test-service" {
		t.Error("Name field should be accessible")
	}

	if health.Status != "healthy" {
		t.Error("Status field should be accessible")
	}

	if health.Message != "All good" {
		t.Error("Message field should be accessible")
	}

	if health.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestHealthStatusStructure(t *testing.T) {
	t.Parallel()
	services := []ServiceHealth{
		{
			Name:      "mongodb",
			Status:    "healthy",
			Timestamp: time.Now(),
		},
		{
			Name:      "vault",
			Status:    "unhealthy",
			Message:   "Connection failed",
			Timestamp: time.Now(),
		},
	}

	health := HealthStatus{
		Status:   "degraded",
		Services: services,
		Uptime:   time.Hour,
	}

	if health.Status != "degraded" {
		t.Error("Status field should be accessible")
	}

	if len(health.Services) != 2 {
		t.Error("Services field should be accessible")
	}

	if health.Uptime != time.Hour {
		t.Error("Uptime field should be accessible")
	}

	if health.Services[0].Name != "mongodb" {
		t.Error("First service name should be mongodb")
	}

	if health.Services[1].Status != "unhealthy" {
		t.Error("Second service status should be unhealthy")
	}
}

func TestContainerOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts ContainerOptions
		want ContainerOptions
	}{
		{
			name: "all options enabled",
			opts: ContainerOptions{
				DisableVault:  true,
				DisableConsul: true,
				Timezone:      "Europe/London",
			},
			want: ContainerOptions{
				DisableVault:  true,
				DisableConsul: true,
				Timezone:      "Europe/London",
			},
		},
		{
			name: "default options",
			opts: ContainerOptions{},
			want: ContainerOptions{
				DisableVault:  false,
				DisableConsul: false,
				Timezone:      "",
			},
		},
		{
			name: "mixed options",
			opts: ContainerOptions{
				DisableVault:  false,
				DisableConsul: true,
				Timezone:      "America/New_York",
			},
			want: ContainerOptions{
				DisableVault:  false,
				DisableConsul: true,
				Timezone:      "America/New_York",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.opts.DisableVault != tt.want.DisableVault {
				t.Errorf("DisableVault = %v, want %v", tt.opts.DisableVault, tt.want.DisableVault)
			}

			if tt.opts.DisableConsul != tt.want.DisableConsul {
				t.Errorf("DisableConsul = %v, want %v", tt.opts.DisableConsul, tt.want.DisableConsul)
			}

			if tt.opts.Timezone != tt.want.Timezone {
				t.Errorf("Timezone = %v, want %v", tt.opts.Timezone, tt.want.Timezone)
			}
		})
	}
}

func TestGettersWithoutBootstrap(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	config := container.GetConfig()
	if config != nil {
		t.Error("Config should be nil before bootstrap")
	}

	mongoClient := container.GetMongoClient()
	if mongoClient != nil {
		t.Error("MongoDB client should be nil before bootstrap")
	}

	vaultClient := container.GetVaultClient()
	if vaultClient != nil {
		t.Error("Vault client should be nil before bootstrap")
	}

	consulClient := container.GetConsulClient()
	if consulClient != nil {
		t.Error("Consul client should be nil before bootstrap")
	}
}

func TestShutdownWithoutRunning(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	err := container.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should not error on non-running container: %v", err)
	}
}

func TestShutdownWithErrors(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	container.addShutdownFunc(func() error {
		return fmt.Errorf("shutdown error")
	})

	container.running = true

	err := container.Shutdown()
	if err == nil {
		t.Error("Shutdown should return error when shutdown functions fail")
	}

	if container.IsRunning() {
		t.Error("Container should not be running after shutdown")
	}
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	select {
	case <-container.ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
	}

	container.running = true
	err := container.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should not error: %v", err)
	}

	select {
	case <-container.ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after shutdown")
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	container := NewContainer(nil)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for range numGoroutines {
		go func() {
			defer func() { done <- true }()
			_ = container.IsRunning()
			_ = container.GetConfig()
			_ = container.GetLogger()
			_ = container.HealthCheck()

			for range 5 {
				_ = container.GetMongoClient()
				_ = container.GetVaultClient()
				_ = container.GetConsulClient()
			}
		}()
	}

	timeout := time.After(5 * time.Second)
	for range numGoroutines {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent access test to complete")
		}
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	container := NewContainer(nil)

	t.Setenv("VAULT_ADDRESS", "")
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("CONSUL_ADDRESS", "")

	err := container.initializeLogging()
	if err != nil {
		t.Errorf("Failed to initialize logging: %v", err)
	}

	err = container.initializeVault()
	if err != nil {
		t.Errorf("initializeVault should not error with missing env vars: %v", err)
	}

	err = container.initializeConsul()
	if err != nil {
		t.Errorf("initializeConsul should not error with missing env vars: %v", err)
	}
}
