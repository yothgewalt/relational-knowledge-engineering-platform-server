package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
}

type HealthStatus struct {
	Connected     bool          `json:"connected"`
	Endpoint      string        `json:"endpoint"`
	Authenticated bool          `json:"authenticated"`
	BucketExists  bool          `json:"bucket_exists"`
	BucketName    string        `json:"bucket_name"`
	Latency       time.Duration `json:"latency"`
	Error         string        `json:"error,omitempty"`
}

type MinIOService interface {
	HealthCheck(ctx context.Context) HealthStatus
	GetClient() *minio.Client
	Close() error

	PutObject(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) error
	GetObject(ctx context.Context, objectName string) (*minio.Object, error)
	DeleteObject(ctx context.Context, objectName string) error
	ListObjects(ctx context.Context, prefix string, recursive bool) ([]minio.ObjectInfo, error)

	BucketExists(ctx context.Context, bucketName string) (bool, error)
	CreateBucket(ctx context.Context, bucketName string) error

	GeneratePresignedURL(ctx context.Context, objectName string, expires time.Duration, method string) (*url.URL, error)
	GetObjectInfo(ctx context.Context, objectName string) (minio.ObjectInfo, error)
}

type MinIOClient struct {
	client     *minio.Client
	config     MinIOConfig
	mu         sync.RWMutex
	bucketName string
}

func NewMinIOService(config MinIOConfig) (*MinIOClient, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("MinIO endpoint is required")
	}
	if config.BucketName == "" {
		return nil, fmt.Errorf("MinIO bucket name is required")
	}
	if config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return nil, fmt.Errorf("MinIO credentials are required")
	}

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	}

	client, err := minio.New(config.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	minioClient := &MinIOClient{
		client:     client,
		config:     config,
		bucketName: config.BucketName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, config.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := minioClient.CreateBucket(ctx, config.BucketName); err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", config.BucketName, err)
		}
	}

	return minioClient, nil
}

func (m *MinIOClient) HealthCheck(ctx context.Context) HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	start := time.Now()
	status := HealthStatus{
		Endpoint:   m.config.Endpoint,
		BucketName: m.bucketName,
	}

	exists, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		status.Connected = false
		status.Authenticated = false
		status.BucketExists = false
		status.Error = fmt.Sprintf("failed to check bucket existence: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Connected = true
	status.Authenticated = true
	status.BucketExists = exists
	status.Latency = time.Since(start)

	if !exists {
		status.Error = fmt.Sprintf("bucket %s does not exist", m.bucketName)
	}

	return status
}

func (m *MinIOClient) GetClient() *minio.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

func (m *MinIOClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *MinIOClient) PutObject(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, err := m.client.PutObject(ctx, m.bucketName, objectName, reader, objectSize, opts)
	if err != nil {
		return fmt.Errorf("failed to put object %s: %w", objectName, err)
	}

	return nil
}

func (m *MinIOClient) GetObject(ctx context.Context, objectName string) (*minio.Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", objectName, err)
	}

	return obj, nil
}

func (m *MinIOClient) DeleteObject(ctx context.Context, objectName string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", objectName, err)
	}

	return nil
}

func (m *MinIOClient) ListObjects(ctx context.Context, prefix string, recursive bool) ([]minio.ObjectInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objectCh := m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	})

	var objects []minio.ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func (m *MinIOClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return exists, nil
}

func (m *MinIOClient) CreateBucket(ctx context.Context, bucketName string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	err := m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
	}

	return nil
}

func (m *MinIOClient) GeneratePresignedURL(ctx context.Context, objectName string, expires time.Duration, method string) (*url.URL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var presignedURL *url.URL
	var err error

	switch method {
	case "GET":
		presignedURL, err = m.client.PresignedGetObject(ctx, m.bucketName, objectName, expires, make(url.Values))
	case "PUT":
		presignedURL, err = m.client.PresignedPutObject(ctx, m.bucketName, objectName, expires)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL for %s: %w", objectName, err)
	}

	return presignedURL, nil
}

func (m *MinIOClient) GetObjectInfo(ctx context.Context, objectName string) (minio.ObjectInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objInfo, err := m.client.StatObject(ctx, m.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return minio.ObjectInfo{}, fmt.Errorf("failed to get object info for %s: %w", objectName, err)
	}

	return objInfo, nil
}