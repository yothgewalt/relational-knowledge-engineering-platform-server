package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
)

// Mock implementations for testing
type MockMinIOClient struct {
	mock.Mock
}

func (m *MockMinIOClient) UploadFile(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (*database.FileInfo, error) {
	args := m.Called(ctx, key, reader, size, contentType, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.FileInfo), args.Error(1)
}

func (m *MockMinIOClient) InitiateMultipartUpload(ctx context.Context, key, contentType string, metadata map[string]string) (string, error) {
	args := m.Called(ctx, key, contentType, metadata)
	return args.String(0), args.Error(1)
}

func (m *MockMinIOClient) UploadChunk(ctx context.Context, uploadID, key string, chunkIndex int, reader io.Reader, size int64) (*database.ChunkUploadInfo, error) {
	args := m.Called(ctx, uploadID, key, chunkIndex, reader, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.ChunkUploadInfo), args.Error(1)
}

func (m *MockMinIOClient) CompleteMultipartUpload(ctx context.Context, uploadID, key string, chunks []database.ChunkUploadInfo) (*database.FileInfo, error) {
	args := m.Called(ctx, uploadID, key, chunks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.FileInfo), args.Error(1)
}

func (m *MockMinIOClient) AbortMultipartUpload(ctx context.Context, uploadID, key string) error {
	args := m.Called(ctx, uploadID, key)
	return args.Error(0)
}

func (m *MockMinIOClient) GetFile(ctx context.Context, key string) (io.ReadCloser, *database.FileInfo, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(io.ReadCloser), args.Get(1).(*database.FileInfo), args.Error(2)
}

func (m *MockMinIOClient) DeleteFile(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockMinIOClient) ListFiles(ctx context.Context, prefix string, maxKeys int) ([]database.FileInfo, error) {
	args := m.Called(ctx, prefix, maxKeys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.FileInfo), args.Error(1)
}

func (m *MockMinIOClient) GetFileURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockMinIOClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockRedisClient struct {
	mock.Mock
	data map[string]string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
	}
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	if args.Error(0) == nil {
		m.data[key] = value.(string)
	}
	return args.Error(0)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	if args.Error(1) == nil {
		if value, exists := m.data[key]; exists {
			return value, nil
		}
	}
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	if args.Error(0) == nil {
		for _, key := range keys {
			delete(m.data, key)
		}
	}
	return args.Error(0)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockDatabaseManager struct {
	MinIO *MockMinIOClient
	Redis *MockRedisClient
}


func setupTestHandler() (*UploadHandler, *MockDatabaseManager) {
	mockMinIO := &MockMinIOClient{}
	mockRedis := NewMockRedisClient()

	dbManager := &MockDatabaseManager{
		MinIO: mockMinIO,
		Redis: mockRedis,
	}

	cfg := config.Config{
		Server: config.ServerConfig{
			UploadMaxSizeMB:      100,
			ChunkSizeMB:          10,
			MaxConcurrentUploads: 5,
		},
	}

	// Create a simple test handler with embedded UploadHandler
	baseHandler := &UploadHandler{
		dbManager: nil, // We won't use the real manager
		config:    cfg,
		uploads:   sync.Map{},
	}

	// For testing purposes, we'll return the base handler but the tests will use
	// the mock manager directly
	return baseHandler, dbManager
}


func setupTestApp(handler *UploadHandler) *fiber.App {
	app := fiber.New()

	// Use simplified test handlers for now
	app.Post("/upload/initiate", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})
	app.Post("/upload/chunk", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})
	app.Post("/upload/:sessionId/complete", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})
	app.Delete("/upload/:sessionId/abort", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})
	app.Get("/upload/:sessionId/progress", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})

	return app
}

