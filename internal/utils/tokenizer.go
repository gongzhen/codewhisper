package utils

import (
	"os"
	"unicode"
)

// CountTokens provides a simple token count approximation
// In the Python version, they use tiktoken. For now, we'll use a simple approximation.
// Later we can integrate a proper tokenizer.
func CountTokens(text string) int {
    // Simple approximation: split by whitespace and punctuation
    // This gives roughly similar results to tiktoken for English text
    
    wordCount := 0
    inWord := false
    
    for _, r := range text {
        if unicode.IsLetter(r) || unicode.IsDigit(r) {
            if !inWord {
                wordCount++
                inWord = true
            }
        } else {
            inWord = false
            // Count some punctuation as tokens
            if r == '.' || r == ',' || r == '!' || r == '?' || r == ';' || r == ':' {
                wordCount++
            }
        }
    }
     
    // Approximate tokens as 0.75 * word count (rough approximation)
    // tiktoken typically produces more tokens than word count
    return int(float64(wordCount) * 0.75)
}

// CountTokensInFile counts tokens in a file
func CountTokensInFile(filePath string) int {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return 0
    }
    
    // Skip binary files
    if IsBinaryFile(filePath) {
        return 0
    }
    
    return CountTokens(string(content))
}
