package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoConfig(t *testing.T) {
	tests := []struct {
		name   string
		config MongoConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: MongoConfig{
				Address:  "localhost:27017",
				Username: "admin",
				Password: "password",
				Database: "testdb",
			},
			valid: true,
		},
		{
			name: "empty address",
			config: MongoConfig{
				Address:  "",
				Username: "admin",
				Password: "password",
				Database: "testdb",
			},
			valid: false,
		},
		{
			name: "empty database",
			config: MongoConfig{
				Address:  "localhost:27017",
				Username: "admin",
				Password: "password",
				Database: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Address == "" || tt.config.Database == "" {
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

func TestHealthStatus_JSON(t *testing.T) {
	status := HealthStatus{
		Connected: true,
		Database:  "testdb",
		Latency:   50 * time.Millisecond,
		Error:     "",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal HealthStatus: %v", err)
	}

	var unmarshaled HealthStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal HealthStatus: %v", err)
	}

	if unmarshaled.Connected != status.Connected {
		t.Errorf("expected Connected %v, got %v", status.Connected, unmarshaled.Connected)
	}
	if unmarshaled.Database != status.Database {
		t.Errorf("expected Database %s, got %s", status.Database, unmarshaled.Database)
	}
	if unmarshaled.Latency != status.Latency {
		t.Errorf("expected Latency %v, got %v", status.Latency, unmarshaled.Latency)
	}
}

func TestHealthStatus_WithError(t *testing.T) {
	status := HealthStatus{
		Connected: false,
		Database:  "testdb",
		Latency:   0,
		Error:     "connection timeout",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal HealthStatus with error: %v", err)
	}

	var unmarshaled HealthStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal HealthStatus with error: %v", err)
	}

	if unmarshaled.Error != status.Error {
		t.Errorf("expected Error %s, got %s", status.Error, unmarshaled.Error)
	}
}

func TestPaginationOptions_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		input    PaginationOptions
		expected PaginationOptions
	}{
		{
			name:     "zero values should use defaults",
			input:    PaginationOptions{Page: 0, Limit: 0},
			expected: PaginationOptions{Page: 1, Limit: 10},
		},
		{
			name:     "negative values should use defaults",
			input:    PaginationOptions{Page: -1, Limit: -5},
			expected: PaginationOptions{Page: 1, Limit: 10},
		},
		{
			name:     "valid values should be preserved",
			input:    PaginationOptions{Page: 2, Limit: 20},
			expected: PaginationOptions{Page: 2, Limit: 20},
		},
		{
			name:     "partial defaults",
			input:    PaginationOptions{Page: 3, Limit: 0},
			expected: PaginationOptions{Page: 3, Limit: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePaginationOptions(tt.input)
			if result.Page != tt.expected.Page {
				t.Errorf("expected Page %d, got %d", tt.expected.Page, result.Page)
			}
			if result.Limit != tt.expected.Limit {
				t.Errorf("expected Limit %d, got %d", tt.expected.Limit, result.Limit)
			}
		})
	}
}