func TestInitiateUpload_Success(t *testing.T) {
	handler, mocks := setupTestHandler()
	app := setupTestApp(handler)

	// Setup mocks
	mocks.MinIO.On("InitiateMultipartUpload", mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("string"), "application/pdf", mock.AnythingOfType("map[string]string")).
		Return("mock-upload-id", nil)

	mocks.Redis.On("Set", mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), 24*time.Hour).
		Return(nil)

	// Prepare request
	req := InitiateUploadRequest{
		FileName:    "test-document.pdf",
		FileSize:    50 * 1024 * 1024, // 50MB
		ContentType: "application/pdf",
		Metadata:    map[string]string{"author": "test"},
	}

	reqBody, _ := json.Marshal(req)

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/initiate", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["session_id"])
	assert.Equal(t, "mock-upload-id", response["upload_id"])
	assert.NotEmpty(t, response["chunk_size"])
	assert.NotEmpty(t, response["total_chunks"])
	assert.Contains(t, response["message"], "successfully")

	// Verify mocks were called
	mocks.MinIO.AssertExpectations(t)
	mocks.Redis.AssertExpectations(t)
}

func TestInitiateUpload_FileSizeExceedsLimit(t *testing.T) {
	handler, _ := setupTestHandler()
	app := setupTestApp(handler)

	// Prepare request with file size exceeding limit
	req := InitiateUploadRequest{
		FileName:    "large-document.pdf",
		FileSize:    200 * 1024 * 1024, // 200MB (exceeds 100MB limit)
		ContentType: "application/pdf",
	}

	reqBody, _ := json.Marshal(req)

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/initiate", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "exceeds maximum allowed size")
}

func TestInitiateUpload_InvalidFileType(t *testing.T) {
	handler, _ := setupTestHandler()
	app := setupTestApp(handler)

	// Prepare request with non-PDF file
	req := InitiateUploadRequest{
		FileName:    "document.txt",
		FileSize:    10 * 1024 * 1024,
		ContentType: "text/plain",
	}

	reqBody, _ := json.Marshal(req)

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/initiate", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Only PDF files are supported")
}

func TestInitiateUpload_MinIOFailure(t *testing.T) {
	handler, mocks := setupTestHandler()
	app := setupTestApp(handler)

	// Setup mocks to fail
	mocks.MinIO.On("InitiateMultipartUpload", mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("string"), "application/pdf", mock.AnythingOfType("map[string]string")).
		Return("", fmt.Errorf("MinIO connection failed"))

	// Prepare request
	req := InitiateUploadRequest{
		FileName:    "test-document.pdf",
		FileSize:    50 * 1024 * 1024,
		ContentType: "application/pdf",
	}

	reqBody, _ := json.Marshal(req)

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/initiate", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to initiate upload session")

	// Verify mocks were called
	mocks.MinIO.AssertExpectations(t)
}

func TestUploadChunk_Success(t *testing.T) {
	handler, mocks := setupTestHandler()
	app := setupTestApp(handler)

	// Create a test upload session
	session := &UploadSession{
		ID:          "test-session-id",
		FileName:    "test.pdf",
		FileSize:    1024,
		ContentType: "application/pdf",
		ChunkSize:   512,
		TotalChunks: 2,
		UploadID:    "test-upload-id",
		MinIOKey:    "uploads/test-session-id/test.pdf",
		Chunks:      make(map[int]*database.ChunkUploadInfo),
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	handler.uploads.Store("test-session-id", session)

	// Setup mocks
	expectedChunkInfo := &database.ChunkUploadInfo{
		UploadID:   "test-upload-id",
		Key:        "uploads/test-session-id/test.pdf",
		ChunkIndex: 1,
		ETag:       "mock-etag",
	}

	mocks.MinIO.On("UploadChunk", mock.AnythingOfType("*context.timerCtx"),
		"test-upload-id", "uploads/test-session-id/test.pdf", 1,
		mock.AnythingOfType("*bytes.Reader"), mock.AnythingOfType("int64")).
		Return(expectedChunkInfo, nil)

	mocks.Redis.On("Set", mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), 24*time.Hour).
		Return(nil)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("session_id", "test-session-id")
	writer.WriteField("chunk_index", "1")

	part, _ := writer.CreateFormFile("chunk", "chunk1.bin")
	part.Write([]byte("test chunk data"))
	writer.Close()

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/chunk", body)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "test-session-id", response["session_id"])
	assert.Equal(t, float64(1), response["chunk_index"])
	assert.Equal(t, "mock-etag", response["chunk_etag"])
	assert.Equal(t, float64(1), response["uploaded_chunks"])
	assert.Equal(t, float64(2), response["total_chunks"])

	// Verify mocks were called
	mocks.MinIO.AssertExpectations(t)
	mocks.Redis.AssertExpectations(t)
}

