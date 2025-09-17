package services

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
)

// MinIOClientInterface defines the interface for MinIO operations needed by PDF service
type MinIOClientInterface interface {
	GetFile(ctx context.Context, key string) (io.ReadCloser, *database.FileInfo, error)
}

type PDFService struct{}

type ExtractedText struct {
	Content   string   `json:"content"`
	Pages     []string `json:"pages"`
	PageCount int      `json:"page_count"`
}

func NewPDFService() *PDFService {
	return &PDFService{}
}

func (p *PDFService) ExtractText(reader io.ReaderAt, size int64) (*ExtractedText, error) {
	pdfReader, err := pdf.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var allText strings.Builder
	var pages []string

	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText, err := p.extractPageText(page)
		if err != nil {
			continue
		}

		pages = append(pages, pageText)
		allText.WriteString(pageText)
		allText.WriteString("\n")
	}

	return &ExtractedText{
		Content:   allText.String(),
		Pages:     pages,
		PageCount: len(pages),
	}, nil
}

func (p *PDFService) extractPageText(page pdf.Page) (string, error) {
	var textBuilder strings.Builder

	content := page.Content()
	for _, text := range content.Text {
		textBuilder.WriteString(text.S)
		textBuilder.WriteString(" ")
	}

	return strings.TrimSpace(textBuilder.String()), nil
}

func (p *PDFService) CleanText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	
	lines := strings.Split(text, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	
	return strings.Join(cleanLines, "\n")
}

func (p *PDFService) SplitIntoSentences(text string) []string {
	text = p.CleanText(text)
	
	sentences := strings.Split(text, ".")
	var cleanSentences []string
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 10 {
			cleanSentences = append(cleanSentences, sentence)
		}
	}
	
	return cleanSentences
}

func (p *PDFService) ExtractTextFromMinIO(ctx context.Context, minioClient MinIOClientInterface, key string) (*ExtractedText, error) {
	// Get file from MinIO
	reader, fileInfo, err := minioClient.GetFile(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from MinIO: %w", err)
	}
	defer reader.Close()

	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// Create a ReaderAt from the data
	readerAt := &bytesReaderAt{data: data}

	// Extract text using existing method
	return p.ExtractText(readerAt, fileInfo.Size)
}

// Helper struct to implement io.ReaderAt interface
type bytesReaderAt struct {
	data []byte
}

func (r *bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	
	return n, nil
}