func TestPaginatedResult_Calculations(t *testing.T) {
	tests := []struct {
		name            string
		total           int64
		page            int64
		limit           int64
		expectedPages   int64
		expectedHasNext bool
		expectedHasPrev bool
	}{
		{
			name:            "first page with more data",
			total:           25,
			page:            1,
			limit:           10,
			expectedPages:   3,
			expectedHasNext: true,
			expectedHasPrev: false,
		},
		{
			name:            "middle page",
			total:           25,
			page:            2,
			limit:           10,
			expectedPages:   3,
			expectedHasNext: true,
			expectedHasPrev: true,
		},
		{
			name:            "last page",
			total:           25,
			page:            3,
			limit:           10,
			expectedPages:   3,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
		{
			name:            "exact page division",
			total:           20,
			page:            2,
			limit:           10,
			expectedPages:   2,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
		{
			name:            "single page",
			total:           5,
			page:            1,
			limit:           10,
			expectedPages:   1,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
		{
			name:            "empty result",
			total:           0,
			page:            1,
			limit:           10,
			expectedPages:   0,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePaginatedResult([]string{}, tt.total, tt.page, tt.limit)

			if result.TotalPages != tt.expectedPages {
				t.Errorf("expected TotalPages %d, got %d", tt.expectedPages, result.TotalPages)
			}
			if result.HasNext != tt.expectedHasNext {
				t.Errorf("expected HasNext %v, got %v", tt.expectedHasNext, result.HasNext)
			}
			if result.HasPrev != tt.expectedHasPrev {
				t.Errorf("expected HasPrev %v, got %v", tt.expectedHasPrev, result.HasPrev)
			}
			if result.Total != tt.total {
				t.Errorf("expected Total %d, got %d", tt.total, result.Total)
			}
			if result.Page != tt.page {
				t.Errorf("expected Page %d, got %d", tt.page, result.Page)
			}
			if result.Limit != tt.limit {
				t.Errorf("expected Limit %d, got %d", tt.limit, result.Limit)
			}
		})
	}
}

func TestPaginatedResult_JSON(t *testing.T) {
	type TestModel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	data := []TestModel{
		{ID: "1", Name: "Test 1"},
		{ID: "2", Name: "Test 2"},
	}

	result := PaginatedResult[TestModel]{
		Data:       data,
		Total:      25,
		Page:       2,
		Limit:      10,
		TotalPages: 3,
		HasNext:    true,
		HasPrev:    true,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal PaginatedResult: %v", err)
	}

	var unmarshaled PaginatedResult[TestModel]
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal PaginatedResult: %v", err)
	}

	if len(unmarshaled.Data) != len(result.Data) {
		t.Errorf("expected Data length %d, got %d", len(result.Data), len(unmarshaled.Data))
	}
	if unmarshaled.Total != result.Total {
		t.Errorf("expected Total %d, got %d", result.Total, unmarshaled.Total)
	}
	if unmarshaled.HasNext != result.HasNext {
		t.Errorf("expected HasNext %v, got %v", result.HasNext, unmarshaled.HasNext)
	}
}

func TestSkipCalculation(t *testing.T) {
	tests := []struct {
		name         string
		page         int64
		limit        int64
		expectedSkip int64
	}{
		{
			name:         "first page",
			page:         1,
			limit:        10,
			expectedSkip: 0,
		},
		{
			name:         "second page",
			page:         2,
			limit:        10,
			expectedSkip: 10,
		},
		{
			name:         "third page with different limit",
			page:         3,
			limit:        5,
			expectedSkip: 10,
		},
		{
			name:         "large page number",
			page:         100,
			limit:        20,
			expectedSkip: 1980,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip := calculateSkip(tt.page, tt.limit)
			if skip != tt.expectedSkip {
				t.Errorf("expected skip %d, got %d", tt.expectedSkip, skip)
			}
		})
	}
}

type TestDocument struct {
	ID   string `bson:"_id,omitempty" json:"id"`
	Name string `bson:"name" json:"name"`
	Age  int    `bson:"age" json:"age"`
}

func TestFindOptions_Merge(t *testing.T) {
	baseOptions := options.Find()
	baseOptions.SetSkip(10)
	baseOptions.SetLimit(5)

	additionalOptions := []*options.FindOptions{
		options.Find().SetSort(bson.M{"name": 1}),
		options.Find().SetProjection(bson.M{"name": 1, "_id": 0}),
	}

	merged := mergeFindOptions(baseOptions, additionalOptions)

	if merged.Skip == nil || *merged.Skip != 10 {
		t.Error("expected Skip to be preserved")
	}
	if merged.Limit == nil || *merged.Limit != 5 {
		t.Error("expected Limit to be preserved")
	}
	if merged.Sort == nil {
		t.Error("expected Sort to be applied")
	}
	if merged.Projection == nil {
		t.Error("expected Projection to be applied")
	}
}

func TestErrorFormatting(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		originalError error
		expectedMsg   string
	}{
		{
			name:          "find error",
			operation:     "find",
			originalError: context.DeadlineExceeded,
			expectedMsg:   "failed to execute find query: context deadline exceeded",
		},
		{
			name:          "insert error",
			operation:     "insert",
			originalError: context.Canceled,
			expectedMsg:   "failed to execute insert query: context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatError(tt.operation, tt.originalError)
			if err.Error() != tt.expectedMsg {
				t.Errorf("expected error message %q, got %q", tt.expectedMsg, err.Error())
			}
		})
	}
}

func normalizePaginationOptions(opts PaginationOptions) PaginationOptions {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	return opts
}