func TestUploadChunk_InvalidChunkIndex(t *testing.T) {
	handler, _ := setupTestHandler()
	app := setupTestApp(handler)

	// Create a test upload session
	session := &UploadSession{
		ID:          "test-session-id",
		TotalChunks: 2,
		Status:      "active",
	}
	handler.uploads.Store("test-session-id", session)

	tests := []struct {
		name       string
		chunkIndex string
		expectMsg  string
	}{
		{
			name:       "Chunk index too low",
			chunkIndex: "0",
			expectMsg:  "Invalid chunk index",
		},
		{
			name:       "Chunk index too high",
			chunkIndex: "3",
			expectMsg:  "Invalid chunk index",
		},
		{
			name:       "Invalid chunk index format",
			chunkIndex: "invalid",
			expectMsg:  "Invalid chunk_index",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			writer.WriteField("session_id", "test-session-id")
			writer.WriteField("chunk_index", tt.chunkIndex)

			part, _ := writer.CreateFormFile("chunk", "chunk.bin")
			part.Write([]byte("test chunk data"))
			writer.Close()

			// Make request
			httpReq, _ := http.NewRequest("POST", "/upload/chunk", body)
			httpReq.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := app.Test(httpReq)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Assert response
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			assert.Contains(t, response["error"], tt.expectMsg)
		})
	}
}

func TestUploadChunk_SessionNotFound(t *testing.T) {
	handler, _ := setupTestHandler()
	app := setupTestApp(handler)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("session_id", "non-existent-session")
	writer.WriteField("chunk_index", "1")

	part, _ := writer.CreateFormFile("chunk", "chunk.bin")
	part.Write([]byte("test chunk data"))
	writer.Close()

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/chunk", body)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Upload session not found")
}

func TestCompleteUpload_Success(t *testing.T) {
	handler, mocks := setupTestHandler()
	app := setupTestApp(handler)

	// Create a test upload session with completed chunks
	chunks := map[int]*database.ChunkUploadInfo{
		1: {UploadID: "test-upload-id", Key: "test-key", ChunkIndex: 1, ETag: "etag1"},
		2: {UploadID: "test-upload-id", Key: "test-key", ChunkIndex: 2, ETag: "etag2"},
	}

	session := &UploadSession{
		ID:          "test-session-id",
		FileName:    "test.pdf",
		FileSize:    1024,
		TotalChunks: 2,
		UploadID:    "test-upload-id",
		MinIOKey:    "uploads/test-session-id/test.pdf",
		Chunks:      chunks,
		Status:      "active",
	}
	handler.uploads.Store("test-session-id", session)

	// Setup mocks
	expectedFileInfo := &database.FileInfo{
		Key:  "uploads/test-session-id/test.pdf",
		Size: 1024,
		ETag: "final-etag",
	}

	mocks.MinIO.On("CompleteMultipartUpload", mock.AnythingOfType("*context.timerCtx"),
		"test-upload-id", "uploads/test-session-id/test.pdf",
		mock.AnythingOfType("[]database.ChunkUploadInfo")).
		Return(expectedFileInfo, nil)

	mocks.Redis.On("Delete", mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("string")).
		Return(nil)

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/test-session-id/complete", nil)

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "test-session-id", response["session_id"])
	assert.NotNil(t, response["file_info"])
	assert.Contains(t, response["message"], "successfully")

	// Verify session status was updated
	sessionInterface, exists := handler.uploads.Load("test-session-id")
	require.True(t, exists)
	updatedSession := sessionInterface.(*UploadSession)
	assert.Equal(t, "completed", updatedSession.Status)

	// Verify mocks were called
	mocks.MinIO.AssertExpectations(t)
	mocks.Redis.AssertExpectations(t)
}

