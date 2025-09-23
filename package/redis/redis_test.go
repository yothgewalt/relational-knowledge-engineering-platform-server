package redis

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewRedisService_ValidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config RedisConfig
	}{
		{
			name: "basic config",
			config: RedisConfig{
				Address:  "localhost:6379",
				Password: "",
				Database: 0,
			},
		},
		{
			name: "config with password",
			config: RedisConfig{
				Address:  "localhost:6379",
				Password: "password123",
				Database: 1,
			},
		},
		{
			name: "config with different database",
			config: RedisConfig{
				Address:  "redis.example.com:6379",
				Password: "secret",
				Database: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRedisService(tt.config)

			if tt.config.Address == "" {
				if err == nil {
					t.Error("Expected error for empty address, got nil")
				}
			} else {
				if client != nil {
					client.Close()
				}
			}
		})
	}
}

func TestNewRedisService_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config RedisConfig
	}{
		{
			name: "empty address",
			config: RedisConfig{
				Address:  "",
				Password: "password",
				Database: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRedisService(tt.config)

			if err == nil {
				t.Error("Expected error for invalid config, got nil")
			}

			if client != nil {
				t.Error("Expected nil client for invalid config")
				client.Close()
			}
		})
	}
}

func TestRedisClient_ImplementsInterface(t *testing.T) {
	var _ RedisService = (*RedisClient)(nil)

	clientType := reflect.TypeOf(&RedisClient{})
	interfaceType := reflect.TypeOf((*RedisService)(nil)).Elem()

	if !clientType.Implements(interfaceType) {
		t.Error("RedisClient does not implement RedisService interface")
	}
}

func TestRedisConfig_Structure(t *testing.T) {
	config := RedisConfig{
		Address:  "localhost:6379",
		Password: "test_password",
		Database: 2,
	}

	if config.Address != "localhost:6379" {
		t.Errorf("Expected Address to be 'localhost:6379', got %s", config.Address)
	}

	if config.Password != "test_password" {
		t.Errorf("Expected Password to be 'test_password', got %s", config.Password)
	}

	if config.Database != 2 {
		t.Errorf("Expected Database to be 2, got %d", config.Database)
	}
}

func TestHealthStatus_Structure(t *testing.T) {
	status := HealthStatus{
		Connected:      true,
		Authenticated:  true,
		DatabaseExists: true,
		Address:        "localhost:6379",
		Database:       0,
		Latency:        100 * time.Millisecond,
		Error:          "",
	}

	if !status.Connected {
		t.Error("Expected Connected to be true")
	}

	if !status.Authenticated {
		t.Error("Expected Authenticated to be true")
	}

	if !status.DatabaseExists {
		t.Error("Expected DatabaseExists to be true")
	}

	if status.Address != "localhost:6379" {
		t.Errorf("Expected Address to be 'localhost:6379', got %s", status.Address)
	}

	if status.Database != 0 {
		t.Errorf("Expected Database to be 0, got %d", status.Database)
	}

	if status.Latency != 100*time.Millisecond {
		t.Errorf("Expected Latency to be 100ms, got %v", status.Latency)
	}
}

