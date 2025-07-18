package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gongzhen/codewhisper-go/pkg/config"
)

// GetIgnoredPatterns returns all patterns that should be ignored
func GetIgnoredPatterns(directory string) []PatternSource {
    patterns := []PatternSource{
        {Pattern: "poetry.lock", BaseDir: directory},
        {Pattern: "package-lock.json", BaseDir: directory},
        {Pattern: ".DS_Store", BaseDir: directory},
        {Pattern: ".git", BaseDir: directory},
    }
    
    // Add additional patterns from environment
    additionalExclude := config.GetEnv(config.EnvAdditionalExcludeDirs, "")
    if additionalExclude != "" {
        excludeList := strings.Split(additionalExclude, ",")
        for _, pattern := range excludeList {
            pattern = strings.TrimSpace(pattern)
            if pattern != "" {
                patterns = append(patterns, PatternSource{
                    Pattern: pattern,
                    BaseDir: directory,
                })
            }
        }
    }
    
    // Recursively read .gitignore files
    patterns = append(patterns, getGitignorePatternsRecursive(directory)...)
    
    return patterns
}

// getGitignorePatternsRecursive recursively finds and reads .gitignore files
func getGitignorePatternsRecursive(dir string) []PatternSource {
    var allPatterns []PatternSource
    
    // Read .gitignore in current directory
    gitignorePath := filepath.Join(dir, ".gitignore")
    if patterns, err := ReadGitignoreFile(gitignorePath); err == nil {
        allPatterns = append(allPatterns, patterns...)
    }
    
    // Walk subdirectories
    entries, err := os.ReadDir(dir)
    if err != nil {
        return allPatterns
    }
    
    for _, entry := range entries {
        if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
            subdir := filepath.Join(dir, entry.Name())
            allPatterns = append(allPatterns, getGitignorePatternsRecursive(subdir)...)
        }
    }
    
    return allPatterns
}

// GetCompleteFileList returns all files in the given directories, respecting ignore patterns
func GetCompleteFileList(baseDir string, ignoredPatterns []PatternSource, includedDirs []string) map[string]struct{} {
    shouldIgnore := ParseGitignorePatterns(ignoredPatterns)
    fileMap := make(map[string]struct{})
    
    for _, relDir := range includedDirs {
        startPath := filepath.Join(baseDir, relDir)
        
        err := filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return nil // Skip errors
            }
            
            // Skip hidden files and directories
            if strings.HasPrefix(filepath.Base(path), ".") {
                if info.IsDir() {
                    return filepath.SkipDir
                }
                return nil
            }
            
            // Check if should be ignored
            if shouldIgnore(path) {
                if info.IsDir() {
                    return filepath.SkipDir
                }
                return nil
            }
            
            // Skip image files
            if !info.IsDir() && IsImageFile(path) {
                return nil
            }
            
            // Add to map
            fileMap[path] = struct{}{}
            
            return nil
        })
        
        if err != nil {
            Log.Warning("Error walking directory %s: %v", startPath, err)
        }
    }
    
    return fileMap
}
