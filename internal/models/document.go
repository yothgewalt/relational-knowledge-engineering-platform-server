package models

import (
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/services"
)

type Document struct {
	ID          string                 `json:"id" bson:"_id"`
	Filename    string                 `json:"filename" bson:"filename"`
	ContentType string                 `json:"content_type" bson:"content_type"`
	Size        int64                  `json:"size" bson:"size"`
	UploadedAt  time.Time             `json:"uploaded_at" bson:"uploaded_at"`
	ProcessedAt *time.Time            `json:"processed_at,omitempty" bson:"processed_at,omitempty"`
	Status      ProcessingStatus       `json:"status" bson:"status"`
	ExtractedText *services.ExtractedText `json:"extracted_text,omitempty" bson:"extracted_text,omitempty"`
	ProcessedText *services.ProcessedText `json:"processed_text,omitempty" bson:"processed_text,omitempty"`
	GraphID     *string               `json:"graph_id,omitempty" bson:"graph_id,omitempty"`
	ErrorMessage *string               `json:"error_message,omitempty" bson:"error_message,omitempty"`
}

type ProcessingStatus string

const (
	StatusUploaded   ProcessingStatus = "uploaded"
	StatusProcessing ProcessingStatus = "processing"
	StatusCompleted  ProcessingStatus = "completed"
	StatusError      ProcessingStatus = "error"
)

type DocumentRepository struct {
	// This will be implemented with actual database operations
}

type ProcessingLog struct {
	ID          string    `json:"id" bson:"_id"`
	DocumentID  string    `json:"document_id" bson:"document_id"`
	Stage       string    `json:"stage" bson:"stage"`
	Status      string    `json:"status" bson:"status"`
	Message     string    `json:"message" bson:"message"`
	StartedAt   time.Time `json:"started_at" bson:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	Duration    *int64    `json:"duration_ms,omitempty" bson:"duration_ms,omitempty"`
}