func TestHealthStatus_JSONTags(t *testing.T) {
	statusType := reflect.TypeOf(HealthStatus{})

	expectedTags := map[string]string{
		"Connected":      "connected",
		"Authenticated":  "authenticated",
		"DatabaseExists": "database_exists",
		"Address":        "address",
		"Database":       "database",
		"Latency":        "latency",
		"Error":          "error,omitempty",
	}

	for fieldName, expectedTag := range expectedTags {
		field, found := statusType.FieldByName(fieldName)
		if !found {
			t.Errorf("Field %s not found in HealthStatus", fieldName)
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag != expectedTag {
			t.Errorf("Field %s: expected JSON tag '%s', got '%s'", fieldName, expectedTag, jsonTag)
		}
	}
}

func TestRedisClient_MethodSignatures(t *testing.T) {
	clientType := reflect.TypeOf(&RedisClient{})

	expectedMethods := []struct {
		name     string
		numIn    int
		numOut   int
		hasError bool
	}{
		{"HealthCheck", 2, 1, false}, // (receiver, context) -> HealthStatus
		{"GetClient", 1, 1, false},   // (receiver) -> *redis.Client
		{"Close", 1, 1, true},        // (receiver) -> error
		{"Set", 5, 1, true},          // (receiver, context, key, value, expiration) -> error
		{"Get", 3, 2, true},          // (receiver, context, key) -> (string, error)
		{"Delete", 3, 2, true},       // (receiver, context, ...keys) -> (int64, error)
		{"HSet", 5, 1, true},         // (receiver, context, key, field, value) -> error
		{"HGet", 4, 2, true},         // (receiver, context, key, field) -> (string, error)
		{"LPush", 4, 2, true},        // (receiver, context, key, ...values) -> (int64, error)
		{"SAdd", 4, 2, true},         // (receiver, context, key, ...members) -> (int64, error)
		{"Ping", 2, 1, true},         // (receiver, context) -> error
		{"Keys", 3, 2, true},         // (receiver, context, pattern) -> ([]string, error)
	}

	for _, expected := range expectedMethods {
		method, found := clientType.MethodByName(expected.name)
		if !found {
			t.Errorf("Method %s not found", expected.name)
			continue
		}

		methodType := method.Type

		if methodType.NumIn() != expected.numIn {
			t.Errorf("Method %s: expected %d input parameters, got %d",
				expected.name, expected.numIn, methodType.NumIn())
		}

		if methodType.NumOut() != expected.numOut {
			t.Errorf("Method %s: expected %d output parameters, got %d",
				expected.name, expected.numOut, methodType.NumOut())
		}

		if expected.hasError && methodType.NumOut() > 0 {
			lastOut := methodType.Out(methodType.NumOut() - 1)
			errorInterface := reflect.TypeOf((*error)(nil)).Elem()
			if !lastOut.Implements(errorInterface) {
				t.Errorf("Method %s: expected last return type to be error, got %v",
					expected.name, lastOut)
			}
		}
	}
}

func TestRedisClient_GetClient(t *testing.T) {
	redisClient := &RedisClient{
		client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}
	defer redisClient.client.Close()

	client := redisClient.GetClient()
	if client == nil {
		t.Error("GetClient() returned nil")
	}

	if reflect.TypeOf(client) != reflect.TypeOf(&redis.Client{}) {
		t.Errorf("GetClient() returned wrong type: %T", client)
	}
}

func TestRedisClient_Close_WithNilClient(t *testing.T) {
	redisClient := &RedisClient{
		client: nil,
	}

	err := redisClient.Close()
	if err != nil {
		t.Errorf("Close() with nil client should not return error, got: %v", err)
	}
}

func TestRedisService_InterfaceCompleteness(t *testing.T) {
	interfaceType := reflect.TypeOf((*RedisService)(nil)).Elem()

	expectedMethods := []string{
		"HealthCheck", "GetClient", "Close",
		"Set", "Get", "GetBytes", "GetSet", "Delete", "Exists", "TTL", "Expire",
		"HSet", "HGet", "HGetAll", "HDelete", "HExists",
		"LPush", "RPush", "LPop", "RPop", "LRange",
		"SAdd", "SMembers", "SIsMember", "SRemove",
		"Keys", "FlushDB", "Ping", "Info",
	}

	for _, methodName := range expectedMethods {
		method, found := interfaceType.MethodByName(methodName)
		if !found {
			t.Errorf("Expected method %s not found in RedisService interface", methodName)
			continue
		}

		if method.Type.NumIn() >= 2 {
			firstParam := method.Type.In(1)
			contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
			if firstParam.Implements(contextType) || firstParam == contextType {
			} else if methodName != "GetClient" && methodName != "Close" {
				t.Errorf("Method %s should take context.Context as first parameter, got %v",
					methodName, firstParam)
			}
		}
	}
}

func TestRedisConfig_DefaultValues(t *testing.T) {
	var config RedisConfig

	if config.Address != "" {
		t.Errorf("Expected default Address to be empty, got %s", config.Address)
	}

	if config.Password != "" {
		t.Errorf("Expected default Password to be empty, got %s", config.Password)
	}

	if config.Database != 0 {
		t.Errorf("Expected default Database to be 0, got %d", config.Database)
	}
}

func TestHealthStatus_DefaultValues(t *testing.T) {
	var status HealthStatus

	if status.Connected {
		t.Error("Expected default Connected to be false")
	}

	if status.Authenticated {
		t.Error("Expected default Authenticated to be false")
	}

	if status.DatabaseExists {
		t.Error("Expected default DatabaseExists to be false")
	}

	if status.Address != "" {
		t.Errorf("Expected default Address to be empty, got %s", status.Address)
	}

	if status.Database != 0 {
		t.Errorf("Expected default Database to be 0, got %d", status.Database)
	}

	if status.Latency != 0 {
		t.Errorf("Expected default Latency to be 0, got %v", status.Latency)
	}

	if status.Error != "" {
		t.Errorf("Expected default Error to be empty, got %s", status.Error)
	}
}