func TestCompleteUpload_MissingChunks(t *testing.T) {
	handler, _ := setupTestHandler()
	app := setupTestApp(handler)

	// Create a test upload session with missing chunks
	chunks := map[int]*database.ChunkUploadInfo{
		1: {UploadID: "test-upload-id", Key: "test-key", ChunkIndex: 1, ETag: "etag1"},
		// Missing chunk 2
	}

	session := &UploadSession{
		ID:          "test-session-id",
		TotalChunks: 2,
		Chunks:      chunks,
		Status:      "active",
	}
	handler.uploads.Store("test-session-id", session)

	// Make request
	httpReq, _ := http.NewRequest("POST", "/upload/test-session-id/complete", nil)

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Not all chunks uploaded")
}

func TestAbortUpload_Success(t *testing.T) {
	handler, mocks := setupTestHandler()
	app := setupTestApp(handler)

	// Create a test upload session
	session := &UploadSession{
		ID:       "test-session-id",
		UploadID: "test-upload-id",
		MinIOKey: "uploads/test-session-id/test.pdf",
		Status:   "active",
	}
	handler.uploads.Store("test-session-id", session)

	// Setup mocks
	mocks.MinIO.On("AbortMultipartUpload", mock.AnythingOfType("*context.timerCtx"),
		"test-upload-id", "uploads/test-session-id/test.pdf").
		Return(nil)

	mocks.Redis.On("Delete", mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("string")).
		Return(nil)

	// Make request
	httpReq, _ := http.NewRequest("DELETE", "/upload/test-session-id/abort", nil)

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "test-session-id", response["session_id"])
	assert.Contains(t, response["message"], "aborted successfully")

	// Verify session status was updated
	sessionInterface, exists := handler.uploads.Load("test-session-id")
	require.True(t, exists)
	updatedSession := sessionInterface.(*UploadSession)
	assert.Equal(t, "aborted", updatedSession.Status)

	// Verify mocks were called
	mocks.MinIO.AssertExpectations(t)
	mocks.Redis.AssertExpectations(t)
}

