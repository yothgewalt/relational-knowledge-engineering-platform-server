package database

import (
	"context"
	"fmt"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
)

type Manager struct {
	MongoDB *MongoClient
	Redis   *RedisClient
	Neo4j   *Neo4jClient
	Qdrant  *QdrantClient
	MinIO   *MinIOClient
}

func NewManager(cfg config.Config) (*Manager, error) {
	mongoClient, err := NewMongoClient(cfg.Database.MongoDB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MongoDB client: %w", err)
	}

	redisClient, err := NewRedisClient(cfg.Database.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis client: %w", err)
	}

	neo4jClient, err := NewNeo4jClient(cfg.Database.Neo4j)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Neo4j client: %w", err)
	}

	qdrantClient, err := NewQdrantClient(cfg.Database.Qdrant)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Qdrant client: %w", err)
	}

	minioClient, err := NewMinIOClient(cfg.Database.MinIO)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return &Manager{
		MongoDB: mongoClient,
		Redis:   redisClient,
		Neo4j:   neo4jClient,
		Qdrant:  qdrantClient,
		MinIO:   minioClient,
	}, nil
}

func (m *Manager) Close(ctx context.Context) error {
	var errors []error

	if err := m.MongoDB.Close(ctx); err != nil {
		errors = append(errors, fmt.Errorf("MongoDB close error: %w", err))
	}

	if err := m.Redis.Close(); err != nil {
		errors = append(errors, fmt.Errorf("Redis close error: %w", err))
	}

	if err := m.Neo4j.Close(ctx); err != nil {
		errors = append(errors, fmt.Errorf("Neo4j close error: %w", err))
	}

	if err := m.Qdrant.Close(); err != nil {
		errors = append(errors, fmt.Errorf("Qdrant close error: %w", err))
	}

	if err := m.MinIO.Close(); err != nil {
		errors = append(errors, fmt.Errorf("MinIO close error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("database close errors: %v", errors)
	}

	return nil
}