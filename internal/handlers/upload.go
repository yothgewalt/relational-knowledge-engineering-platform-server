package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
)

type UploadHandler struct {
	dbManager *database.Manager
	config    config.Config
	uploads   sync.Map // Store active uploads
}

type UploadSession struct {
	ID          string                            `json:"id"`
	FileName    string                            `json:"filename"`
	FileSize    int64                             `json:"file_size"`
	ContentType string                            `json:"content_type"`
	ChunkSize   int64                             `json:"chunk_size"`
	TotalChunks int                               `json:"total_chunks"`
	UploadID    string                            `json:"upload_id"`
	MinIOKey    string                            `json:"minio_key"`
	Chunks      map[int]*database.ChunkUploadInfo `json:"chunks"`
	CreatedAt   time.Time                         `json:"created_at"`
	UpdatedAt   time.Time                         `json:"updated_at"`
	Status      string                            `json:"status"` // "active", "completed", "failed", "aborted"
}

type InitiateUploadRequest struct {
	FileName    string            `json:"filename"`
	FileSize    int64             `json:"file_size"`
	ContentType string            `json:"content_type"`
	ChunkSize   int64             `json:"chunk_size,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type UploadChunkRequest struct {
	SessionID  string `json:"session_id"`
	ChunkIndex int    `json:"chunk_index"`
}

type UploadProgress struct {
	SessionID       string    `json:"session_id"`
	FileName        string    `json:"filename"`
	FileSize        int64     `json:"file_size"`
	UploadedBytes   int64     `json:"uploaded_bytes"`
	CompletedChunks int       `json:"completed_chunks"`
	TotalChunks     int       `json:"total_chunks"`
	Progress        float64   `json:"progress"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewUploadHandler(dbManager *database.Manager, config config.Config) *UploadHandler {
	return &UploadHandler{
		dbManager: dbManager,
		config:    config,
		uploads:   sync.Map{},
	}
}

func (h *UploadHandler) InitiateUpload(c *fiber.Ctx) error {
	var req InitiateUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate file size limits
	maxSizeMB := int64(h.config.Server.UploadMaxSizeMB)
	maxSizeBytes := maxSizeMB * 1024 * 1024
	if req.FileSize > maxSizeBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File size %d bytes exceeds maximum allowed size of %d MB", req.FileSize, maxSizeMB),
		})
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(req.FileName), ".pdf") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Only PDF files are supported",
		})
	}

	// Calculate optimal chunk size
	chunkSize := req.ChunkSize
	if chunkSize == 0 {
		chunkSize = database.CalculateOptimalChunkSize(req.FileSize, h.config.Server.ChunkSizeMB)
	}

	// Generate session and MinIO key
	sessionID := uuid.New().String()
	minioKey := fmt.Sprintf("uploads/%s/%s", sessionID, req.FileName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initiate multipart upload
	uploadID, err := h.dbManager.MinIO.InitiateMultipartUpload(ctx, minioKey, req.ContentType, req.Metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initiate upload session",
		})
	}

	// Calculate total chunks
	totalChunks := int((req.FileSize + chunkSize - 1) / chunkSize)

	// Create upload session
	session := &UploadSession{
		ID:          sessionID,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		UploadID:    uploadID,
		MinIOKey:    minioKey,
		Chunks:      make(map[int]*database.ChunkUploadInfo),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      "active",
	}

	// Store session in memory (in production, you might want to use Redis)
	h.uploads.Store(sessionID, session)

	// Cache session info in Redis for persistence
	sessionKey := fmt.Sprintf("upload_session:%s", sessionID)
	h.dbManager.Redis.Set(ctx, sessionKey, fmt.Sprintf("%v", session), 24*time.Hour)

	return c.JSON(fiber.Map{
		"session_id":   sessionID,
		"upload_id":    uploadID,
		"chunk_size":   chunkSize,
		"total_chunks": totalChunks,
		"minio_key":    minioKey,
		"message":      "Upload session initiated successfully",
	})
}

func (h *UploadHandler) UploadChunk(c *fiber.Ctx) error {
	sessionID := c.FormValue("session_id")
	chunkIndexStr := c.FormValue("chunk_index")

	if sessionID == "" || chunkIndexStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id and chunk_index are required",
		})
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid chunk_index",
		})
	}

	// Get file from form
	file, err := c.FormFile("chunk")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No chunk file provided",
		})
	}

	// Get upload session
	sessionInterface, exists := h.uploads.Load(sessionID)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Upload session not found",
		})
	}

	session := sessionInterface.(*UploadSession)
	if session.Status != "active" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Upload session is %s", session.Status),
		})
	}

	// Validate chunk index
	if chunkIndex < 1 || chunkIndex > session.TotalChunks {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid chunk index. Expected 1-%d", session.TotalChunks),
		})
	}

	// Open and read chunk data
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open chunk file",
		})
	}
	defer src.Close()

	// Read chunk data into buffer
	chunkData, err := io.ReadAll(src)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read chunk data",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Upload chunk to MinIO
	chunkInfo, err := h.dbManager.MinIO.UploadChunk(ctx, session.UploadID, session.MinIOKey, chunkIndex, bytes.NewReader(chunkData), int64(len(chunkData)))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to upload chunk",
		})
	}

	// Update session
	session.Chunks[chunkIndex] = chunkInfo
	session.UpdatedAt = time.Now()
	h.uploads.Store(sessionID, session)

	// Update cache
	sessionKey := fmt.Sprintf("upload_session:%s", sessionID)
	h.dbManager.Redis.Set(ctx, sessionKey, fmt.Sprintf("%v", session), 24*time.Hour)

	return c.JSON(fiber.Map{
		"session_id":      sessionID,
		"chunk_index":     chunkIndex,
		"chunk_etag":      chunkInfo.ETag,
		"uploaded_chunks": len(session.Chunks),
		"total_chunks":    session.TotalChunks,
		"message":         "Chunk uploaded successfully",
	})
}

