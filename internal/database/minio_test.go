package database

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
)

func TestCalculateOptimalChunkSize(t *testing.T) {
	tests := []struct {
		name             string
		fileSize         int64
		maxChunkSizeMB   int
		expectedChunkSize int64
	}{
		{
			name:             "Small file - single chunk",
			fileSize:         5 * 1024 * 1024, // 5MB
			maxChunkSizeMB:   10,
			expectedChunkSize: 5 * 1024 * 1024, // Should return file size
		},
		{
			name:             "Large file - normal chunking",
			fileSize:         50 * 1024 * 1024, // 50MB
			maxChunkSizeMB:   10,
			expectedChunkSize: 10 * 1024 * 1024, // 10MB chunks
		},
		{
			name:             "Very large file - over 10000 parts limit",
			fileSize:         100000 * 1024 * 1024, // 100GB
			maxChunkSizeMB:   10,
			expectedChunkSize: (100000*1024*1024 + 9999) / 10000, // Should calculate to stay under 10000 parts
		},
		{
			name:             "Exact chunk size boundary",
			fileSize:         10 * 1024 * 1024, // 10MB
			maxChunkSizeMB:   10,
			expectedChunkSize: 10 * 1024 * 1024, // Should return file size
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateOptimalChunkSize(tt.fileSize, tt.maxChunkSizeMB)
			assert.Equal(t, tt.expectedChunkSize, result)
		})
	}
}