func TestGetUploadProgress_Success(t *testing.T) {
	handler, _ := setupTestHandler()
	app := setupTestApp(handler)

	// Create a test upload session with some uploaded chunks
	chunks := map[int]*database.ChunkUploadInfo{
		1: {UploadID: "test-upload-id", Key: "test-key", ChunkIndex: 1, ETag: "etag1"},
		2: {UploadID: "test-upload-id", Key: "test-key", ChunkIndex: 2, ETag: "etag2"},
	}

	session := &UploadSession{
		ID:          "test-session-id",
		FileName:    "test.pdf",
		FileSize:    2048,
		ChunkSize:   512,
		TotalChunks: 4,
		Chunks:      chunks,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	handler.uploads.Store("test-session-id", session)

	// Make request
	httpReq, _ := http.NewRequest("GET", "/upload/test-session-id/progress", nil)

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var progress UploadProgress
	err = json.NewDecoder(resp.Body).Decode(&progress)
	require.NoError(t, err)

	assert.Equal(t, "test-session-id", progress.SessionID)
	assert.Equal(t, "test.pdf", progress.FileName)
	assert.Equal(t, int64(2048), progress.FileSize)
	assert.Equal(t, 2, progress.CompletedChunks)
	assert.Equal(t, 4, progress.TotalChunks)
	assert.Equal(t, float64(50), progress.Progress) // 2/4 * 100
	assert.Equal(t, "active", progress.Status)
	assert.Equal(t, int64(1024), progress.UploadedBytes) // 2 chunks * 512 bytes
}

func TestUploadHandler_ConcurrentChunkUploads(t *testing.T) {
	handler, mocks := setupTestHandler()
	app := setupTestApp(handler)

	// Create a test upload session
	session := &UploadSession{
		ID:          "test-session-id",
		FileName:    "test.pdf",
		FileSize:    2048,
		ContentType: "application/pdf",
		ChunkSize:   512,
		TotalChunks: 4,
		UploadID:    "test-upload-id",
		MinIOKey:    "uploads/test-session-id/test.pdf",
		Chunks:      make(map[int]*database.ChunkUploadInfo),
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	handler.uploads.Store("test-session-id", session)

	// Setup mocks for multiple chunks
	for i := 1; i <= 4; i++ {
		expectedChunkInfo := &database.ChunkUploadInfo{
			UploadID:   "test-upload-id",
			Key:        "uploads/test-session-id/test.pdf",
			ChunkIndex: i,
			ETag:       fmt.Sprintf("mock-etag-%d", i),
		}

		mocks.MinIO.On("UploadChunk", mock.AnythingOfType("*context.timerCtx"),
			"test-upload-id", "uploads/test-session-id/test.pdf", i,
			mock.AnythingOfType("*bytes.Reader"), mock.AnythingOfType("int64")).
			Return(expectedChunkInfo, nil)
	}

	// Allow multiple Redis Set calls
	mocks.Redis.On("Set", mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), 24*time.Hour).
		Return(nil).Times(4)

	// Upload chunks concurrently
	done := make(chan bool, 4)

	for i := 1; i <= 4; i++ {
		go func(chunkIndex int) {
			defer func() { done <- true }()

			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			writer.WriteField("session_id", "test-session-id")
			writer.WriteField("chunk_index", strconv.Itoa(chunkIndex))

			part, _ := writer.CreateFormFile("chunk", fmt.Sprintf("chunk%d.bin", chunkIndex))
			part.Write([]byte(fmt.Sprintf("test chunk data %d", chunkIndex)))
			writer.Close()

			// Make request
			httpReq, _ := http.NewRequest("POST", "/upload/chunk", body)
			httpReq.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := app.Test(httpReq)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		}(i)
	}

	// Wait for all uploads to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify all chunks were uploaded
	sessionInterface, exists := handler.uploads.Load("test-session-id")
	require.True(t, exists)
	updatedSession := sessionInterface.(*UploadSession)
	assert.Equal(t, 4, len(updatedSession.Chunks))

	// Verify all chunk indices are present
	for i := 1; i <= 4; i++ {
		chunk, exists := updatedSession.Chunks[i]
		assert.True(t, exists, "Chunk %d should exist", i)
		assert.Equal(t, i, chunk.ChunkIndex)
	}

	// Verify mocks were called
	mocks.MinIO.AssertExpectations(t)
	mocks.Redis.AssertExpectations(t)
}

func TestUploadHandler_InvalidRequest_MissingParameters(t *testing.T) {
	handler, _ := setupTestHandler()
	app := setupTestApp(handler)

	tests := []struct {
		name           string
		sessionID      string
		chunkIndex     string
		includeFile    bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing session_id",
			sessionID:      "",
			chunkIndex:     "1",
			includeFile:    true,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "session_id and chunk_index are required",
		},
		{
			name:           "Missing chunk_index",
			sessionID:      "test-session",
			chunkIndex:     "",
			includeFile:    true,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "session_id and chunk_index are required",
		},
		{
			name:           "Missing file",
			sessionID:      "test-session",
			chunkIndex:     "1",
			includeFile:    false,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "No chunk file provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			if tt.sessionID != "" {
				writer.WriteField("session_id", tt.sessionID)
			}
			if tt.chunkIndex != "" {
				writer.WriteField("chunk_index", tt.chunkIndex)
			}
			if tt.includeFile {
				part, _ := writer.CreateFormFile("chunk", "chunk.bin")
				part.Write([]byte("test chunk data"))
			}
			writer.Close()

			// Make request
			httpReq, _ := http.NewRequest("POST", "/upload/chunk", body)
			httpReq.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := app.Test(httpReq)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Assert response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			assert.Contains(t, response["error"], tt.expectedError)
		})
	}
}
