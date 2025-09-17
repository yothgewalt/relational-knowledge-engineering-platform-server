package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
)

// MockMinIOClient for PDF service testing
type MockMinIOClientForPDF struct {
	mock.Mock
	files map[string][]byte
}

func NewMockMinIOClientForPDF() *MockMinIOClientForPDF {
	return &MockMinIOClientForPDF{
		files: make(map[string][]byte),
	}
}

func (m *MockMinIOClientForPDF) GetFile(ctx context.Context, key string) (io.ReadCloser, *database.FileInfo, error) {
	args := m.Called(ctx, key)
	
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	
	return args.Get(0).(io.ReadCloser), args.Get(1).(*database.FileInfo), args.Error(2)
}

// Add file to mock for testing
func (m *MockMinIOClientForPDF) AddFile(key string, data []byte) {
	m.files[key] = data
}

// Generate a minimal valid PDF for testing
func generateTestPDF() []byte {
	// This is a simple but complete PDF structure
	pdfContent := []byte{
		0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34, 0x0A, 0x25, 0xE2, 0xE3, 0xCF, 0xD3, 0x0A, 
		0x31, 0x20, 0x30, 0x20, 0x6F, 0x62, 0x6A, 0x0A, 0x3C, 0x3C, 0x0A, 0x2F, 0x54, 0x79, 0x70, 
		0x65, 0x20, 0x2F, 0x43, 0x61, 0x74, 0x61, 0x6C, 0x6F, 0x67, 0x0A, 0x2F, 0x50, 0x61, 0x67, 
		0x65, 0x73, 0x20, 0x32, 0x20, 0x30, 0x20, 0x52, 0x0A, 0x3E, 0x3E, 0x0A, 0x65, 0x6E, 0x64, 
		0x6F, 0x62, 0x6A, 0x0A, 0x32, 0x20, 0x30, 0x20, 0x6F, 0x62, 0x6A, 0x0A, 0x3C, 0x3C, 0x0A, 
		0x2F, 0x54, 0x79, 0x70, 0x65, 0x20, 0x2F, 0x50, 0x61, 0x67, 0x65, 0x73, 0x0A, 0x2F, 0x4B, 
		0x69, 0x64, 0x73, 0x20, 0x5B, 0x33, 0x20, 0x30, 0x20, 0x52, 0x5D, 0x0A, 0x2F, 0x43, 0x6F, 
		0x75, 0x6E, 0x74, 0x20, 0x31, 0x0A, 0x3E, 0x3E, 0x0A, 0x65, 0x6E, 0x64, 0x6F, 0x62, 0x6A, 
		0x0A, 0x33, 0x20, 0x30, 0x20, 0x6F, 0x62, 0x6A, 0x0A, 0x3C, 0x3C, 0x0A, 0x2F, 0x54, 0x79, 
		0x70, 0x65, 0x20, 0x2F, 0x50, 0x61, 0x67, 0x65, 0x0A, 0x2F, 0x50, 0x61, 0x72, 0x65, 0x6E, 
		0x74, 0x20, 0x32, 0x20, 0x30, 0x20, 0x52, 0x0A, 0x2F, 0x4D, 0x65, 0x64, 0x69, 0x61, 0x42, 
		0x6F, 0x78, 0x20, 0x5B, 0x30, 0x20, 0x30, 0x20, 0x36, 0x31, 0x32, 0x20, 0x37, 0x39, 0x32, 
		0x5D, 0x0A, 0x2F, 0x43, 0x6F, 0x6E, 0x74, 0x65, 0x6E, 0x74, 0x73, 0x20, 0x34, 0x20, 0x30, 
		0x20, 0x52, 0x0A, 0x2F, 0x52, 0x65, 0x73, 0x6F, 0x75, 0x72, 0x63, 0x65, 0x73, 0x20, 0x3C, 
		0x3C, 0x2F, 0x46, 0x6F, 0x6E, 0x74, 0x20, 0x3C, 0x3C, 0x2F, 0x46, 0x31, 0x20, 0x35, 0x20, 
		0x30, 0x20, 0x52, 0x3E, 0x3E, 0x3E, 0x3E, 0x0A, 0x3E, 0x3E, 0x0A, 0x65, 0x6E, 0x64, 0x6F, 
		0x62, 0x6A, 0x0A, 0x34, 0x20, 0x30, 0x20, 0x6F, 0x62, 0x6A, 0x0A, 0x3C, 0x3C, 0x2F, 0x4C, 
		0x65, 0x6E, 0x67, 0x74, 0x68, 0x20, 0x34, 0x34, 0x3E, 0x3E, 0x0A, 0x73, 0x74, 0x72, 0x65, 
		0x61, 0x6D, 0x0A, 0x42, 0x54, 0x0A, 0x2F, 0x46, 0x31, 0x20, 0x31, 0x32, 0x20, 0x54, 0x66, 
		0x0A, 0x31, 0x30, 0x30, 0x20, 0x37, 0x30, 0x30, 0x20, 0x54, 0x64, 0x0A, 0x28, 0x48, 0x65, 
		0x6C, 0x6C, 0x6F, 0x20, 0x57, 0x6F, 0x72, 0x6C, 0x64, 0x29, 0x20, 0x54, 0x6A, 0x0A, 0x45, 
		0x54, 0x0A, 0x65, 0x6E, 0x64, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6D, 0x0A, 0x65, 0x6E, 0x64, 
		0x6F, 0x62, 0x6A, 0x0A, 0x35, 0x20, 0x30, 0x20, 0x6F, 0x62, 0x6A, 0x0A, 0x3C, 0x3C, 0x2F, 
		0x54, 0x79, 0x70, 0x65, 0x2F, 0x46, 0x6F, 0x6E, 0x74, 0x2F, 0x53, 0x75, 0x62, 0x74, 0x79, 
		0x70, 0x65, 0x2F, 0x54, 0x79, 0x70, 0x65, 0x31, 0x2F, 0x42, 0x61, 0x73, 0x65, 0x46, 0x6F, 
		0x6E, 0x74, 0x2F, 0x48, 0x65, 0x6C, 0x76, 0x65, 0x74, 0x69, 0x63, 0x61, 0x3E, 0x3E, 0x0A, 
		0x65, 0x6E, 0x64, 0x6F, 0x62, 0x6A, 0x0A, 0x78, 0x72, 0x65, 0x66, 0x0A, 0x30, 0x20, 0x36, 
		0x0A, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x20, 0x36, 0x35, 0x35, 
		0x33, 0x35, 0x20, 0x66, 0x20, 0x0A, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31, 0x30, 
		0x20, 0x30, 0x30, 0x30, 0x30, 0x30, 0x20, 0x6E, 0x20, 0x0A, 0x30, 0x30, 0x30, 0x30, 0x30, 
		0x30, 0x30, 0x37, 0x39, 0x20, 0x30, 0x30, 0x30, 0x30, 0x30, 0x20, 0x6E, 0x20, 0x0A, 0x30, 
		0x30, 0x30, 0x30, 0x30, 0x30, 0x31, 0x37, 0x33, 0x20, 0x30, 0x30, 0x30, 0x30, 0x30, 0x20, 
		0x6E, 0x20, 0x0A, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x33, 0x30, 0x31, 0x20, 0x30, 0x30, 
		0x30, 0x30, 0x30, 0x20, 0x6E, 0x20, 0x0A, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x33, 0x38, 
		0x30, 0x20, 0x30, 0x30, 0x30, 0x30, 0x30, 0x20, 0x6E, 0x20, 0x0A, 0x74, 0x72, 0x61, 0x69, 
		0x6C, 0x65, 0x72, 0x0A, 0x3C, 0x3C, 0x2F, 0x53, 0x69, 0x7A, 0x65, 0x20, 0x36, 0x2F, 0x52, 
		0x6F, 0x6F, 0x74, 0x20, 0x31, 0x20, 0x30, 0x20, 0x52, 0x3E, 0x3E, 0x0A, 0x73, 0x74, 0x61, 
		0x72, 0x74, 0x78, 0x72, 0x65, 0x66, 0x0A, 0x34, 0x32, 0x39, 0x0A, 0x25, 0x25, 0x45, 0x4F, 
		0x46, 0x0A,
	}
	return pdfContent
}

