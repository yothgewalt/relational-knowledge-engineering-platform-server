package mocks

import (
	"context"
	"io"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
)

// MinIOInterface defines the interface for MinIO operations
// This interface allows for easy mocking in tests
type MinIOInterface interface {
	UploadFile(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (*database.FileInfo, error)
	InitiateMultipartUpload(ctx context.Context, key, contentType string, metadata map[string]string) (string, error)
	UploadChunk(ctx context.Context, uploadID, key string, chunkIndex int, reader io.Reader, size int64) (*database.ChunkUploadInfo, error)
	CompleteMultipartUpload(ctx context.Context, uploadID, key string, chunks []database.ChunkUploadInfo) (*database.FileInfo, error)
	AbortMultipartUpload(ctx context.Context, uploadID, key string) error
	GetFile(ctx context.Context, key string) (io.ReadCloser, *database.FileInfo, error)
	DeleteFile(ctx context.Context, key string) error
	ListFiles(ctx context.Context, prefix string, maxKeys int) ([]database.FileInfo, error)
	GetFileURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Close() error
}

// RedisInterface defines the interface for Redis operations
type RedisInterface interface {
	Set(ctx context.Context, key, value string, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

// DatabaseManagerInterface defines the interface for database operations
type DatabaseManagerInterface interface {
	GetMinIO() MinIOInterface
	GetRedis() RedisInterface
	Close() error
}

// Ensure that the real MinIOClient implements the interface
var _ MinIOInterface = (*database.MinIOClient)(nil)