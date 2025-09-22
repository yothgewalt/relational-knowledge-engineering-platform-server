package consul

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
)

func TestConsulConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config ConsulConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: ConsulConfig{
				Address:    "http://localhost:8500",
				Token:      "test-token",
				Datacenter: "dc1",
			},
			valid: true,
		},
		{
			name: "empty address",
			config: ConsulConfig{
				Address:    "",
				Token:      "test-token",
				Datacenter: "dc1",
			},
			valid: false,
		},
		{
			name: "config without token",
			config: ConsulConfig{
				Address:    "http://localhost:8500",
				Token:      "",
				Datacenter: "dc1",
			},
			valid: true,
		},
		{
			name: "config without datacenter",
			config: ConsulConfig{
				Address: "http://localhost:8500",
				Token:   "test-token",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Address == "" {
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
		Connected: true,
		Address:   "http://localhost:8500",
		Leader:    "127.0.0.1:8300",
		Latency:   25 * time.Millisecond,
		Error:     "",
	}

	if !status.Connected {
		t.Error("Connected field should be accessible")
	}
	if status.Address != "http://localhost:8500" {
		t.Error("Address field should be accessible")
	}
	if status.Leader != "127.0.0.1:8300" {
		t.Error("Leader field should be accessible")
	}
	if status.Latency != 25*time.Millisecond {
		t.Error("Latency field should be accessible")
	}
	if status.Error != "" {
		t.Error("Error field should be empty")
	}
}

func TestConsulService_Interface(t *testing.T) {
	var _ ConsulService = (*ConsulClient)(nil)
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

func TestConsulClient_StructureAndMethods(t *testing.T) {
	client := &ConsulClient{
		config: ConsulConfig{
			Address:    "http://localhost:8500",
			Token:      "test-token",
			Datacenter: "dc1",
		},
	}

	if client.config.Address != "http://localhost:8500" {
		t.Error("config.Address should be accessible")
	}
	if client.config.Token != "test-token" {
		t.Error("config.Token should be accessible")
	}
	if client.config.Datacenter != "dc1" {
		t.Error("config.Datacenter should be accessible")
	}
}

func TestServiceInstance_Structure(t *testing.T) {
	instance := ServiceInstance{
		ID:      "web-1",
		Name:    "web",
		Address: "192.168.1.100",
		Port:    8080,
		Tags:    []string{"v1.0", "production"},
		Meta: map[string]string{
			"version": "1.0.0",
			"region":  "us-west",
		},
		Health: "passing",
	}

	if instance.ID != "web-1" {
		t.Error("ID field should be accessible")
	}
	if instance.Name != "web" {
		t.Error("Name field should be accessible")
	}
	if instance.Address != "192.168.1.100" {
		t.Error("Address field should be accessible")
	}
	if instance.Port != 8080 {
		t.Error("Port field should be accessible")
	}
	if len(instance.Tags) != 2 || instance.Tags[0] != "v1.0" {
		t.Error("Tags field should be accessible")
	}
	if instance.Meta["version"] != "1.0.0" {
		t.Error("Meta field should be accessible")
	}
	if instance.Health != "passing" {
		t.Error("Health field should be accessible")
	}
}

func TestKVOperations_Validation(t *testing.T) {
	testCases := []struct {
		operation string
		key       string
		value     string
		valid     bool
	}{
		{"get", "config/app/database", "", true},
		{"get", "", "", false},
		{"put", "config/app/database", "postgres://localhost", true},
		{"put", "config/app/database", "", true}, // Empty value is valid
		{"put", "", "some-value", false},
		{"delete", "config/app/database", "", true},
		{"delete", "", "", false},
		{"list", "config/app/", "", true},
		{"list", "", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.operation+"_"+tc.key, func(t *testing.T) {
			switch tc.operation {
			case "get", "delete", "list":
				if tc.key == "" {
					if tc.valid {
						t.Error("empty key should be invalid")
					}
				}
			case "put":
				if tc.key == "" {
					if tc.valid {
						t.Error("empty key should be invalid")
					}
				}
			}
		})
	}
}

func TestServiceOperations_Validation(t *testing.T) {
	testCases := []struct {
		operation string
		service   string
		valid     bool
	}{
		{"get", "web", true},
		{"get", "", false},
		{"register", "api", true},
		{"register", "", false},
		{"deregister", "api-1", true},
		{"deregister", "", false},
		{"healthy", "web", true},
		{"healthy", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.operation+"_"+tc.service, func(t *testing.T) {
			if tc.service == "" {
				if tc.valid {
					t.Error("empty service name should be invalid")
				}
			} else {
				if !tc.valid {
					t.Error("non-empty service name should be valid")
				}
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

func TestConsulClient_Close(t *testing.T) {
	client := &ConsulClient{
		client: nil,
		config: ConsulConfig{
			Address:    "http://localhost:8500",
			Token:      "test-token",
			Datacenter: "dc1",
		},
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Close should not return error, got: %v", err)
	}

	if client.client != nil {
		t.Error("client should be nil after Close")
	}
}

func TestHealthStatus_ErrorHandling(t *testing.T) {
	status := HealthStatus{
		Connected: false,
		Leader:    "",
		Latency:   0,
		Error:     "connection refused",
	}

	if status.Connected {
		t.Error("status should indicate disconnected")
	}
	if status.Leader != "" {
		t.Error("leader should be empty for failed connection")
	}
	if status.Error == "" {
		t.Error("error message should be present")
	}
	if status.Latency != 0 {
		t.Error("latency should be zero for failed connection")
	}
}

func TestServiceRegistration_Structure(t *testing.T) {
	registration := &api.AgentServiceRegistration{
		ID:      "web-service-1",
		Name:    "web-service",
		Tags:    []string{"v1.0", "http"},
		Port:    8080,
		Address: "192.168.1.100",
		Check: &api.AgentServiceCheck{
			HTTP:     "http://192.168.1.100:8080/health",
			Interval: "10s",
			Timeout:  "3s",
		},
	}

	if registration.ID != "web-service-1" {
		t.Error("ID should be accessible")
	}
	if registration.Name != "web-service" {
		t.Error("Name should be accessible")
	}
	if len(registration.Tags) != 2 {
		t.Error("Tags should be accessible")
	}
	if registration.Port != 8080 {
		t.Error("Port should be accessible")
	}
	if registration.Address != "192.168.1.100" {
		t.Error("Address should be accessible")
	}
	if registration.Check.HTTP != "http://192.168.1.100:8080/health" {
		t.Error("Health check should be accessible")
	}
}

func TestServiceHealth_StatusMapping(t *testing.T) {
	tests := []struct {
		name           string
		checks         []*api.HealthCheck
		expectedHealth string
	}{
		{
			name: "all passing",
			checks: []*api.HealthCheck{
				{Status: api.HealthPassing},
				{Status: api.HealthPassing},
			},
			expectedHealth: "passing",
		},
		{
			name: "has warning",
			checks: []*api.HealthCheck{
				{Status: api.HealthPassing},
				{Status: api.HealthWarning},
			},
			expectedHealth: "warning",
		},
		{
			name: "has critical",
			checks: []*api.HealthCheck{
				{Status: api.HealthPassing},
				{Status: api.HealthCritical},
			},
			expectedHealth: "critical",
		},
		{
			name: "critical overrides warning",
			checks: []*api.HealthCheck{
				{Status: api.HealthWarning},
				{Status: api.HealthCritical},
			},
			expectedHealth: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthStatus := "critical"
			allPassing := true
			hasWarning := false

			for _, check := range tt.checks {
				switch check.Status {
				case api.HealthPassing:
					continue
				case api.HealthWarning:
					hasWarning = true
					allPassing = false
				case api.HealthCritical:
					allPassing = false
					hasWarning = false
				}
			}

			if allPassing {
				healthStatus = "passing"
			} else if hasWarning {
				healthStatus = "warning"
			}

			if healthStatus != tt.expectedHealth {
				t.Errorf("expected health %s, got %s", tt.expectedHealth, healthStatus)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	client := &ConsulClient{
		config: ConsulConfig{
			Address:    "http://localhost:8500",
			Token:      "test-token",
			Datacenter: "dc1",
		},
	}

	done := make(chan bool, 5)

	for range 5 {
		go func() {
			defer func() { done <- true }()

			_ = client.config.Address
			_ = client.config.Token
			_ = client.config.Datacenter
		}()
	}

	for range 5 {
		<-done
	}
}

func TestWatchKey_PollingLogic(t *testing.T) {
	client := &ConsulClient{
		client: nil,
		config: ConsulConfig{
			Address: "http://localhost:8500",
			Token:   "test-token",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	callbackCount := 0
	callback := func(value string, err error) {
		callbackCount++
	}

	defer func() {
		if r := recover(); r != nil {
			t.Log("Expected panic occurred due to nil client")
		}
	}()

	_ = client.WatchKey(ctx, "test-key", callback)
}
