package config

import (
	"sync"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/env"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Mongo    MongoConfig    `json:"mongo"`
	Redis    RedisConfig    `json:"redis"`
	Features FeaturesConfig `json:"features"`
}

type ServerConfig struct {
	Port        string        `json:"port"`
	Host        string        `json:"host"`
	AppName     string        `json:"app_name"`
	ReadTimeout time.Duration `json:"read_timeout"`
}

type MongoConfig struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type RedisConfig struct {
	Address  string `json:"address"`
	Database int    `json:"database"`
	Password string `json:"password"`
}

type FeaturesConfig struct {
	DebugMode      bool `json:"debug_mode"`
	APIDocsEnabled bool `json:"api_docs_enabled"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
)

func Load() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	var err error
	once.Do(func() {
		instance, err = loadConfig()
	})

	if err != nil {
		return nil, err
	}

	return instance, nil
}

func loadServerConfig() (ServerConfig, error) {
	var serverConfig ServerConfig

	port, err := env.Get("PORT", "3000")
	if err != nil {
		return serverConfig, err
	}
	serverConfig.Port = port

	host, err := env.Get("HOST", "0.0.0.0")
	if err != nil {
		return serverConfig, err
	}
	serverConfig.Host = host

	appName, err := env.Get("APP_NAME", "Relational Knowledge Engineering Platform")
	if err != nil {
		return serverConfig, err
	}
	serverConfig.AppName = appName

	readTimeout, err := env.Get("READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return serverConfig, err
	}
	serverConfig.ReadTimeout = readTimeout

	return serverConfig, nil
}

func loadMongoConfig() (MongoConfig, error) {
	var mongoConfig MongoConfig

	address, err := env.Get("MONGO_ADDRESS", "localhost:27017")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Address = address

	username, err := env.Get("MONGO_USERNAME", "admin")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Username = username

	password, err := env.Get("MONGO_PASSWORD", "password")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Password = password

	database, err := env.Get("MONGO_DATABASE", "relational_knowledge_engineering_platform")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Database = database

	return mongoConfig, nil
}

func loadRedisConfig() (RedisConfig, error) {
	var redisConfig RedisConfig

	address, err := env.Get("REDIS_ADDRESS", "localhost:6379")
	if err != nil {
		return redisConfig, err
	}
	redisConfig.Address = address

	database, err := env.Get("REDIS_DATABASE", 0)
	if err != nil {
		return redisConfig, err
	}
	redisConfig.Database = database

	password, err := env.Get("REDIS_PASSWORD", "")
	if err != nil {
		return redisConfig, err
	}
	redisConfig.Password = password

	return redisConfig, nil
}

func loadFeaturesConfig() (FeaturesConfig, error) {
	var featuresConfig FeaturesConfig

	debugMode, err := env.Get("DEBUG_MODE", false)
	if err != nil {
		return featuresConfig, err
	}
	featuresConfig.DebugMode = debugMode

	apiDocs, err := env.Get("API_DOCS_ENABLED", true)
	if err != nil {
		return featuresConfig, err
	}
	featuresConfig.APIDocsEnabled = apiDocs

	return featuresConfig, nil
}

func loadConfig() (*Config, error) {
	config := &Config{}

	serverConfig, err := loadServerConfig()
	if err != nil {
		return nil, err
	}
	config.Server = serverConfig

	mongoConfig, err := loadMongoConfig()
	if err != nil {
		return nil, err
	}
	config.Mongo = mongoConfig

	redisConfig, err := loadRedisConfig()
	if err != nil {
		return nil, err
	}
	config.Redis = redisConfig

	featuresConfig, err := loadFeaturesConfig()
	if err != nil {
		return nil, err
	}
	config.Features = featuresConfig

	return config, nil
}

func MustLoad() *Config {
	config, err := Load()
	if err != nil {
		panic(err)
	}
	return config
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("config not loaded, call Load() first")
	}

	return instance
}

func Reload() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	once = sync.Once{}
	instance = nil

	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	instance = config
	return instance, nil
}