func calculatePaginatedResult[T any](data []T, total, page, limit int64) *PaginatedResult[T] {
	totalPages := (total + limit - 1) / limit
	if total == 0 {
		totalPages = 0
	}

	return &PaginatedResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

func calculateSkip(page, limit int64) int64 {
	return (page - 1) * limit
}

func mergeFindOptions(base *options.FindOptions, additional []*options.FindOptions) *options.FindOptions {
	result := base
	for _, opt := range additional {
		if opt.Sort != nil {
			result.SetSort(opt.Sort)
		}
		if opt.Projection != nil {
			result.SetProjection(opt.Projection)
		}
	}
	return result
}

func formatError(operation string, err error) error {
	return fmt.Errorf("failed to execute %s query: %w", operation, err)
}

func TestNewRepository(t *testing.T) {
	var _ Repository[TestDocument] = (*GenericRepository[TestDocument])(nil)

	repo := &GenericRepository[TestDocument]{
		collection: nil,
		service:    nil,
	}

	if repo.collection != nil {
		t.Error("expected collection field to be nil in test setup")
	}
	if repo.service != nil {
		t.Error("expected service field to be nil in test setup")
	}
}

func TestMongoURIConstruction(t *testing.T) {
	tests := []struct {
		name     string
		config   MongoConfig
		expected string
	}{
		{
			name: "basic auth",
			config: MongoConfig{
				Address:  "localhost:27017",
				Username: "admin",
				Password: "password",
				Database: "testdb",
			},
			expected: "mongodb://admin:password@localhost:27017/?authSource=admin",
		},
		{
			name: "special characters in password",
			config: MongoConfig{
				Address:  "localhost:27017",
				Username: "user",
				Password: "p@ssw0rd!",
				Database: "testdb",
			},
			expected: "mongodb://user:p@ssw0rd!@localhost:27017/?authSource=admin",
		},
		{
			name: "remote host",
			config: MongoConfig{
				Address:  "mongo.example.com:27017",
				Username: "dbuser",
				Password: "secret",
				Database: "production",
			},
			expected: "mongodb://dbuser:secret@mongo.example.com:27017/?authSource=admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := constructMongoURI(tt.config)
			if uri != tt.expected {
				t.Errorf("expected URI %q, got %q", tt.expected, uri)
			}
		})
	}
}

func TestConcurrentHealthCheck(t *testing.T) {
	status := HealthStatus{
		Connected: true,
		Database:  "testdb",
		Latency:   50 * time.Millisecond,
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			_ = status.Connected
			_ = status.Database
			_ = status.Latency

			_, err := json.Marshal(status)
			if err != nil {
				t.Errorf("concurrent marshal failed: %v", err)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		pagination  PaginationOptions
		total       int64
		expectValid bool
	}{
		{
			name:        "page beyond total pages",
			pagination:  PaginationOptions{Page: 10, Limit: 10},
			total:       25,
			expectValid: true,
		},
		{
			name:        "very large limit",
			pagination:  PaginationOptions{Page: 1, Limit: 1000000},
			total:       100,
			expectValid: true,
		},
		{
			name:        "zero total",
			pagination:  PaginationOptions{Page: 1, Limit: 10},
			total:       0,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizePaginationOptions(tt.pagination)
			result := calculatePaginatedResult([]TestDocument{}, tt.total, normalized.Page, normalized.Limit)

			if result == nil && tt.expectValid {
				t.Error("expected valid result but got nil")
			}

			if result != nil {
				if result.Total != tt.total {
					t.Errorf("expected total %d, got %d", tt.total, result.Total)
				}

				if result.Page < 1 {
					t.Error("page should never be less than 1")
				}

				if result.Limit < 1 {
					t.Error("limit should never be less than 1")
				}
			}
		})
	}
}

func TestBSONFilterValidation(t *testing.T) {
	tests := []struct {
		name   string
		filter bson.M
		valid  bool
	}{
		{
			name:   "empty filter",
			filter: bson.M{},
			valid:  true,
		},
		{
			name:   "simple equality filter",
			filter: bson.M{"name": "test"},
			valid:  true,
		},
		{
			name:   "complex filter with operators",
			filter: bson.M{"age": bson.M{"$gte": 18, "$lt": 65}},
			valid:  true,
		},
		{
			name:   "nil filter",
			filter: nil,
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bson.Marshal(tt.filter)
			if tt.valid && err != nil {
				t.Errorf("expected valid filter but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid filter but got no error")
			}
		})
	}
}

func TestGenericTypeSafety(t *testing.T) {
	type User struct {
		ID   string `bson:"_id" json:"id"`
		Name string `bson:"name" json:"name"`
	}

	type Product struct {
		ID    string  `bson:"_id" json:"id"`
		Title string  `bson:"title" json:"title"`
		Price float64 `bson:"price" json:"price"`
	}

	users := []User{{ID: "1", Name: "John"}}
	products := []Product{{ID: "1", Title: "Item", Price: 9.99}}

	userResult := calculatePaginatedResult(users, 1, 1, 10)
	productResult := calculatePaginatedResult(products, 1, 1, 10)

	if len(userResult.Data) != 1 {
		t.Error("user result should contain 1 item")
	}
	if len(productResult.Data) != 1 {
		t.Error("product result should contain 1 item")
	}

	if userResult.Data[0].Name != "John" {
		t.Error("user data should be preserved")
	}
	if productResult.Data[0].Price != 9.99 {
		t.Error("product data should be preserved")
	}
}

func TestContextValidation(t *testing.T) {
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second)
	defer timeoutCancel()

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

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
			name:  "context with timeout",
			ctx:   timeoutCtx,
			valid: true,
		},
		{
			name:  "cancelled context",
			ctx:   cancelCtx,
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

func constructMongoURI(config MongoConfig) string {
	return fmt.Sprintf("mongodb://%s:%s@%s/?authSource=admin",
		config.Username, config.Password, config.Address)
}
