package consul

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

type ConsulConfig struct {
	Address    string
	Token      string
	Datacenter string
	TLSConfig  *TLSConfig
}

type TLSConfig struct {
	CACert     string
	ClientCert string
	ClientKey  string
	Insecure   bool
}

type HealthStatus struct {
	Connected bool          `json:"connected"`
	Address   string        `json:"address"`
	Leader    string        `json:"leader,omitempty"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
}

type ServiceInstance struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Port    int               `json:"port"`
	Tags    []string          `json:"tags"`
	Meta    map[string]string `json:"meta,omitempty"`
	Health  string            `json:"health"`
}

type ConsulService interface {
	HealthCheck(ctx context.Context) HealthStatus
	GetValue(ctx context.Context, key string) (string, error)
	PutValue(ctx context.Context, key, value string) error
	DeleteValue(ctx context.Context, key string) error
	ListKeys(ctx context.Context, prefix string) ([]string, error)
	GetService(ctx context.Context, service string) ([]ServiceInstance, error)
	Close() error
}

type ConsulClient struct {
	client *api.Client
	config ConsulConfig
	mu     sync.RWMutex
}

func NewConsulClient(config ConsulConfig) (*ConsulClient, error) {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = config.Address
	consulConfig.Token = config.Token

	if config.Datacenter != "" {
		consulConfig.Datacenter = config.Datacenter
	}

	if config.TLSConfig != nil {
		consulConfig.TLSConfig = api.TLSConfig{
			CAFile:             config.TLSConfig.CACert,
			CertFile:           config.TLSConfig.ClientCert,
			KeyFile:            config.TLSConfig.ClientKey,
			InsecureSkipVerify: config.TLSConfig.Insecure,
		}
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	if _, err := client.Agent().Self(); err != nil {
		return nil, fmt.Errorf("failed to connect to Consul: %w", err)
	}

	return &ConsulClient{
		client: client,
		config: config,
	}, nil
}

func (c *ConsulClient) HealthCheck(ctx context.Context) HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	start := time.Now()
	status := HealthStatus{
		Address: c.config.Address,
	}

	agent := c.client.Agent()
	if _, err := agent.Self(); err != nil {
		status.Connected = false
		status.Error = err.Error()
		return status
	}

	status.Connected = true
	status.Latency = time.Since(start)

	if leader, err := c.client.Status().Leader(); err == nil {
		status.Leader = leader
	}

	return status
}

func (c *ConsulClient) GetValue(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	kv := c.client.KV()
	pair, _, err := kv.Get(key, &api.QueryOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if pair == nil {
		return "", fmt.Errorf("key not found: %s", key)
	}

	return string(pair.Value), nil
}

func (c *ConsulClient) PutValue(ctx context.Context, key, value string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	kv := c.client.KV()
	pair := &api.KVPair{
		Key:   key,
		Value: []byte(value),
	}

	_, err := kv.Put(pair, &api.WriteOptions{})
	if err != nil {
		return fmt.Errorf("failed to put key %s: %w", key, err)
	}

	return nil
}

func (c *ConsulClient) DeleteValue(ctx context.Context, key string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	kv := c.client.KV()
	_, err := kv.Delete(key, &api.WriteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

func (c *ConsulClient) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	kv := c.client.KV()
	keys, _, err := kv.Keys(prefix, "", &api.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list keys with prefix %s: %w", prefix, err)
	}

	return keys, nil
}

func (c *ConsulClient) GetService(ctx context.Context, service string) ([]ServiceInstance, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	health := c.client.Health()
	entries, _, err := health.Service(service, "", true, &api.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s: %w", service, err)
	}

	instances := make([]ServiceInstance, len(entries))
	for i, entry := range entries {
		healthStatus := "critical"

		allPassing := true
		hasWarning := false

		for _, check := range entry.Checks {
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

		instances[i] = ServiceInstance{
			ID:      entry.Service.ID,
			Name:    entry.Service.Service,
			Address: entry.Service.Address,
			Port:    entry.Service.Port,
			Tags:    entry.Service.Tags,
			Meta:    entry.Service.Meta,
			Health:  healthStatus,
		}
	}

	return instances, nil
}

func (c *ConsulClient) RegisterService(ctx context.Context, service *api.AgentServiceRegistration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	agent := c.client.Agent()
	if err := agent.ServiceRegister(service); err != nil {
		return fmt.Errorf("failed to register service %s: %w", service.Name, err)
	}

	return nil
}

func (c *ConsulClient) DeregisterService(ctx context.Context, serviceID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	agent := c.client.Agent()
	if err := agent.ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("failed to deregister service %s: %w", serviceID, err)
	}

	return nil
}

func (c *ConsulClient) GetHealthyServices(ctx context.Context, service string) ([]ServiceInstance, error) {
	instances, err := c.GetService(ctx, service)
	if err != nil {
		return nil, err
	}

	var healthy []ServiceInstance
	for _, instance := range instances {
		if instance.Health == "passing" {
			healthy = append(healthy, instance)
		}
	}

	return healthy, nil
}

func (c *ConsulClient) WatchKey(ctx context.Context, key string, callback func(string, error)) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastValue string

	if value, err := c.GetValue(ctx, key); err == nil {
		lastValue = value
		callback(value, nil)
	} else {
		callback("", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if value, err := c.GetValue(ctx, key); err != nil {
				callback("", err)
			} else if value != lastValue {
				lastValue = value
				callback(value, nil)
			}
		}
	}
}

func (c *ConsulClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.client = nil

	return nil
}