func TestNewMinIOClient(t *testing.T) {
	tests := []struct {
		name        string
		config      config.MinIOConfig
		expectError bool
	}{
		{
			name: "Valid configuration",
			config: config.MinIOConfig{
				Endpoint:        "localhost:9000",
				AccessKeyID:     "testkey",
				SecretAccessKey: "testsecret",
				UseSSL:          false,
				BucketName:      "test-bucket",
				Region:          "us-east-1",
			},
			expectError: true, // Will fail because no actual MinIO server
		},
		{
			name: "Invalid endpoint",
			config: config.MinIOConfig{
				Endpoint:        "",
				AccessKeyID:     "testkey",
				SecretAccessKey: "testsecret",
				UseSSL:          false,
				BucketName:      "test-bucket",
				Region:          "us-east-1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewMinIOClient(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// MockMinIOClient implements the MinIO interface for testing
type MockMinIOClient struct {
	files       map[string][]byte
	metadata    map[string]map[string]string
	uploads     map[string]*MockMultipartUpload
	failUpload  bool
	failGet     bool
	failDelete  bool
	failList    bool
}

type MockMultipartUpload struct {
	uploadID    string
	key         string
	chunks      map[int][]byte
	contentType string
	metadata    map[string]string
}

func NewMockMinIOClient() *MockMinIOClient {
	return &MockMinIOClient{
		files:    make(map[string][]byte),
		metadata: make(map[string]map[string]string),
		uploads:  make(map[string]*MockMultipartUpload),
	}
}

func (m *MockMinIOClient) UploadFile(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (*FileInfo, error) {
	if m.failUpload {
		return nil, fmt.Errorf("mock upload failure")
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	m.files[key] = data
	m.metadata[key] = metadata

	return &FileInfo{
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		ETag:        "mock-etag",
		Metadata:    metadata,
	}, nil
}

func (m *MockMinIOClient) InitiateMultipartUpload(ctx context.Context, key, contentType string, metadata map[string]string) (string, error) {
	if m.failUpload {
		return "", fmt.Errorf("mock initiate upload failure")
	}

	uploadID := fmt.Sprintf("mock-upload-%d", len(m.uploads))
	m.uploads[uploadID] = &MockMultipartUpload{
		uploadID:    uploadID,
		key:         key,
		chunks:      make(map[int][]byte),
		contentType: contentType,
		metadata:    metadata,
	}

	return uploadID, nil
}

func (m *MockMinIOClient) UploadChunk(ctx context.Context, uploadID, key string, chunkIndex int, reader io.Reader, size int64) (*ChunkUploadInfo, error) {
	if m.failUpload {
		return nil, fmt.Errorf("mock chunk upload failure")
	}

	upload, exists := m.uploads[uploadID]
	if !exists {
		return nil, fmt.Errorf("upload not found")
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	upload.chunks[chunkIndex] = data

	return &ChunkUploadInfo{
		UploadID:   uploadID,
		Key:        key,
		ChunkIndex: chunkIndex,
		ETag:       fmt.Sprintf("mock-etag-%d", chunkIndex),
	}, nil
}

func (m *MockMinIOClient) CompleteMultipartUpload(ctx context.Context, uploadID, key string, chunks []ChunkUploadInfo) (*FileInfo, error) {
	if m.failUpload {
		return nil, fmt.Errorf("mock complete upload failure")
	}

	upload, exists := m.uploads[uploadID]
	if !exists {
		return nil, fmt.Errorf("upload not found")
	}

	// Combine all chunks in order
	var fullData []byte
	for i := 1; i <= len(chunks); i++ {
		chunkData, exists := upload.chunks[i]
		if !exists {
			return nil, fmt.Errorf("missing chunk %d", i)
		}
		fullData = append(fullData, chunkData...)
	}

	m.files[key] = fullData
	m.metadata[key] = upload.metadata
	delete(m.uploads, uploadID)

	return &FileInfo{
		Key:          key,
		Size:         int64(len(fullData)),
		ContentType:  upload.contentType,
		LastModified: time.Now().Format("2006-01-02T15:04:05Z"),
		ETag:         "mock-final-etag",
		Metadata:     upload.metadata,
	}, nil
}

func (m *MockMinIOClient) AbortMultipartUpload(ctx context.Context, uploadID, key string) error {
	if m.failUpload {
		return fmt.Errorf("mock abort upload failure")
	}

	delete(m.uploads, uploadID)
	return nil
}

func (m *MockMinIOClient) GetFile(ctx context.Context, key string) (io.ReadCloser, *FileInfo, error) {
	if m.failGet {
		return nil, nil, fmt.Errorf("mock get failure")
	}

	data, exists := m.files[key]
	if !exists {
		return nil, nil, fmt.Errorf("file not found")
	}

	metadata := m.metadata[key]
	if metadata == nil {
		metadata = make(map[string]string)
	}

	reader := io.NopCloser(bytes.NewReader(data))
	fileInfo := &FileInfo{
		Key:          key,
		Size:         int64(len(data)),
		ContentType:  "application/octet-stream",
		LastModified: time.Now().Format("2006-01-02T15:04:05Z"),
		ETag:         "mock-etag",
		Metadata:     metadata,
	}

	return reader, fileInfo, nil
}

func (m *MockMinIOClient) DeleteFile(ctx context.Context, key string) error {
	if m.failDelete {
		return fmt.Errorf("mock delete failure")
	}

	delete(m.files, key)
	delete(m.metadata, key)
	return nil
}

func (m *MockMinIOClient) ListFiles(ctx context.Context, prefix string, maxKeys int) ([]FileInfo, error) {
	if m.failList {
		return nil, fmt.Errorf("mock list failure")
	}

	var files []FileInfo
	count := 0
	for key, data := range m.files {
		if maxKeys > 0 && count >= maxKeys {
			break
		}
		if strings.HasPrefix(key, prefix) {
			metadata := m.metadata[key]
			if metadata == nil {
				metadata = make(map[string]string)
			}
			files = append(files, FileInfo{
				Key:          key,
				Size:         int64(len(data)),
				LastModified: time.Now().Format("2006-01-02T15:04:05Z"),
				ETag:         "mock-etag",
				ContentType:  "application/octet-stream",
			})
			count++
		}
	}
	return files, nil
}

func (m *MockMinIOClient) GetFileURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.failGet {
		return "", fmt.Errorf("mock get URL failure")
	}
	return fmt.Sprintf("https://mock-url/%s?expires=%d", key, time.Now().Add(expiry).Unix()), nil
}

func (m *MockMinIOClient) Close() error {
	return nil
}

func TestMockMinIOClient_UploadFile(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		content     string
		contentType string
		metadata    map[string]string
		failUpload  bool
		expectError bool
	}{
		{
			name:        "Successful upload",
			key:         "test/file.txt",
			content:     "test content",
			contentType: "text/plain",
			metadata:    map[string]string{"author": "test"},
			failUpload:  false,
			expectError: false,
		},
		{
			name:        "Upload failure",
			key:         "test/file.txt",
			content:     "test content",
			contentType: "text/plain",
			metadata:    nil,
			failUpload:  true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockMinIOClient()
			mock.failUpload = tt.failUpload

			reader := strings.NewReader(tt.content)
			ctx := context.Background()

			fileInfo, err := mock.UploadFile(ctx, tt.key, reader, int64(len(tt.content)), tt.contentType, tt.metadata)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, fileInfo)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, fileInfo)
				assert.Equal(t, tt.key, fileInfo.Key)
				assert.Equal(t, int64(len(tt.content)), fileInfo.Size)
				assert.Equal(t, tt.contentType, fileInfo.ContentType)
				if tt.metadata != nil {
					assert.Equal(t, tt.metadata, fileInfo.Metadata)
				}

				// Verify file was stored
				storedData, exists := mock.files[tt.key]
				assert.True(t, exists)
				assert.Equal(t, []byte(tt.content), storedData)
			}
		})
	}
}

func TestMockMinIOClient_MultipartUpload(t *testing.T) {
	mock := NewMockMinIOClient()
	ctx := context.Background()

	// Test multipart upload workflow
	key := "test/large-file.bin"
	contentType := "application/octet-stream"
	metadata := map[string]string{"test": "value"}

	// 1. Initiate upload
	uploadID, err := mock.InitiateMultipartUpload(ctx, key, contentType, metadata)
	require.NoError(t, err)
	assert.NotEmpty(t, uploadID)

	// 2. Upload chunks
	chunks := []string{"chunk1", "chunk2", "chunk3"}
	var chunkInfos []ChunkUploadInfo

	for i, chunkData := range chunks {
		chunkIndex := i + 1
		reader := strings.NewReader(chunkData)
		
		chunkInfo, err := mock.UploadChunk(ctx, uploadID, key, chunkIndex, reader, int64(len(chunkData)))
		require.NoError(t, err)
		require.NotNil(t, chunkInfo)
		
		assert.Equal(t, uploadID, chunkInfo.UploadID)
		assert.Equal(t, key, chunkInfo.Key)
		assert.Equal(t, chunkIndex, chunkInfo.ChunkIndex)
		assert.NotEmpty(t, chunkInfo.ETag)
		
		chunkInfos = append(chunkInfos, *chunkInfo)
	}

	// 3. Complete upload
	fileInfo, err := mock.CompleteMultipartUpload(ctx, uploadID, key, chunkInfos)
	require.NoError(t, err)
	require.NotNil(t, fileInfo)

	assert.Equal(t, key, fileInfo.Key)
	assert.Equal(t, int64(len("chunk1chunk2chunk3")), fileInfo.Size)
	assert.Equal(t, contentType, fileInfo.ContentType)
	assert.Equal(t, metadata, fileInfo.Metadata)

	// Verify file was stored correctly
	storedData, exists := mock.files[key]
	assert.True(t, exists)
	assert.Equal(t, []byte("chunk1chunk2chunk3"), storedData)

	// Verify upload was cleaned up
	_, exists = mock.uploads[uploadID]
	assert.False(t, exists)
}

func TestMockMinIOClient_MultipartUpload_Errors(t *testing.T) {
	tests := []struct {
		name        string
		failUpload  bool
		expectError bool
	}{
		{
			name:        "Initiate upload failure",
			failUpload:  true,
			expectError: true,
		},
		{
			name:        "Upload chunk with invalid upload ID",
			failUpload:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockMinIOClient()
			mock.failUpload = tt.failUpload
			ctx := context.Background()

			if tt.name == "Initiate upload failure" {
				_, err := mock.InitiateMultipartUpload(ctx, "test/file", "text/plain", nil)
				assert.Error(t, err)
			} else if tt.name == "Upload chunk with invalid upload ID" {
				// Try to upload chunk with invalid upload ID
				reader := strings.NewReader("test")
				_, err := mock.UploadChunk(ctx, "invalid-upload-id", "test/file", 1, reader, 4)
				assert.Error(t, err)
			}
		})
	}
}

