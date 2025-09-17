package services

import (
	"testing"
)

func TestPDFService_CleanText(t *testing.T) {
	pdfService := NewPDFService()
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic text cleaning",
			input:    "Hello World\r\nThis is a test\r\nWith multiple lines",
			expected: "Hello World\nThis is a test\nWith multiple lines",
		},
		{
			name:     "Remove empty lines",
			input:    "Line 1\n\n\nLine 2\n\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "Trim whitespace",
			input:    "  Trimmed  \n  Text  \n  Here  ",
			expected: "Trimmed\nText\nHere",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pdfService.CleanText(tt.input)
			if result != tt.expected {
				t.Errorf("CleanText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPDFService_SplitIntoSentences(t *testing.T) {
	pdfService := NewPDFService()
	
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Basic sentence splitting",
			input:    "This is the first sentence. This is the second sentence. Short.",
			expected: []string{"This is the first sentence", "This is the second sentence"},
		},
		{
			name:     "Filter short sentences",
			input:    "This is a long sentence with enough characters. Hi. This is another long sentence.",
			expected: []string{"This is a long sentence with enough characters", "This is another long sentence"},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pdfService.SplitIntoSentences(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitIntoSentences() length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, sentence := range result {
				if sentence != tt.expected[i] {
					t.Errorf("SplitIntoSentences()[%d] = %q, want %q", i, sentence, tt.expected[i])
				}
			}
		})
	}
}