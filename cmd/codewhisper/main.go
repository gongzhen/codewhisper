package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gongzhen/codewhisper-go/internal/server"
	"github.com/gongzhen/codewhisper-go/internal/utils"
	"github.com/gongzhen/codewhisper-go/pkg/config"
	"github.com/joho/godotenv"
)

// Version information
const (
    defaultPort = 6969
)

// Config holds the application configuration
type Config struct {
    Port          int
    Exclude       []string
    Profile       string
    Model         string
    MaxDepth      int
    Version       bool
    CheckAuth     bool
    Target        string
    Endpoint      string
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() error {
    // Try to load from current directory first
    if err := godotenv.Load(); err == nil {
        utils.Log.Info("Loaded .env from current directory")
        return nil
    }
    
    // Try to load from home directory ~/.codewhisper/.env (like Python version)
    homeDir, err := os.UserHomeDir()
    if err == nil {
        codewhisperEnvPath := filepath.Join(homeDir, ".codewhisper", ".env")
        if err := godotenv.Load(codewhisperEnvPath); err == nil {
            utils.Log.Info("Loaded .env from ~/.codewhisper/.env")
            return nil
        }
    }
    
    // Try to load from the same directory as the executable
    execPath, err := os.Executable()
    if err == nil {
        execDir := filepath.Dir(execPath)
        execEnvPath := filepath.Join(execDir, ".env")
        if err := godotenv.Load(execEnvPath); err == nil {
            utils.Log.Info("Loaded .env from executable directory")
            return nil
        }
    }
    
    return fmt.Errorf("no .env file found")
}

func main() {
    // Load .env file
    if err := loadEnvFile(); err != nil {
        utils.Log.Warning("No .env file loaded: %v", err)
    }
        
	cfg := parseFlags()

    if cfg.Version {
        fmt.Printf("CodeWhisper version %s\n", utils.CurrentVersion)
        return
	}  
    
    // Check for updates (but don't block)
    go checkVersionAsync()

    // Setup environment variables based on flags
    setupEnvironment(cfg)	

    // Now use our custom logger instead of log package    
    utils.Log.Info("Starting CodeWhisper on port %d...", cfg.Port)
    utils.Log.Info("Target directory: %s", cfg.Target)
    utils.Log.Info("Model endpoint: %s", cfg.Endpoint)
    if cfg.Model != "" {
        utils.Log.Info("Model: %s", cfg.Model)
    }

    // Create and start server
    srv := server.NewServer(cfg.Port)

    go func ()  {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
        <-sigChan

        utils.Log.Info("Shutting down server...")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

        defer cancel()

        if err := srv.Shutdown(ctx); err != nil {
            utils.Log.Error("Server shutdown error: %v", err)
        }
    }()

        // Start server
    if err := srv.Start(); err != nil && err != http.ErrServerClosed {
        utils.Log.Error("Server error: %v", err)
        os.Exit(1)
    }
}

func checkVersionAsync() {
    hasUpdate, latestVersion, err := utils.CheckForUpdates()

    if err != nil {
        utils.Log.Warning("Failed to check for updates. %v", err)
        return
    }

    if hasUpdate {
        utils.Log.Info("New version available: %s (current: %s)", latestVersion, utils.CurrentVersion)
        utils.Log.Info("You are using the Go version of CodeWhisper. Please update manually if needed.")
    } else {
        utils.Log.Info("CodeWhisper version %s is up to date", utils.CurrentVersion)
    }
}

func setupEnvironment(cfg *Config) {
    // Set environment variables from config
    config.SetEnv(config.EnvUserCodebaseDir, cfg.Target)
    
    // Convert exclude slice to comma-separated string
    if len(cfg.Exclude) > 0 {
        config.SetEnv(config.EnvAdditionalExcludeDirs, strings.Join(cfg.Exclude, ","))
    }
    
    if cfg.Profile != "" {
        config.SetEnv(config.EnvAWSProfile, cfg.Profile)
    }
    
    if cfg.Model != "" {
        config.SetEnv(config.EnvModel, cfg.Model)
    }
    
    config.SetEnv(config.EnvEndpoint, cfg.Endpoint)
    config.SetEnv(config.EnvMaxDepth, fmt.Sprintf("%d", cfg.MaxDepth))
}

func parseFlags() *Config {
	config := &Config{}

	// Define command-line flags (equivalent to Python's argparse)
	flag.IntVar(&config.Port, "port", defaultPort, "Port number to run CodeWhisper frontend on")
    flag.StringVar(&config.Target, "target", ".", "Target directory to analyze")
    flag.StringVar(&config.Profile, "profile", "", "AWS profile to use")
    flag.StringVar(&config.Model, "model", "", "Model to use from selected endpoint")
    flag.StringVar(&config.Endpoint, "endpoint", "openai", "Model endpoint to use (bedrock, google, openai, deepseek)")
    flag.IntVar(&config.MaxDepth, "max-depth", 15, "Maximum depth for folder structure traversal")
    flag.BoolVar(&config.Version, "version", false, "Print version information")
    flag.BoolVar(&config.CheckAuth, "check-auth", false, "Check authentication setup without starting server")
    	
	// Custom flag for exclude (we'll handle the comma-separated list)
	var excludeStr string
    flag.StringVar(&excludeStr, "exclude", "", "Comma-separated list of files/directories to exclude")
    
    flag.Parse()
	
    // Process exclude list	
	if excludeStr != "" {
		config.Exclude = parseExcludeList(excludeStr)
	}

	if absPath, err := filepath.Abs(config.Target); err == nil {
		config.Target = absPath
	}

	return config
}

func parseExcludeList(excludeStr string) []string {
    // Split the string by comma
	parts := strings.Split(excludeStr, ",")

	// Create a slice to hold the cleaned results
	result := make([]string, 0, len(parts))

	// Trim whitespace from each part
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" { // Only add non-empty strings
			result = append(result, trimmed)
		}
	}
    return result
}