func TestMockMinIOClient_GetFile(t *testing.T) {
	mock := NewMockMinIOClient()
	ctx := context.Background()

	// Setup test file
	testKey := "test/file.txt"
	testContent := "test file content"
	testMetadata := map[string]string{"key": "value"}

	mock.files[testKey] = []byte(testContent)
	mock.metadata[testKey] = testMetadata

	tests := []struct {
		name        string
		key         string
		failGet     bool
		expectError bool
	}{
		{
			name:        "Get existing file",
			key:         testKey,
			failGet:     false,
			expectError: false,
		},
		{
			name:        "Get non-existent file",
			key:         "non/existent.txt",
			failGet:     false,
			expectError: true,
		},
		{
			name:        "Get file with failure",
			key:         testKey,
			failGet:     true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.failGet = tt.failGet

			reader, fileInfo, err := mock.GetFile(ctx, tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reader)
				assert.Nil(t, fileInfo)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, reader)
				require.NotNil(t, fileInfo)

				// Verify file info
				assert.Equal(t, tt.key, fileInfo.Key)
				assert.Equal(t, int64(len(testContent)), fileInfo.Size)
				assert.Equal(t, testMetadata, fileInfo.Metadata)

				// Verify content
				data, err := io.ReadAll(reader)
				require.NoError(t, err)
				assert.Equal(t, []byte(testContent), data)

				reader.Close()
			}
		})
	}
}