func (h *UploadHandler) CompleteUpload(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id is required",
		})
	}

	// Get upload session
	sessionInterface, exists := h.uploads.Load(sessionID)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Upload session not found",
		})
	}

	session := sessionInterface.(*UploadSession)
	if session.Status != "active" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Upload session is %s", session.Status),
		})
	}

	// Verify all chunks are uploaded
	if len(session.Chunks) != session.TotalChunks {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Not all chunks uploaded. Expected %d, got %d", session.TotalChunks, len(session.Chunks)),
		})
	}

	// Create ordered chunks list
	var chunks []database.ChunkUploadInfo
	for i := 1; i <= session.TotalChunks; i++ {
		chunk, exists := session.Chunks[i]
		if !exists {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Missing chunk %d", i),
			})
		}
		chunks = append(chunks, *chunk)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Complete multipart upload
	fileInfo, err := h.dbManager.MinIO.CompleteMultipartUpload(ctx, session.UploadID, session.MinIOKey, chunks)
	if err != nil {
		session.Status = "failed"
		h.uploads.Store(sessionID, session)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to complete upload",
		})
	}

	// Update session status
	session.Status = "completed"
	session.UpdatedAt = time.Now()
	h.uploads.Store(sessionID, session)

	// Clean up Redis cache
	sessionKey := fmt.Sprintf("upload_session:%s", sessionID)
	h.dbManager.Redis.Delete(ctx, sessionKey)

	// Trigger PDF processing (asynchronous)
	go h.triggerPDFProcessing(sessionID, session.MinIOKey, fileInfo)

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"file_info":  fileInfo,
		"message":    "Upload completed successfully",
	})
}

func (h *UploadHandler) AbortUpload(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id is required",
		})
	}

	// Get upload session
	sessionInterface, exists := h.uploads.Load(sessionID)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Upload session not found",
		})
	}

	session := sessionInterface.(*UploadSession)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Abort multipart upload in MinIO
	if session.UploadID != "" {
		err := h.dbManager.MinIO.AbortMultipartUpload(ctx, session.UploadID, session.MinIOKey)
		if err != nil {
			// Log error but continue with cleanup
		}
	}

	// Update session status
	session.Status = "aborted"
	session.UpdatedAt = time.Now()
	h.uploads.Store(sessionID, session)

	// Clean up Redis cache
	sessionKey := fmt.Sprintf("upload_session:%s", sessionID)
	h.dbManager.Redis.Delete(ctx, sessionKey)

	// Remove from active uploads after a delay
	go func() {
		time.Sleep(5 * time.Minute)
		h.uploads.Delete(sessionID)
	}()

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"message":    "Upload aborted successfully",
	})
}

func (h *UploadHandler) GetUploadProgress(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id is required",
		})
	}

	// Get upload session
	sessionInterface, exists := h.uploads.Load(sessionID)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Upload session not found",
		})
	}

	session := sessionInterface.(*UploadSession)

	// Calculate progress
	completedChunks := len(session.Chunks)
	progress := float64(completedChunks) / float64(session.TotalChunks) * 100
	uploadedBytes := int64(completedChunks) * session.ChunkSize
	if uploadedBytes > session.FileSize {
		uploadedBytes = session.FileSize
	}

	return c.JSON(UploadProgress{
		SessionID:       session.ID,
		FileName:        session.FileName,
		FileSize:        session.FileSize,
		UploadedBytes:   uploadedBytes,
		CompletedChunks: completedChunks,
		TotalChunks:     session.TotalChunks,
		Progress:        progress,
		Status:          session.Status,
		CreatedAt:       session.CreatedAt,
		UpdatedAt:       session.UpdatedAt,
	})
}

func (h *UploadHandler) triggerPDFProcessing(sessionID, minioKey string, fileInfo *database.FileInfo) {
	// Extract filename from MinIO key
	parts := strings.Split(minioKey, "/")
	filename := parts[len(parts)-1]

	// Create a new DocumentHandler to process the file
	documentHandler := NewDocumentHandler(h.dbManager)

	// Use session ID as document ID for consistency
	err := documentHandler.ProcessMinIODocument(sessionID, minioKey, filename, fileInfo.Size)
	if err != nil {
		fmt.Printf("Failed to start PDF processing for session %s: %v\n", sessionID, err)
	} else {
		fmt.Printf("PDF processing started for session %s, file %s (size: %d bytes)\n",
			sessionID, minioKey, fileInfo.Size)
	}
}
