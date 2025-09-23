package mongo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoConfig struct {
	Address  string
	Username string
	Password string
	Database string
}

type HealthStatus struct {
	Connected      bool          `json:"connected"`
	Database       string        `json:"database"`
	Authenticated  bool          `json:"authenticated"`
	DatabaseExists bool          `json:"database_exists"`
	Latency        time.Duration `json:"latency"`
	Error          string        `json:"error,omitempty"`
}

type MongoService struct {
	client   *mongo.Client
	database *mongo.Database
	config   MongoConfig
	mu       sync.RWMutex
}

type PaginatedResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int64 `json:"page"`
	Limit      int64 `json:"limit"`
	TotalPages int64 `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

type PaginationOptions struct {
	Page  int64 `json:"page"`
	Limit int64 `json:"limit"`
}

type Repository[T any] interface {
	Find(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]T, error)
	FindWithPagination(ctx context.Context, filter bson.M, pagination PaginationOptions, opts ...*options.FindOptions) (*PaginatedResult[T], error)
	FindOne(ctx context.Context, filter bson.M, opts ...*options.FindOneOptions) (*T, error)
	Create(ctx context.Context, document T) (*T, error)
	Update(ctx context.Context, filter bson.M, update bson.M, opts ...*options.UpdateOptions) (*T, error)
	Delete(ctx context.Context, filter bson.M, opts ...*options.DeleteOptions) error
	Count(ctx context.Context, filter bson.M, opts ...*options.CountOptions) (int64, error)
}

type GenericRepository[T any] struct {
	collection *mongo.Collection
	service    *MongoService
}

func NewMongoService(config MongoConfig) (*MongoService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := fmt.Sprintf("mongodb://%s:%s@%s/?authSource=admin",
		config.Username, config.Password, config.Address)

	clientOptions := options.Client().ApplyURI(uri)
	clientOptions.SetMaxPoolSize(100)
	clientOptions.SetMinPoolSize(5)
	clientOptions.SetMaxConnIdleTime(30 * time.Second)
	clientOptions.SetConnectTimeout(10 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(config.Database)

	service := &MongoService{
		client:   client,
		database: database,
		config:   config,
	}

	return service, nil
}

func (s *MongoService) HealthCheck(ctx context.Context) HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()

	status := HealthStatus{
		Database: s.config.Database,
	}

	// Test basic connectivity and authentication
	if err := s.client.Ping(ctx, readpref.Primary()); err != nil {
		status.Connected = false
		status.Authenticated = false
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("ping failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Connected = true

	// Test authentication by running a simple admin command
	var result bson.M
	err := s.client.Database("admin").RunCommand(ctx, bson.D{{"ismaster", 1}}).Decode(&result)
	if err != nil {
		status.Authenticated = false
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("authentication failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Authenticated = true

	// Test database access by listing collections
	_, err = s.database.ListCollectionNames(ctx, bson.M{})
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

func (s *MongoService) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s.client.Disconnect(ctx)
	}
	return nil
}

func (s *MongoService) GetCollection(name string) *mongo.Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.database.Collection(name)
}

func NewRepository[T any](service *MongoService, collectionName string) Repository[T] {
	collection := service.GetCollection(collectionName)
	return &GenericRepository[T]{
		collection: collection,
		service:    service,
	}
}

func (r *GenericRepository[T]) Find(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]T, error) {
	cursor, err := r.collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute find query: %w", err)
	}
	defer cursor.Close(ctx)

	var results []T
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode find results: %w", err)
	}

	return results, nil
}

func (r *GenericRepository[T]) FindOne(ctx context.Context, filter bson.M, opts ...*options.FindOneOptions) (*T, error) {
	var result T
	err := r.collection.FindOne(ctx, filter, opts...).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to execute findOne query: %w", err)
	}

	return &result, nil
}

func (r *GenericRepository[T]) Create(ctx context.Context, document T) (*T, error) {
	result, err := r.collection.InsertOne(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	filter := bson.M{"_id": result.InsertedID}
	created, err := r.FindOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created document: %w", err)
	}

	return created, nil
}

func (r *GenericRepository[T]) Update(ctx context.Context, filter bson.M, update bson.M, opts ...*options.UpdateOptions) (*T, error) {
	findAndUpdateOptions := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var result T
	err := r.collection.FindOneAndUpdate(ctx, filter, update, findAndUpdateOptions).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return &result, nil
}

func (r *GenericRepository[T]) Delete(ctx context.Context, filter bson.M, opts ...*options.DeleteOptions) error {
	result, err := r.collection.DeleteOne(ctx, filter, opts...)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no document found matching filter")
	}

	return nil
}

func (r *GenericRepository[T]) FindWithPagination(ctx context.Context, filter bson.M, pagination PaginationOptions, opts ...*options.FindOptions) (*PaginatedResult[T], error) {
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	skip := (pagination.Page - 1) * pagination.Limit

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(pagination.Limit)

	if len(opts) > 0 {
		for _, opt := range opts {
			if opt.Sort != nil {
				findOptions.SetSort(opt.Sort)
			}
			if opt.Projection != nil {
				findOptions.SetProjection(opt.Projection)
			}
		}
	}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to execute paginated find query: %w", err)
	}
	defer cursor.Close(ctx)

	var results []T
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode paginated find results: %w", err)
	}

	paginatedResult := &PaginatedResult[T]{
		Data:       results,
		Total:      total,
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
		HasNext:    pagination.Page < totalPages,
		HasPrev:    pagination.Page > 1,
	}

	return paginatedResult, nil
}

func (r *GenericRepository[T]) Count(ctx context.Context, filter bson.M, opts ...*options.CountOptions) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, filter, opts...)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}
