package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Address  string
	Password string
	Database int
}

type HealthStatus struct {
	Connected      bool          `json:"connected"`
	Authenticated  bool          `json:"authenticated"`
	DatabaseExists bool          `json:"database_exists"`
	Address        string        `json:"address"`
	Database       int           `json:"database"`
	Latency        time.Duration `json:"latency"`
	Error          string        `json:"error,omitempty"`
}

type RedisService interface {
	HealthCheck(ctx context.Context) HealthStatus
	GetClient() *redis.Client
	Close() error

	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GetBytes(ctx context.Context, key string) ([]byte, error)
	GetSet(ctx context.Context, key string, value interface{}) (string, error)
	Delete(ctx context.Context, keys ...string) (int64, error)
	Exists(ctx context.Context, keys ...string) (int64, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error

	HSet(ctx context.Context, key, field string, value interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDelete(ctx context.Context, key string, fields ...string) (int64, error)
	HExists(ctx context.Context, key, field string) (bool, error)

	LPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	RPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	LPop(ctx context.Context, key string) (string, error)
	RPop(ctx context.Context, key string) (string, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)

	SAdd(ctx context.Context, key string, members ...interface{}) (int64, error)
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SRemove(ctx context.Context, key string, members ...interface{}) (int64, error)

	Keys(ctx context.Context, pattern string) ([]string, error)
	FlushDB(ctx context.Context) error
	Ping(ctx context.Context) error
	Info(ctx context.Context, section ...string) (string, error)
}

type RedisClient struct {
	client *redis.Client
	config RedisConfig
	mu     sync.RWMutex
}

func NewRedisService(config RedisConfig) (*RedisClient, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:            config.Address,
		Password:        config.Password,
		DB:              config.Database,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		DialTimeout:     10 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: rdb,
		config: config,
	}, nil
}

func (r *RedisClient) HealthCheck(ctx context.Context) HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := time.Now()

	status := HealthStatus{
		Address:  r.config.Address,
		Database: r.config.Database,
	}

	pong, err := r.client.Ping(ctx).Result()
	if err != nil {
		status.Connected = false
		status.Authenticated = false
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("ping failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	if pong != "PONG" {
		status.Connected = false
		status.Authenticated = false
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("unexpected ping response: %s", pong)
		status.Latency = time.Since(start)
		return status
	}

	status.Connected = true

	_, err = r.client.Info(ctx).Result()
	if err != nil {
		status.Authenticated = false
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("authentication failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Authenticated = true

	result, err := r.client.Do(ctx, "CONFIG", "GET", "databases").Result()
	if err != nil {
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("database check failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	if resultSlice, ok := result.([]interface{}); ok && len(resultSlice) >= 2 {
		if dbCountStr, ok := resultSlice[1].(string); ok {
			if dbCount, parseErr := strconv.Atoi(dbCountStr); parseErr == nil {
				if r.config.Database >= dbCount {
					status.DatabaseExists = false
					status.Error = fmt.Sprintf("database %d does not exist (max: %d)", r.config.Database, dbCount-1)
					status.Latency = time.Since(start)
					return status
				}
			}
		}
	}

	_, err = r.client.DBSize(ctx).Result()
	if err != nil {
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("database access failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.DatabaseExists = true
	status.Latency = time.Since(start)

	return status
}

func (r *RedisClient) GetClient() *redis.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

func (r *RedisClient) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// String operations
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return result, err
}

func (r *RedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return result, err
}

func (r *RedisClient) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, err := r.client.GetSet(ctx, key, value).Result()
	if err == redis.Nil {
		return "", nil
	}
	return result, err
}

func (r *RedisClient) Delete(ctx context.Context, keys ...string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(keys) == 0 {
		return 0, nil
	}
	return r.client.Del(ctx, keys...).Result()
}

func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(keys) == 0 {
		return 0, nil
	}
	return r.client.Exists(ctx, keys...).Result()
}

func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.TTL(ctx, key).Result()
}

func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.Expire(ctx, key, expiration).Err()
}

// Hash operations
func (r *RedisClient) HSet(ctx context.Context, key, field string, value interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.HSet(ctx, key, field, value).Err()
}

func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, err := r.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("field not found: %s in key %s", field, key)
	}
	return result, err
}

func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.HGetAll(ctx, key).Result()
}

func (r *RedisClient) HDelete(ctx context.Context, key string, fields ...string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(fields) == 0 {
		return 0, nil
	}
	return r.client.HDel(ctx, key, fields...).Result()
}

func (r *RedisClient) HExists(ctx context.Context, key, field string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.HExists(ctx, key, field).Result()
}

// List operations
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(values) == 0 {
		return 0, nil
	}
	return r.client.LPush(ctx, key, values...).Result()
}

func (r *RedisClient) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(values) == 0 {
		return 0, nil
	}
	return r.client.RPush(ctx, key, values...).Result()
}

func (r *RedisClient) LPop(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, err := r.client.LPop(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("list is empty or key does not exist: %s", key)
	}
	return result, err
}

func (r *RedisClient) RPop(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, err := r.client.RPop(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("list is empty or key does not exist: %s", key)
	}
	return result, err
}

func (r *RedisClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.LRange(ctx, key, start, stop).Result()
}

// Set operations
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(members) == 0 {
		return 0, nil
	}
	return r.client.SAdd(ctx, key, members...).Result()
}

func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.SMembers(ctx, key).Result()
}

func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.SIsMember(ctx, key, member).Result()
}

func (r *RedisClient) SRemove(ctx context.Context, key string, members ...interface{}) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(members) == 0 {
		return 0, nil
	}
	return r.client.SRem(ctx, key, members...).Result()
}

// Utility operations
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.Keys(ctx, pattern).Result()
}

func (r *RedisClient) FlushDB(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.FlushDB(ctx).Err()
}

func (r *RedisClient) Ping(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) Info(ctx context.Context, section ...string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(section) == 0 {
		return r.client.Info(ctx).Result()
	}
	return r.client.Info(ctx, section...).Result()
}
