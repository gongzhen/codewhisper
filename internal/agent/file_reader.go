package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gongzhen/codewhisper-go/internal/utils"
)

type FileReader struct{}

func NewFileReader() *FileReader {
    return &FileReader{}
}

func (fr *FileReader) ReadFile(baseDir, relPath string) (string, error) {
    // Construct full path
    fullPath := filepath.Join(baseDir, relPath)
    
    // Security check - ensure path is within base directory
    absPath, err := filepath.Abs(fullPath)
    if err != nil {
        return "", fmt.Errorf("invalid path: %w", err)
    }
    
    absBase, err := filepath.Abs(baseDir)
    if err != nil {
        return "", fmt.Errorf("invalid base directory: %w", err)
    }
    
    // Ensure the path is within the base directory
    if !strings.HasPrefix(absPath, absBase) {
        return "", fmt.Errorf("path outside base directory")
    }
    
    // Check if path exists
    info, err := os.Stat(fullPath)
    if err != nil {
        if os.IsNotExist(err) {
            return "", fmt.Errorf("file not found: %s", relPath)
        }
        return "", fmt.Errorf("cannot access file: %w", err)
    }
    
    // Skip directories
    if info.IsDir() {
        return "", fmt.Errorf("path is a directory: %s", relPath)
    }
    
    // Skip binary files
    if utils.IsBinaryFile(fullPath) {
        return "", fmt.Errorf("binary file: %s", relPath)
    }
    
    // Skip large files (> 1MB)
    if info.Size() > 1024*1024 {
        return "", fmt.Errorf("file too large: %s (size: %d bytes)", relPath, info.Size())
    }
    
    // Read file content
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return "", fmt.Errorf("error reading file: %w", err)
    }
    
    return string(content), nil
}

// ReadFiles reads multiple files and returns a map of path -> content
func (fr *FileReader) ReadFiles(baseDir string, files []string) (map[string]string, error) {
    fileContents := make(map[string]string)
    
    for _, file := range files {
        content, err := fr.ReadFile(baseDir, file)
        if err != nil {
            utils.Log.Warning("Skipping file %s: %v", file, err)
            continue
        }
        fileContents[file] = content
    }
    
    if len(fileContents) == 0 {
        return nil, fmt.Errorf("no valid files could be read")
    }
    
    return fileContents, nil
}
