package models

import (
	"context"
	"fmt"

	"github.com/gongzhen/codewhisper-go/pkg/config"
)

// StreamChunk represents a piece of streamed response
type StreamChunk struct {
    Content string
    Error   error
}

// ModelInfo contains information about the current model
type ModelInfo struct {
    ModelID    string `json:"model_id"`
    Endpoint   string `json:"endpoint"`
    MaxTokens  int    `json:"max_tokens"`
}

// Provider interface for different model providers
type Provider interface {
    StreamChat(ctx context.Context, prompt string) (<-chan StreamChunk, error)
    ValidateAuth() error
    GetModelInfo() ModelInfo
}

// ModelManager placeholder - will be implemented in next task
// ModelManager manages different model providers
type ModelManager struct {
    providers map[string]Provider
    current   string
}

// NewModelManager creates a new model manager
func NewModelManager() (*ModelManager, error) {
    mm := &ModelManager{
        providers: make(map[string]Provider),
    }
    
    // Get endpoint from config
    endpoint := config.GetEnv(config.EnvEndpoint, "openai")
    
    switch endpoint {
    case "openai":
        provider, err := NewOpenAIProvider()
        if err != nil {
            return nil, fmt.Errorf("failed to initialize OpenAI provider: %w", err)
        }
        mm.providers["openai"] = provider
        mm.current = "openai"
        
    case "bedrock":
        // We can add bedrock later
        return nil, fmt.Errorf("bedrock not implemented yet, use --endpoint openai")
        
    default:
        return nil, fmt.Errorf("unsupported endpoint: %s", endpoint)
    }
    
    // Validate authentication
    if err := mm.ValidateAuth(); err != nil {
        return nil, fmt.Errorf("authentication failed: %w", err)
    }
    
    return mm, nil
}

// StreamChat streams a chat response
func (mm *ModelManager) StreamChat(ctx context.Context, prompt string) (<-chan StreamChunk, error) {
    provider, exists := mm.providers[mm.current]
    if !exists {
        return nil, fmt.Errorf("no provider for endpoint: %s", mm.current)
    }
    
    return provider.StreamChat(ctx, prompt)
}

// ValidateAuth validates authentication for the current provider
func (mm *ModelManager) ValidateAuth() error {
    provider, exists := mm.providers[mm.current]
    if !exists {
        return fmt.Errorf("no provider configured")
    }
    
    return provider.ValidateAuth()
}

// GetCurrentModelInfo returns information about the current model
func (mm *ModelManager) GetCurrentModelInfo() ModelInfo {
    if provider, exists := mm.providers[mm.current]; exists {
        return provider.GetModelInfo()
    }
    
    return ModelInfo{
        ModelID:  config.GetEnv(config.EnvModel, "gpt-4-turbo-preview"),
        Endpoint: mm.current,
    }
}