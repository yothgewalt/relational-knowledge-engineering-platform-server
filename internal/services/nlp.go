package services

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
)

type NLPService struct {
	stopWords map[string]bool
}

type NounEntity struct {
	Word      string  `json:"word"`
	Frequency int     `json:"frequency"`
	Positions []int   `json:"positions"`
	Stemmed   string  `json:"stemmed"`
}

type ProcessedText struct {
	Nouns     []NounEntity `json:"nouns"`
	Sentences []string     `json:"sentences"`
	WordCount int          `json:"word_count"`
}

func NewNLPService() *NLPService {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "up": true, "about": true, "into": true,
		"through": true, "during": true, "before": true, "after": true, "above": true,
		"below": true, "between": true, "among": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true, "can": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "me": true,
		"my": true, "myself": true, "we": true, "our": true, "ours": true, "ourselves": true,
		"you": true, "your": true, "yours": true, "yourself": true, "yourselves": true,
		"he": true, "him": true, "his": true, "himself": true, "she": true, "her": true,
		"hers": true, "herself": true, "it": true, "its": true, "itself": true, "they": true,
		"them": true, "their": true, "theirs": true, "themselves": true, "what": true,
		"which": true, "who": true, "whom": true, "whose": true, "where": true, "when": true,
		"why": true, "how": true, "all": true, "any": true, "both": true, "each": true,
		"few": true, "more": true, "most": true, "other": true, "some": true, "such": true,
		"no": true, "nor": true, "not": true, "only": true, "own": true, "same": true,
		"so": true, "than": true, "too": true, "very": true, "just": true, "now": true,
	}

	return &NLPService{
		stopWords: stopWords,
	}
}

func (n *NLPService) ProcessText(text string) *ProcessedText {
	sentences := n.splitIntoSentences(text)
	words := n.extractWords(text)
	nouns := n.extractNouns(words, text)

	return &ProcessedText{
		Nouns:     nouns,
		Sentences: sentences,
		WordCount: len(words),
	}
}

func (n *NLPService) splitIntoSentences(text string) []string {
	sentencePattern := regexp.MustCompile(`[.!?]+`)
	sentences := sentencePattern.Split(text, -1)
	
	var cleanSentences []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 10 {
			cleanSentences = append(cleanSentences, sentence)
		}
	}
	
	return cleanSentences
}

func (n *NLPService) extractWords(text string) []string {
	wordPattern := regexp.MustCompile(`\b[a-zA-Z]{2,}\b`)
	words := wordPattern.FindAllString(strings.ToLower(text), -1)
	
	var filteredWords []string
	for _, word := range words {
		if !n.stopWords[word] {
			filteredWords = append(filteredWords, word)
		}
	}
	
	return filteredWords
}

func (n *NLPService) extractNouns(words []string, originalText string) []NounEntity {
	wordFreq := make(map[string]int)
	wordPositions := make(map[string][]int)
	originalText = strings.ToLower(originalText)
	
	for _, word := range words {
		if n.isLikelyNoun(word) {
			wordFreq[word]++
			positions := n.findWordPositions(word, originalText)
			wordPositions[word] = positions
		}
	}
	
	var nouns []NounEntity
	for word, freq := range wordFreq {
		stemmed, err := snowball.Stem(word, "english", true)
		if err != nil {
			stemmed = word
		}
		
		noun := NounEntity{
			Word:      word,
			Frequency: freq,
			Positions: wordPositions[word],
			Stemmed:   stemmed,
		}
		nouns = append(nouns, noun)
	}
	
	return nouns
}

func (n *NLPService) isLikelyNoun(word string) bool {
	if len(word) < 3 {
		return false
	}
	
	if n.stopWords[word] {
		return false
	}
	
	if strings.HasSuffix(word, "ing") || strings.HasSuffix(word, "ed") {
		return false
	}
	
	if strings.HasSuffix(word, "ly") {
		return false
	}
	
	if unicode.IsUpper(rune(word[0])) {
		return true
	}
	
	nounSuffixes := []string{"tion", "sion", "ness", "ment", "ity", "ty", "er", "or", "ist", "ism"}
	for _, suffix := range nounSuffixes {
		if strings.HasSuffix(word, suffix) {
			return true
		}
	}
	
	return len(word) >= 4
}

func (n *NLPService) findWordPositions(word, text string) []int {
	var positions []int
	wordPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	matches := wordPattern.FindAllStringIndex(text, -1)
	
	for _, match := range matches {
		positions = append(positions, match[0])
	}
	
	return positions
}

func (n *NLPService) BuildCoOccurrenceMatrix(nouns []NounEntity, sentences []string, windowSize int) map[string]map[string]int {
	matrix := make(map[string]map[string]int)
	
	for _, noun := range nouns {
		matrix[noun.Word] = make(map[string]int)
	}
	
	for _, sentence := range sentences {
		words := n.extractWords(sentence)
		sentenceNouns := make([]string, 0)
		
		for _, word := range words {
			if n.isLikelyNoun(word) {
				sentenceNouns = append(sentenceNouns, word)
			}
		}
		
		for i, noun1 := range sentenceNouns {
			for j, noun2 := range sentenceNouns {
				if i != j && abs(i-j) <= windowSize {
					if matrix[noun1] == nil {
						matrix[noun1] = make(map[string]int)
					}
					matrix[noun1][noun2]++
				}
			}
		}
	}
	
	return matrix
}

func (n *NLPService) BuildSequenceMatrix(nouns []NounEntity, originalText string) map[string]map[string]int {
	matrix := make(map[string]map[string]int)
	
	for _, noun := range nouns {
		matrix[noun.Word] = make(map[string]int)
	}
	
	words := n.extractWords(originalText)
	sequenceNouns := make([]string, 0)
	
	for _, word := range words {
		if n.isLikelyNoun(word) {
			sequenceNouns = append(sequenceNouns, word)
		}
	}
	
	for i := 0; i < len(sequenceNouns)-1; i++ {
		currentNoun := sequenceNouns[i]
		nextNoun := sequenceNouns[i+1]
		
		if matrix[currentNoun] == nil {
			matrix[currentNoun] = make(map[string]int)
		}
		matrix[currentNoun][nextNoun]++
	}
	
	return matrix
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}