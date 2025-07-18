package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// IgnorePattern represents a single gitignore pattern with its context
type IgnorePattern struct {
    Pattern      string
    BaseDir      string
    IsNegation   bool
    IsDirectory  bool
    regex        *regexp.Regexp
}

// IgnoreMatcher is a function that checks if a path should be ignored
type IgnoreMatcher func(path string) bool

// ParseGitignorePatterns creates an ignore matcher from a list of patterns
func ParseGitignorePatterns(patterns []PatternSource) IgnoreMatcher {
    rules := make([]*IgnorePattern, 0)
    
    for _, ps := range patterns {
        rule := parsePattern(ps.Pattern, ps.BaseDir)
        if rule != nil {
            rules = append(rules, rule)
        }
    }
    
    // If no negation rules, we can use simple matching
    hasNegation := false
    for _, rule := range rules {
        if rule.IsNegation {
            hasNegation = true
            break
        }
    }
    
    if !hasNegation {
        return func(path string) bool {
            for _, rule := range rules {
                if rule.Match(path) {
                    return true
                }
            }
            return false
        }
    }
    
    // With negation rules, we need to check in reverse order
    return func(path string) bool {
        // Later rules override earlier rules
        for i := len(rules) - 1; i >= 0; i-- {
            if rules[i].Match(path) {
                return !rules[i].IsNegation
            }
        }
        return false
    }
}

// PatternSource represents a pattern and its source directory
type PatternSource struct {
    Pattern string
    BaseDir string
}

// parsePattern converts a gitignore pattern to an IgnorePattern
func parsePattern(pattern, baseDir string) *IgnorePattern {
    pattern = strings.TrimSpace(pattern)
    
    // Skip empty lines and comments
    if pattern == "" || strings.HasPrefix(pattern, "#") {
        return nil
    }
    
    ip := &IgnorePattern{
        Pattern: pattern,
        BaseDir: normalizePath(baseDir),
    }
    
    // Handle negation
    if strings.HasPrefix(pattern, "!") {
        ip.IsNegation = true
        pattern = pattern[1:]
    }
    
    // Handle directory-only patterns
    if strings.HasSuffix(pattern, "/") {
        ip.IsDirectory = true
        pattern = strings.TrimSuffix(pattern, "/")
    }
    
    // Convert pattern to regex
    regexStr := patternToRegex(pattern)
    regex, err := regexp.Compile(regexStr)
    if err != nil {
        Log.Warning("Invalid gitignore pattern '%s': %v", pattern, err)
        return nil
    }
    ip.regex = regex
    
    return ip
}

// Match checks if a path matches this ignore pattern
func (ip *IgnorePattern) Match(path string) bool {
    path = normalizePath(path)
    
    // Get relative path from base directory
    relPath := path
    if ip.BaseDir != "" {
        rel, err := filepath.Rel(ip.BaseDir, path)
        if err != nil || strings.HasPrefix(rel, "..") {
            // Path is outside base directory
            return false
        }
        relPath = rel
    }
    
    // Normalize path separators for matching
    relPath = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
    
    return ip.regex.MatchString(relPath)
}

// patternToRegex converts a gitignore pattern to a regular expression
func patternToRegex(pattern string) string {
    // Remove leading slash (patterns are relative to their directory)
    anchored := strings.Contains(pattern, "/")
    if strings.HasPrefix(pattern, "/") {
        pattern = pattern[1:]
        anchored = true
    }
    
    // Handle ** patterns
    pattern = strings.ReplaceAll(pattern, "**/", "(.*/)?")
    
    // Escape special regex characters except * and ?
    specialChars := []string{".", "+", "^", "$", "(", ")", "[", "]", "{", "}", "|"}
    for _, char := range specialChars {
        pattern = strings.ReplaceAll(pattern, char, "\\"+char)
    }
    
    // Convert * and ? to regex equivalents
    // * matches anything except /
    pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
    // ? matches any single character except /
    pattern = strings.ReplaceAll(pattern, "?", "[^/]")
    
    // Build final regex
    if anchored {
        return "^" + pattern + "(/.*)?$"
    }
    return "(^|/)" + pattern + "(/.*)?$"
}

// normalizePath normalizes a file path
func normalizePath(path string) string {
    if path == "" {
        return ""
    }
    abs, err := filepath.Abs(path)
    if err != nil {
        return path
    }
    return abs
}

// ReadGitignoreFile reads patterns from a .gitignore file
func ReadGitignoreFile(filePath string) ([]PatternSource, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var patterns []PatternSource
    baseDir := filepath.Dir(filePath)
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line != "" && !strings.HasPrefix(line, "#") {
            patterns = append(patterns, PatternSource{
                Pattern: line,
                BaseDir: baseDir,
            })
        }
    }
    
    return patterns, scanner.Err()
}
