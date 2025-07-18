package utils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Binary file extensions
var binaryExtensions = map[string]bool{
    ".pyc":   true,
    ".pyo":   true,
    ".pyd":   true,
    ".ico":   true,
    ".png":   true,
    ".jpg":   true,
    ".jpeg":  true,
    ".gif":   true,
    ".svg":   true,
    ".core":  true,
    ".bin":   true,
    ".exe":   true,
    ".dll":   true,
    ".so":    true,
    ".dylib": true,
    ".class": true,
    ".woff":  true,
    ".woff2": true,
    ".ttf":   true,
    ".eot":   true,
    ".zip":   true,
}

// IsBinaryFile checks if a file is binary based on extension or content
func IsBinaryFile(filePath string) bool {
    // Check if path is a directory first
    info, err := os.Stat(filePath)
    if err != nil || info.IsDir() {
        return false
    }
    
    // Check extension for known binary types
    ext := strings.ToLower(filepath.Ext(filePath))
    if binaryExtensions[ext] {
        Log.Debug("Detected binary file by extension: %s", filePath)
        return true
    }
    
    // Try to detect if file is binary by reading first few bytes
    file, err := os.Open(filePath)
    if err != nil {
        Log.Debug("Unable to open file %s: %v", filePath, err)
        return false
    }
    defer file.Close()
    
    // Read first 1024 bytes
    buffer := make([]byte, 1024)
    n, err := file.Read(buffer)
    if err != nil && err != io.EOF {
        return false
    }
    
    // Check for null bytes (common in binary files)
    for i := 0; i < n; i++ {
        if buffer[i] == 0 {
            return true
        }
    }
    
    return false
}

// IsImageFile checks if a file is an image based on extension
func IsImageFile(filePath string) bool {
    imageExtensions := []string{
        ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".ico",
    }
    
    ext := strings.ToLower(filepath.Ext(filePath))
    for _, imgExt := range imageExtensions {
        if ext == imgExt {
            return true
        }
    }
    return false
}