func TestMockMinIOClient_DeleteFile(t *testing.T) {
	tests := []struct {
		name        string
		failDelete  bool
		expectError bool
	}{
		{
			name:        "Successful delete",
			failDelete:  false,
			expectError: false,
		},
		{
			name:        "Delete failure",
			failDelete:  true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh mock for each test case
			mock := NewMockMinIOClient()
			ctx := context.Background()

			// Setup test file
			testKey := "test/file.txt"
			mock.files[testKey] = []byte("test content")
			mock.failDelete = tt.failDelete

			err := mock.DeleteFile(ctx, testKey)

			if tt.expectError {
				assert.Error(t, err)
				// File should still exist
				_, exists := mock.files[testKey]
				assert.True(t, exists)
			} else {
				assert.NoError(t, err)
				// File should be deleted
				_, exists := mock.files[testKey]
				assert.False(t, exists)
			}
		})
	}
}

func TestMockMinIOClient_ListFiles(t *testing.T) {
	mock := NewMockMinIOClient()
	ctx := context.Background()

	// Setup test files
	testFiles := map[string]string{
		"prefix1/file1.txt": "content1",
		"prefix1/file2.txt": "content2",
		"prefix2/file3.txt": "content3",
		"other/file4.txt":   "content4",
	}

	for key, content := range testFiles {
		mock.files[key] = []byte(content)
	}

	tests := []struct {
		name         string
		prefix       string
		maxKeys      int
		failList     bool
		expectError  bool
		expectedCount int
	}{
		{
			name:         "List with prefix",
			prefix:       "prefix1/",
			maxKeys:      10,
			failList:     false,
			expectError:  false,
			expectedCount: 2,
		},
		{
			name:         "List all files",
			prefix:       "",
			maxKeys:      10,
			failList:     false,
			expectError:  false,
			expectedCount: 4,
		},
		{
			name:         "List with max keys limit",
			prefix:       "",
			maxKeys:      2,
			failList:     false,
			expectError:  false,
			expectedCount: 2,
		},
		{
			name:        "List failure",
			prefix:      "",
			maxKeys:     10,
			failList:    true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.failList = tt.failList

			files, err := mock.ListFiles(ctx, tt.prefix, tt.maxKeys)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, files)
			} else {
				assert.NoError(t, err)
				assert.Len(t, files, tt.expectedCount)

				// Verify prefix matching
				for _, file := range files {
					assert.True(t, strings.HasPrefix(file.Key, tt.prefix))
				}
			}
		})
	}
}

func TestMockMinIOClient_AbortMultipartUpload(t *testing.T) {
	tests := []struct {
		name        string
		failUpload  bool
		expectError bool
	}{
		{
			name:        "Successful abort",
			failUpload:  false,
			expectError: false,
		},
		{
			name:        "Abort failure",
			failUpload:  true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh mock for each test case
			mock := NewMockMinIOClient()
			ctx := context.Background()

			// Setup multipart upload
			uploadID, err := mock.InitiateMultipartUpload(ctx, "test/file", "text/plain", nil)
			require.NoError(t, err)

			mock.failUpload = tt.failUpload

			err = mock.AbortMultipartUpload(ctx, uploadID, "test/file")

			if tt.expectError {
				assert.Error(t, err)
				// Upload should still exist
				_, exists := mock.uploads[uploadID]
				assert.True(t, exists)
			} else {
				assert.NoError(t, err)
				// Upload should be removed
				_, exists := mock.uploads[uploadID]
				assert.False(t, exists)
			}
		})
	}
}

func TestMockMinIOClient_GetFileURL(t *testing.T) {
	mock := NewMockMinIOClient()
	ctx := context.Background()
	expiry := 1 * time.Hour

	tests := []struct {
		name        string
		key         string
		failGet     bool
		expectError bool
	}{
		{
			name:        "Generate URL successfully",
			key:         "test/file.txt",
			failGet:     false,
			expectError: false,
		},
		{
			name:        "Generate URL failure",
			key:         "test/file.txt",
			failGet:     true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.failGet = tt.failGet

			url, err := mock.GetFileURL(ctx, tt.key, expiry)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, url, tt.key)
				assert.Contains(t, url, "expires=")
			}
		})
	}
}