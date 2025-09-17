package database

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
)

type MinIOClient struct {
	Client     *minio.Client
	BucketName string
	config     config.MinIOConfig
}

type FileInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	LastModified string            `json:"last_modified"`
	ETag         string            `json:"etag"`
	Metadata     map[string]string `json:"metadata"`
}

type ChunkUploadInfo struct {
	UploadID   string `json:"upload_id"`
	Key        string `json:"key"`
	ChunkIndex int    `json:"chunk_index"`
	ETag       string `json:"etag"`
}

type MultipartUploadInfo struct {
	UploadID    string             `json:"upload_id"`
	Key         string             `json:"key"`
	TotalChunks int                `json:"total_chunks"`
	Chunks      []ChunkUploadInfo  `json:"chunks"`
}

func NewMinIOClient(cfg config.MinIOConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	minioClient := &MinIOClient{
		Client:     client,
		BucketName: cfg.BucketName,
		config:     cfg,
	}

	// Ensure bucket exists
	if err := minioClient.ensureBucketExists(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return minioClient, nil
}

func (m *MinIOClient) ensureBucketExists(ctx context.Context) error {
	exists, err := m.Client.BucketExists(ctx, m.BucketName)
	if err != nil {
		return fmt.Errorf("error checking bucket existence: %w", err)
	}

	if !exists {
		err = m.Client.MakeBucket(ctx, m.BucketName, minio.MakeBucketOptions{
			Region: m.config.Region,
		})
		if err != nil {
			return fmt.Errorf("error creating bucket: %w", err)
		}
	}
	return nil
}

func (m *MinIOClient) UploadFile(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (*FileInfo, error) {
	opts := minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: metadata,
	}

	uploadInfo, err := m.Client.PutObject(ctx, m.BucketName, key, reader, size, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &FileInfo{
		Key:         key,
		Size:        uploadInfo.Size,
		ContentType: contentType,
		ETag:        uploadInfo.ETag,
		Metadata:    metadata,
	}, nil
}

func (m *MinIOClient) InitiateMultipartUpload(ctx context.Context, key, contentType string, metadata map[string]string) (string, error) {
	// For MinIO v7, we need to use the Core API for multipart uploads
	core := &minio.Core{Client: m.Client}
	
	uploadID, err := core.NewMultipartUpload(ctx, m.BucketName, key, minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	return uploadID, nil
}

func (m *MinIOClient) UploadChunk(ctx context.Context, uploadID, key string, chunkIndex int, reader io.Reader, size int64) (*ChunkUploadInfo, error) {
	core := &minio.Core{Client: m.Client}
	
	part, err := core.PutObjectPart(ctx, m.BucketName, key, uploadID, chunkIndex, reader, size, minio.PutObjectPartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to upload chunk %d: %w", chunkIndex, err)
	}

	return &ChunkUploadInfo{
		UploadID:   uploadID,
		Key:        key,
		ChunkIndex: chunkIndex,
		ETag:       part.ETag,
	}, nil
}

func (m *MinIOClient) CompleteMultipartUpload(ctx context.Context, uploadID, key string, chunks []ChunkUploadInfo) (*FileInfo, error) {
	var completeParts []minio.CompletePart
	for _, chunk := range chunks {
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: chunk.ChunkIndex,
			ETag:       chunk.ETag,
		})
	}

	core := &minio.Core{Client: m.Client}
	uploadInfo, err := core.CompleteMultipartUpload(ctx, m.BucketName, key, uploadID, completeParts, minio.PutObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	// Get object info for complete file information
	objInfo, err := m.Client.StatObject(ctx, m.BucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info after upload: %w", err)
	}

	return &FileInfo{
		Key:          key,
		Size:         objInfo.Size,
		ContentType:  objInfo.ContentType,
		LastModified: objInfo.LastModified.Format("2006-01-02T15:04:05Z"),
		ETag:         uploadInfo.ETag,
		Metadata:     objInfo.UserMetadata,
	}, nil
}

func (m *MinIOClient) AbortMultipartUpload(ctx context.Context, uploadID, key string) error {
	core := &minio.Core{Client: m.Client}
	err := core.AbortMultipartUpload(ctx, m.BucketName, key, uploadID)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}
	return nil
}

func (m *MinIOClient) GetFile(ctx context.Context, key string) (io.ReadCloser, *FileInfo, error) {
	// Get object info first
	objInfo, err := m.Client.StatObject(ctx, m.BucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object info: %w", err)
	}

	// Get object
	object, err := m.Client.GetObject(ctx, m.BucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	fileInfo := &FileInfo{
		Key:          key,
		Size:         objInfo.Size,
		ContentType:  objInfo.ContentType,
		LastModified: objInfo.LastModified.Format("2006-01-02T15:04:05Z"),
		ETag:         objInfo.ETag,
		Metadata:     objInfo.UserMetadata,
	}

	return object, fileInfo, nil
}

func (m *MinIOClient) DeleteFile(ctx context.Context, key string) error {
	err := m.Client.RemoveObject(ctx, m.BucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (m *MinIOClient) ListFiles(ctx context.Context, prefix string, maxKeys int) ([]FileInfo, error) {
	var files []FileInfo

	options := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
		MaxKeys:   maxKeys,
	}

	for object := range m.Client.ListObjects(ctx, m.BucketName, options) {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}

		files = append(files, FileInfo{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified.Format("2006-01-02T15:04:05Z"),
			ETag:         object.ETag,
			ContentType:  object.ContentType,
		})
	}

	return files, nil
}

func (m *MinIOClient) GetFileURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := m.Client.PresignedGetObject(ctx, m.BucketName, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

func (m *MinIOClient) Close() error {
	// MinIO client doesn't need explicit closing
	return nil
}

// Helper function to calculate optimal chunk size based on file size
func CalculateOptimalChunkSize(fileSize int64, maxChunkSizeMB int) int64 {
	maxChunkSize := int64(maxChunkSizeMB * 1024 * 1024)
	
	// For files smaller than max chunk size, use single upload
	if fileSize <= maxChunkSize {
		return fileSize
	}
	
	// Calculate number of chunks needed
	numChunks := (fileSize + maxChunkSize - 1) / maxChunkSize
	
	// MinIO has a limit of 10,000 parts per upload
	if numChunks > 10000 {
		return (fileSize + 9999) / 10000
	}
	
	return maxChunkSize
}