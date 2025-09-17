package services

import (
	"testing"
)

func TestNLPService_ExtractWords(t *testing.T) {
	nlpService := NewNLPService()
	
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Basic word extraction",
			input:    "The quick brown fox jumps over the lazy dog",
			expected: []string{"quick", "brown", "fox", "jumps", "over", "lazy", "dog"},
		},
		{
			name:     "Remove stop words",
			input:    "This is a test with some stop words and normal words",
			expected: []string{"test", "stop", "words", "normal", "words"},
		},
		{
			name:     "Filter short words",
			input:    "A big elephant in zoo",
			expected: []string{"big", "elephant", "zoo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nlpService.extractWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("extractWords() length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, word := range result {
				if word != tt.expected[i] {
					t.Errorf("extractWords()[%d] = %q, want %q", i, word, tt.expected[i])
				}
			}
		})
	}
}

func TestNLPService_IsLikelyNoun(t *testing.T) {
	nlpService := NewNLPService()
	
	tests := []struct {
		name     string
		word     string
		expected bool
	}{
		{
			name:     "Likely noun with suffix",
			word:     "information",
			expected: true,
		},
		{
			name:     "Verb with -ing suffix",
			word:     "running",
			expected: false,
		},
		{
			name:     "Adverb with -ly suffix",
			word:     "quickly",
			expected: false,
		},
		{
			name:     "Short word",
			word:     "is",
			expected: false,
		},
		{
			name:     "Stop word",
			word:     "the",
			expected: false,
		},
		{
			name:     "Long word likely noun",
			word:     "elephant",
			expected: true,
		},
		{
			name:     "Proper noun (capitalized)",
			word:     "John",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nlpService.isLikelyNoun(tt.word)
			if result != tt.expected {
				t.Errorf("isLikelyNoun(%q) = %v, want %v", tt.word, result, tt.expected)
			}
		})
	}
}

func TestNLPService_ProcessText(t *testing.T) {
	nlpService := NewNLPService()
	
	input := "The elephant is a large mammal. Elephants live in Africa and Asia. These animals are very intelligent creatures."
	
	result := nlpService.ProcessText(input)
	
	if result == nil {
		t.Fatal("ProcessText() returned nil")
	}
	
	if len(result.Sentences) == 0 {
		t.Error("ProcessText() should return sentences")
	}
	
	if len(result.Nouns) == 0 {
		t.Error("ProcessText() should return nouns")
	}
	
	if result.WordCount == 0 {
		t.Error("ProcessText() should count words")
	}
	
	// Check if 'elephant' appears in nouns (it should be extracted as a noun)
	foundElephant := false
	for _, noun := range result.Nouns {
		if noun.Word == "elephant" || noun.Word == "elephants" {
			foundElephant = true
			if noun.Frequency == 0 {
				t.Error("Noun frequency should be greater than 0")
			}
			break
		}
	}
	
	if !foundElephant {
		t.Error("Should extract 'elephant' as a noun")
	}
}

func TestNLPService_BuildCoOccurrenceMatrix(t *testing.T) {
	nlpService := NewNLPService()
	
	nouns := []NounEntity{
		{Word: "elephant", Frequency: 2},
		{Word: "mammal", Frequency: 1},
		{Word: "animal", Frequency: 1},
	}
	
	sentences := []string{
		"The elephant is a large mammal",
		"The animal lives in forests",
	}
	
	matrix := nlpService.BuildCoOccurrenceMatrix(nouns, sentences, 3)
	
	if matrix == nil {
		t.Fatal("BuildCoOccurrenceMatrix() returned nil")
	}
	
	// Check if matrix contains expected words
	if _, exists := matrix["elephant"]; !exists {
		t.Error("Matrix should contain 'elephant'")
	}
	
	if _, exists := matrix["mammal"]; !exists {
		t.Error("Matrix should contain 'mammal'")
	}
}