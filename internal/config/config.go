package config

import (
	"os"
	"strconv"
	"time"
)

func init() {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	time.Local = loc
}

type DatabaseConfig struct {
	MongoDB  MongoDBConfig  `json:"mongodb"`
	Redis    RedisConfig    `json:"redis"`
	Neo4j    Neo4jConfig    `json:"neo4j"`
	Qdrant   QdrantConfig   `json:"qdrant"`
	MinIO    MinIOConfig    `json:"minio"`
}

type MongoDBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type Neo4jConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type QdrantConfig struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	APIKey string `json:"api_key"`
}

type MinIOConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	UseSSL          bool   `json:"use_ssl"`
	BucketName      string `json:"bucket_name"`
	Region          string `json:"region"`
}

type ServerConfig struct {
	Port               int `json:"port"`
	Host               string `json:"host"`
	UploadMaxSizeMB    int `json:"upload_max_size_mb"`
	ChunkSizeMB        int `json:"chunk_size_mb"`
	MaxConcurrentUploads int `json:"max_concurrent_uploads"`
}

type Config struct {
	Database DatabaseConfig `json:"database"`
	Server   ServerConfig   `json:"server"`
}

func New() *Config {
	c := &Config{
		Database: DatabaseConfig{
			MongoDB: MongoDBConfig{
				Host:     getEnv("MONGODB_HOST", "localhost"),
				Port:     getEnvAsInt("MONGODB_PORT", 27017),
				Database: getEnv("MONGODB_DATABASE", "relational_knowledge_db"),
				Username: getEnv("MONGODB_USERNAME", "app_user"),
				Password: getEnv("MONGODB_PASSWORD", "app_password123"),
			},
			Redis: RedisConfig{
				Host:     getEnv("REDIS_HOST", "localhost"),
				Port:     getEnvAsInt("REDIS_PORT", 6379),
				Password: getEnv("REDIS_PASSWORD", ""),
				DB:       getEnvAsInt("REDIS_DB", 0),
			},
			Neo4j: Neo4jConfig{
				Host:     getEnv("NEO4J_HOST", "localhost"),
				Port:     getEnvAsInt("NEO4J_PORT", 7687),
				Username: getEnv("NEO4J_USERNAME", "neo4j"),
				Password: getEnv("NEO4J_PASSWORD", "password123"),
				Database: getEnv("NEO4J_DATABASE", "neo4j"),
			},
			Qdrant: QdrantConfig{
				Host:   getEnv("QDRANT_HOST", "localhost"),
				Port:   getEnvAsInt("QDRANT_PORT", 6333),
				APIKey: getEnv("QDRANT_API_KEY", ""),
			},
			MinIO: MinIOConfig{
				Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
				AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
				SecretAccessKey: getEnv("MINIO_SECRET_KEY", "minioadmin123"),
				UseSSL:          getEnvAsBool("MINIO_USE_SSL", false),
				BucketName:      getEnv("MINIO_BUCKET_NAME", "pdf-documents"),
				Region:          getEnv("MINIO_REGION", "us-east-1"),
			},
		},
		Server: ServerConfig{
			Port:               getEnvAsInt("SERVER_PORT", 3000),
			Host:               getEnv("SERVER_HOST", "localhost"),
			UploadMaxSizeMB:    getEnvAsInt("UPLOAD_MAX_SIZE_MB", 100),
			ChunkSizeMB:        getEnvAsInt("CHUNK_SIZE_MB", 10),
			MaxConcurrentUploads: getEnvAsInt("MAX_CONCURRENT_UPLOADS", 5),
		},
	}
	return c
}

func Load() *Config {
	c := New()
	return c
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
