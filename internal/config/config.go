package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/env"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/log"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/vault"
)

type Config struct {
	Server       ServerConfig       `json:"server"`
	Mongo        MongoConfig        `json:"mongo"`
	Redis        RedisConfig        `json:"redis"`
	Neo4j        Neo4jConfig        `json:"neo4j"`
	Features     FeaturesConfig     `json:"features"`
	VaultSecrets VaultSecretsConfig `json:"vault_secrets"`
	Vault        VaultConfig        `json:"vault"`
	Resend       ResendConfig       `json:"resend"`
	JWT          JWTConfig          `json:"jwt"`
}

type VaultSecretsConfig struct {
	MongoSecretPath  string `json:"mongo_secret_path"`
	RedisSecretPath  string `json:"redis_secret_path"`
	Neo4jSecretPath  string `json:"neo4j_secret_path"`
	ResendSecretPath string `json:"resend_secret_path"`
	JwtSecretPath    string `json:"jwt_secret_path"`
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

type Neo4jConfig struct {
	URI      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type FeaturesConfig struct {
	DebugMode      bool `json:"debug_mode"`
	APIDocsEnabled bool `json:"api_docs_enabled"`
}

type VaultConfig struct {
	Address string `json:"address"`
	Token   string `json:"token"`
}

type ResendConfig struct {
	ApiKey string `json:"api_key"`
}

type JWTConfig struct {
	Secret     string        `json:"secret"`
	Expiration time.Duration `json:"expiration"`
	Issuer     string        `json:"issuer"`
}

var (
	instance    *Config
	once        sync.Once
	mu          sync.RWMutex
	vaultClient vault.VaultService
	logger      = log.New()
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

func LoadWithVault(client vault.VaultService) (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	vaultClient = client

	var err error
	once.Do(func() {
		instance, err = loadConfig()
	})

	if err != nil {
		return nil, err
	}

	return instance, nil
}

func getFromVaultOrEnv(secretPath, key, envKey, defaultValue string) (string, error) {
	if vaultClient != nil && secretPath != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		secretData, err := vaultClient.GetSecret(ctx, secretPath)
		if err == nil {
			if value, ok := secretData[key]; ok {
				if strValue, ok := value.(string); ok {
					return strValue, nil
				}
			}
		}
	}

	return env.Get(envKey, defaultValue)
}

func loadVaultSecretsConfig() (VaultSecretsConfig, error) {
	var vaultConfig VaultSecretsConfig

	mongoPath, err := env.Get("VAULT_MONGO_SECRET_PATH", "secret/database/mongodb")
	if err != nil {
		return vaultConfig, err
	}
	vaultConfig.MongoSecretPath = mongoPath

	redisPath, err := env.Get("VAULT_REDIS_SECRET_PATH", "secret/database/redis")
	if err != nil {
		return vaultConfig, err
	}
	vaultConfig.RedisSecretPath = redisPath

	neo4jPath, err := env.Get("VAULT_NEO4J_SECRET_PATH", "secret/database/neo4j")
	if err != nil {
		return vaultConfig, err
	}
	vaultConfig.Neo4jSecretPath = neo4jPath

	resendPath, err := env.Get("VAULT_RESEND_SECRET_PATH", "secret/email/resend")
	if err != nil {
		return vaultConfig, err
	}
	vaultConfig.ResendSecretPath = resendPath

	jwtPath, err := env.Get("VAULT_JWT_SECRET_PATH", "secret/jwt")
	if err != nil {
		return vaultConfig, err
	}
	vaultConfig.JwtSecretPath = jwtPath

	return vaultConfig, nil
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

func loadMongoConfig(vaultConfig VaultSecretsConfig) (MongoConfig, error) {
	var mongoConfig MongoConfig

	address, err := getFromVaultOrEnv(vaultConfig.MongoSecretPath, "address", "MONGO_ADDRESS", "localhost:27017")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Address = address

	username, err := getFromVaultOrEnv(vaultConfig.MongoSecretPath, "username", "MONGO_USERNAME", "admin")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Username = username

	password, err := getFromVaultOrEnv(vaultConfig.MongoSecretPath, "password", "MONGO_PASSWORD", "password")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Password = password

	database, err := getFromVaultOrEnv(vaultConfig.MongoSecretPath, "database", "MONGO_DATABASE", "relational_knowledge_engineering_platform")
	if err != nil {
		return mongoConfig, err
	}
	mongoConfig.Database = database

	return mongoConfig, nil
}

func loadRedisConfig(vaultConfig VaultSecretsConfig) (RedisConfig, error) {
	var redisConfig RedisConfig

	address, err := getFromVaultOrEnv(vaultConfig.RedisSecretPath, "address", "REDIS_ADDRESS", "localhost:6379")
	if err != nil {
		return redisConfig, err
	}
	redisConfig.Address = address

	var database int
	if vaultClient != nil && vaultConfig.RedisSecretPath != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		secretData, err := vaultClient.GetSecret(ctx, vaultConfig.RedisSecretPath)
		if err == nil {
			if value, ok := secretData["database"]; ok {
				if intValue, ok := value.(float64); ok {
					database = int(intValue)
				} else if strValue, ok := value.(string); ok {
					if parsed, parseErr := fmt.Sscanf(strValue, "%d", &database); parseErr != nil || parsed != 1 {
						logger.Warn().Msg("Could not parse database value from Vault, falling back to environment variable")
						database, err = env.Get("REDIS_DATABASE", 0)
						if err != nil {
							return redisConfig, err
						}
					}
				}
			} else {
				database, err = env.Get("REDIS_DATABASE", 0)
				if err != nil {
					return redisConfig, err
				}
			}
		} else {
			logger.Warn().
				Str("vault_path", vaultConfig.RedisSecretPath).
				Msg("Could not get database from Vault, falling back to environment variable")
			database, err = env.Get("REDIS_DATABASE", 0)
			if err != nil {
				return redisConfig, err
			}
		}
	} else {
		database, err = env.Get("REDIS_DATABASE", 0)
		if err != nil {
			return redisConfig, err
		}
	}
	redisConfig.Database = database

	password, err := getFromVaultOrEnv(vaultConfig.RedisSecretPath, "password", "REDIS_PASSWORD", "")
	if err != nil {
		return redisConfig, err
	}
	redisConfig.Password = password

	return redisConfig, nil
}

func loadNeo4jConfig(vaultConfig VaultSecretsConfig) (Neo4jConfig, error) {
	var neo4jConfig Neo4jConfig

	logger.Debug().
		Str("vault_path", vaultConfig.Neo4jSecretPath).
		Msg("Loading Neo4j configuration")

	uri, err := getFromVaultOrEnv(vaultConfig.Neo4jSecretPath, "uri", "NEO4J_URI", "bolt://localhost:7687")
	if err != nil {
		return neo4jConfig, err
	}
	neo4jConfig.URI = uri

	username, err := getFromVaultOrEnv(vaultConfig.Neo4jSecretPath, "username", "NEO4J_USERNAME", "neo4j")
	if err != nil {
		return neo4jConfig, err
	}
	neo4jConfig.Username = username

	password, err := getFromVaultOrEnv(vaultConfig.Neo4jSecretPath, "password", "NEO4J_PASSWORD", "")
	if err != nil {
		return neo4jConfig, err
	}
	neo4jConfig.Password = password

	database, err := getFromVaultOrEnv(vaultConfig.Neo4jSecretPath, "database", "NEO4J_DATABASE", "neo4j")
	if err != nil {
		return neo4jConfig, err
	}
	neo4jConfig.Database = database

	return neo4jConfig, nil
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

func loadVaultConfig() (VaultConfig, error) {
	var vaultConfig VaultConfig

	address, err := env.Get("VAULT_ADDRESS", "localhost:8200")
	if err != nil {
		return vaultConfig, err
	}
	vaultConfig.Address = address

	token, err := env.Get("VAULT_TOKEN", "")
	if err != nil {
		return vaultConfig, err
	}
	vaultConfig.Token = token

	return vaultConfig, nil
}

func loadResendConfig(vaultSecretsConfig VaultSecretsConfig) (ResendConfig, error) {
	var resendConfig ResendConfig

	apiKey, err := getFromVaultOrEnv(vaultSecretsConfig.ResendSecretPath, "api_key", "RESEND_API_KEY", "")
	if err != nil {
		logger.Error().
			Err(err).
			Str("vault_path", vaultSecretsConfig.ResendSecretPath).
			Msg("Failed to load Resend API key")
		return resendConfig, err
	}
	resendConfig.ApiKey = apiKey

	return resendConfig, nil
}

func loadJWTConfig(vaultSecretsConfig VaultSecretsConfig) (JWTConfig, error) {
	var jwtConfig JWTConfig

	secret, err := getFromVaultOrEnv(vaultSecretsConfig.JwtSecretPath, "secret", "JWT_SECRET", "default-jwt-secret-change-in-production")
	if err != nil {
		logger.Error().
			Err(err).
			Str("vault_path", vaultSecretsConfig.JwtSecretPath).
			Msg("Failed to load JWT secret")
		return jwtConfig, err
	}
	jwtConfig.Secret = secret

	var expiration time.Duration
	if vaultClient != nil && vaultSecretsConfig.JwtSecretPath != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		secretData, err := vaultClient.GetSecret(ctx, vaultSecretsConfig.JwtSecretPath)
		if err == nil {
			if value, ok := secretData["expiration"]; ok {
				if strValue, ok := value.(string); ok {
					if parsed, parseErr := time.ParseDuration(strValue); parseErr == nil {
						expiration = parsed
					} else {
						logger.Warn().Msg("Could not parse JWT expiration from Vault, falling back to environment variable")
						expiration, err = env.Get("JWT_EXPIRATION", 24*time.Hour)
						if err != nil {
							return jwtConfig, err
						}
					}
				}
			} else {
				expiration, err = env.Get("JWT_EXPIRATION", 24*time.Hour)
				if err != nil {
					return jwtConfig, err
				}
			}
		} else {
			logger.Warn().
				Str("vault_path", vaultSecretsConfig.JwtSecretPath).
				Msg("Could not get JWT expiration from Vault, falling back to environment variable")
			expiration, err = env.Get("JWT_EXPIRATION", 24*time.Hour)
			if err != nil {
				return jwtConfig, err
			}
		}
	} else {
		expiration, err = env.Get("JWT_EXPIRATION", 24*time.Hour)
		if err != nil {
			return jwtConfig, err
		}
	}
	jwtConfig.Expiration = expiration

	issuer, err := getFromVaultOrEnv(vaultSecretsConfig.JwtSecretPath, "issuer", "JWT_ISSUER", "relational-knowledge-engineering-platform")
	if err != nil {
		logger.Error().
			Err(err).
			Str("vault_path", vaultSecretsConfig.JwtSecretPath).
			Msg("Failed to load JWT issuer")
		return jwtConfig, err
	}
	jwtConfig.Issuer = issuer

	return jwtConfig, nil
}

func loadConfig() (*Config, error) {
	config := &Config{}

	vaultSecretsConfig, err := loadVaultSecretsConfig()
	if err != nil {
		return nil, err
	}
	config.VaultSecrets = vaultSecretsConfig

	vaultConfig, err := loadVaultConfig()
	if err != nil {
		return nil, err
	}
	config.Vault = vaultConfig

	serverConfig, err := loadServerConfig()
	if err != nil {
		return nil, err
	}
	config.Server = serverConfig

	mongoConfig, err := loadMongoConfig(vaultSecretsConfig)
	if err != nil {
		return nil, err
	}
	config.Mongo = mongoConfig

	redisConfig, err := loadRedisConfig(vaultSecretsConfig)
	if err != nil {
		return nil, err
	}
	config.Redis = redisConfig

	neo4jConfig, err := loadNeo4jConfig(vaultSecretsConfig)
	if err != nil {
		return nil, err
	}
	config.Neo4j = neo4jConfig

	featuresConfig, err := loadFeaturesConfig()
	if err != nil {
		return nil, err
	}
	config.Features = featuresConfig

	resendConfig, err := loadResendConfig(vaultSecretsConfig)
	if err != nil {
		return nil, err
	}
	config.Resend = resendConfig

	jwtConfig, err := loadJWTConfig(vaultSecretsConfig)
	if err != nil {
		return nil, err
	}
	config.JWT = jwtConfig

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

func ReloadWithVault(client vault.VaultService) (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	vaultClient = client
	once = sync.Once{}
	instance = nil

	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	instance = config
	return instance, nil
}
