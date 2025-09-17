package database

import (
	"context"
	"fmt"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
	config   config.MongoDBConfig
}

func NewMongoClient(cfg config.MongoDBConfig) (*MongoClient, error) {
	connectionString := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	clientOptions := options.Client().ApplyURI(connectionString)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)

	return &MongoClient{
		Client:   client,
		Database: database,
		config:   cfg,
	}, nil
}

func (m *MongoClient) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

func (m *MongoClient) GetDocumentsCollection() *mongo.Collection {
	return m.Database.Collection("documents")
}

func (m *MongoClient) GetGraphsCollection() *mongo.Collection {
	return m.Database.Collection("graphs")
}

func (m *MongoClient) GetProcessingLogsCollection() *mongo.Collection {
	return m.Database.Collection("processing_logs")
}