// Generate corrupted PDF data
func generateCorruptedPDF() []byte {
	return []byte("This is not a PDF file content")
}

func TestPDFService_ExtractTextFromMinIO_Success(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	// Prepare test data
	key := "test/document.pdf"
	pdfData := generateTestPDF()
	
	fileInfo := &database.FileInfo{
		Key:          key,
		Size:         int64(len(pdfData)),
		ContentType:  "application/pdf",
		LastModified: time.Now().Format("2006-01-02T15:04:05Z"),
		ETag:         "test-etag",
	}

	// Setup mock
	reader := io.NopCloser(bytes.NewReader(pdfData))
	mockClient.On("GetFile", ctx, key).Return(reader, fileInfo, nil)

	// Execute test
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// The test PDF should contain "Hello World"
	assert.Contains(t, result.Content, "Hello World")
	assert.Equal(t, 1, result.PageCount)
	assert.Len(t, result.Pages, 1)
	assert.Contains(t, result.Pages[0], "Hello World")

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestPDFService_ExtractTextFromMinIO_MinIOError(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	key := "test/nonexistent.pdf"
	
	// Setup mock to return error
	mockClient.On("GetFile", ctx, key).Return(nil, nil, fmt.Errorf("file not found"))

	// Execute test
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get file from MinIO")

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestPDFService_ExtractTextFromMinIO_ReadError(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	key := "test/document.pdf"
	fileInfo := &database.FileInfo{
		Key:  key,
		Size: 1024,
	}

	// Create a reader that will fail on read
	failingReader := &FailingReader{}
	mockClient.On("GetFile", ctx, key).Return(failingReader, fileInfo, nil)

	// Execute test
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to read file data")

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestPDFService_ExtractTextFromMinIO_CorruptedPDF(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	key := "test/corrupted.pdf"
	corruptedData := generateCorruptedPDF()
	
	fileInfo := &database.FileInfo{
		Key:         key,
		Size:        int64(len(corruptedData)),
		ContentType: "application/pdf",
	}

	// Setup mock
	reader := io.NopCloser(bytes.NewReader(corruptedData))
	mockClient.On("GetFile", ctx, key).Return(reader, fileInfo, nil)

	// Execute test
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create PDF reader")

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestPDFService_ExtractTextFromMinIO_EmptyPDF(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	key := "test/empty.pdf"
	emptyData := []byte{}
	
	fileInfo := &database.FileInfo{
		Key:         key,
		Size:        0,
		ContentType: "application/pdf",
	}

	// Setup mock
	reader := io.NopCloser(bytes.NewReader(emptyData))
	mockClient.On("GetFile", ctx, key).Return(reader, fileInfo, nil)

	// Execute test
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestPDFService_ExtractTextFromMinIO_LargePDF(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	// Generate a larger test PDF with multiple pages
	key := "test/large-document.pdf"
	
	// Create a PDF with multiple pages (simulated)
	pdfContent := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj

2 0 obj
<<
/Type /Pages
/Kids [3 0 R 4 0 R]
/Count 2
>>
endobj

3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 5 0 R
/Resources <<
  /Font <<
    /F1 7 0 R
  >>
>>
>>
endobj

4 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 6 0 R
/Resources <<
  /Font <<
    /F1 7 0 R
  >>
>>
>>
endobj

5 0 obj
<<
/Length 50
>>
stream
BT
/F1 12 Tf
100 700 Td
(Page 1 Content) Tj
ET
endstream
endobj

6 0 obj
<<
/Length 50
>>
stream
BT
/F1 12 Tf
100 700 Td
(Page 2 Content) Tj
ET
endstream
endobj

7 0 obj
<<
/Type /Font
/Subtype /Type1
/BaseFont /Helvetica
>>
endobj

xref
0 8
0000000000 65535 f 
0000000010 00000 n 
0000000079 00000 n 
0000000145 00000 n 
0000000310 00000 n 
0000000475 00000 n 
0000000576 00000 n 
0000000677 00000 n 
trailer
<<
/Size 8
/Root 1 0 R
>>
startxref
747
%%EOF`
	
	pdfData := []byte(pdfContent)
	
	fileInfo := &database.FileInfo{
		Key:          key,
		Size:         int64(len(pdfData)),
		ContentType:  "application/pdf",
		LastModified: time.Now().Format("2006-01-02T15:04:05Z"),
		ETag:         "test-etag-large",
	}

	// Setup mock
	reader := io.NopCloser(bytes.NewReader(pdfData))
	mockClient.On("GetFile", ctx, key).Return(reader, fileInfo, nil)

	// Execute test
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// Should contain content from both pages
	assert.Contains(t, result.Content, "Page 1 Content")
	assert.Contains(t, result.Content, "Page 2 Content")
	assert.Equal(t, 2, result.PageCount)
	assert.Len(t, result.Pages, 2)

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestPDFService_Integration_WithTextProcessing(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	// Create PDF with text that needs cleaning
	key := "test/messy-document.pdf"
	
	// This PDF content simulates text with extra whitespace and formatting
	messyPDFContent := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj

2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj

3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 4 0 R
/Resources <<
  /Font <<
    /F1 5 0 R
  >>
>>
>>
endobj

4 0 obj
<<
/Length 88
>>
stream
BT
/F1 12 Tf
100 700 Td
(This is a test sentence.) Tj
100 680 Td
(This is another sentence for testing.) Tj
ET
endstream
endobj

5 0 obj
<<
/Type /Font
/Subtype /Type1
/BaseFont /Helvetica
>>
endobj

xref
0 6
0000000000 65535 f 
0000000010 00000 n 
0000000079 00000 n 
0000000136 00000 n 
0000000301 00000 n 
0000000440 00000 n 
trailer
<<
/Size 6
/Root 1 0 R
>>
startxref
510
%%EOF`
	
	pdfData := []byte(messyPDFContent)
	
	fileInfo := &database.FileInfo{
		Key:         key,
		Size:        int64(len(pdfData)),
		ContentType: "application/pdf",
	}

	// Setup mock
	reader := io.NopCloser(bytes.NewReader(pdfData))
	mockClient.On("GetFile", ctx, key).Return(reader, fileInfo, nil)

	// Execute extraction
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test text cleaning
	cleanedText := pdfService.CleanText(result.Content)
	assert.NotEmpty(t, cleanedText)
	assert.Contains(t, cleanedText, "This is a test sentence")
	assert.Contains(t, cleanedText, "This is another sentence")

	// Test sentence splitting
	sentences := pdfService.SplitIntoSentences(result.Content)
	assert.Greater(t, len(sentences), 0)
	
	// Should have at least 2 sentences
	assert.GreaterOrEqual(t, len(sentences), 2)
	
	// Check that sentences are properly cleaned (no empty or very short sentences)
	for _, sentence := range sentences {
		assert.Greater(t, len(sentence), 10)
	}

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestBytesReaderAt_ReadAt(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		offset         int64
		bufferSize     int
		expectedN      int
		expectedErr    error
		expectedData   []byte
	}{
		{
			name:         "Read from beginning",
			data:         []byte("Hello, World!"),
			offset:       0,
			bufferSize:   5,
			expectedN:    5,
			expectedErr:  nil,
			expectedData: []byte("Hello"),
		},
		{
			name:         "Read from middle",
			data:         []byte("Hello, World!"),
			offset:       7,
			bufferSize:   5,
			expectedN:    5,
			expectedErr:  nil,
			expectedData: []byte("World"),
		},
		{
			name:         "Read beyond end",
			data:         []byte("Hello"),
			offset:       10,
			bufferSize:   5,
			expectedN:    0,
			expectedErr:  io.EOF,
			expectedData: nil,
		},
		{
			name:         "Read partial at end",
			data:         []byte("Hello, World!"),
			offset:       10,
			bufferSize:   5,
			expectedN:    3,
			expectedErr:  io.EOF,
			expectedData: []byte("ld!"),
		},
		{
			name:         "Read empty data",
			data:         []byte{},
			offset:       0,
			bufferSize:   5,
			expectedN:    0,
			expectedErr:  io.EOF,
			expectedData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &bytesReaderAt{data: tt.data}
			buffer := make([]byte, tt.bufferSize)
			
			n, err := reader.ReadAt(buffer, tt.offset)
			
			assert.Equal(t, tt.expectedN, n)
			assert.Equal(t, tt.expectedErr, err)
			
			if tt.expectedData != nil {
				assert.Equal(t, tt.expectedData, buffer[:n])
			}
		})
	}
}

func TestPDFService_ErrorHandling_ContextCancellation(t *testing.T) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	key := "test/document.pdf"
	
	// Setup mock to return context error
	mockClient.On("GetFile", ctx, key).Return(nil, nil, context.Canceled)

	// Execute test
	result, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get file from MinIO")

	// Verify mock was called
	mockClient.AssertExpectations(t)
}

func TestPDFService_TextProcessing_EdgeCases(t *testing.T) {
	pdfService := NewPDFService()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only whitespace",
			input:    "   \n\r\t   ",
			expected: "",
		},
		{
			name:     "Mixed line endings",
			input:    "Line1\r\nLine2\rLine3\nLine4",
			expected: "Line1\nLine2\nLine3\nLine4",
		},
		{
			name:     "Multiple empty lines",
			input:    "Line1\n\n\n\nLine2",
			expected: "Line1\nLine2",
		},
		{
			name:     "Leading and trailing whitespace",
			input:    "   Line1   \n   Line2   ",
			expected: "Line1\nLine2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pdfService.CleanText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPDFService_SentenceSplitting_EdgeCases(t *testing.T) {
	pdfService := NewPDFService()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "No sentences",
			input:    "No sentence ending here",
			expected: []string{},
		},
		{
			name:     "Short sentences filtered out",
			input:    "Hi. Ok. This is a longer sentence that should be included.",
			expected: []string{"This is a longer sentence that should be included"},
		},
		{
			name:     "Multiple sentences",
			input:    "First sentence is here. Second sentence follows. Third sentence ends.",
			expected: []string{
				"First sentence is here",
				"Second sentence follows",
				"Third sentence ends",
			},
		},
		{
			name:     "Sentences with extra spaces",
			input:    "First sentence.   Second sentence.    Third sentence.",
			expected: []string{
				"First sentence",
				"Second sentence",
				"Third sentence",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pdfService.SplitIntoSentences(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// FailingReader is a helper type that always fails on Read
type FailingReader struct{}

func (f *FailingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read failure")
}

func (f *FailingReader) Close() error {
	return nil
}

// Benchmark tests for performance verification
func BenchmarkPDFService_ExtractTextFromMinIO(b *testing.B) {
	pdfService := NewPDFService()
	mockClient := NewMockMinIOClientForPDF()
	ctx := context.Background()

	key := "test/benchmark.pdf"
	pdfData := generateTestPDF()
	
	fileInfo := &database.FileInfo{
		Key:  key,
		Size: int64(len(pdfData)),
	}

	// Setup mock for benchmark
	for i := 0; i < b.N; i++ {
		reader := io.NopCloser(bytes.NewReader(pdfData))
		mockClient.On("GetFile", ctx, key).Return(reader, fileInfo, nil).Once()
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := pdfService.ExtractTextFromMinIO(ctx, mockClient, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPDFService_CleanText(b *testing.B) {
	pdfService := NewPDFService()
	
	// Create test text with various formatting issues
	testText := `
		Line 1 with spaces    
	
	Line 2 with tabs		
	
	
	Line 3 after empty lines
	
		Line 4 indented   
	`
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = pdfService.CleanText(testText)
	}
}

func BenchmarkPDFService_SplitIntoSentences(b *testing.B) {
	pdfService := NewPDFService()
	
	testText := "This is the first sentence. This is the second sentence. This is a longer third sentence with more content. Short. This is the final sentence."
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = pdfService.SplitIntoSentences(testText)
	}
}