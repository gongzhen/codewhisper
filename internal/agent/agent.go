package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/gongzhen/codewhisper-go/internal/models"
	"github.com/gongzhen/codewhisper-go/internal/utils"
	"github.com/gongzhen/codewhisper-go/pkg/config"
)

type Agent struct {
    modelManager *models.ModelManager
    fileReader   *FileReader
}

func NewAgent() (*Agent, error) {
    modelManager, err := models.NewModelManager()
    if err != nil {
        return nil, fmt.Errorf("failed to initialize model manager: %w", err)
    }
    
    return &Agent{
        modelManager: modelManager,
        fileReader:   NewFileReader(),
    }, nil
}

type ChatRequest struct {
    Input struct {
        Question    string     `json:"question"`
        ChatHistory [][]string `json:"chat_history"`
        Config      struct {
            Files []string `json:"files"`
        } `json:"config"`
        ConversationID string `json:"conversation_id,omitempty"`
    } `json:"input"`
}

type StreamEvent struct {
    Content string `json:"content,omitempty"`
    Error   string `json:"error,omitempty"`
    Detail  string `json:"detail,omitempty"`
}

func (a *Agent) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
    eventChan := make(chan StreamEvent, 100)

    go func() {
        defer func() {
            utils.Log.Info("Closing event channel")
            close(eventChan)
        }()

        if req.Input.Question == "" || strings.TrimSpace(req.Input.Question) == "" {
            eventChan <- StreamEvent{
                Error:  "validation_error",
                Detail: "Please provide a question to continue.",
            }
            return
        }

        utils.Log.Info("Stream chat request - Question: %s", req.Input.Question)
        utils.Log.Info("Files to analyze: %d", len(req.Input.Config.Files))

        // Step 1: Build context from selected files
        codebaseContext, err := a.buildCodebaseContext(req.Input.Config.Files)
        if err != nil {
            eventChan <- StreamEvent{
                Error:  "file_error",
                Detail: fmt.Sprintf("Error reading files: %v", err),
            }
            return
        }

        tokenCount := utils.CountTokens(codebaseContext)
        utils.Log.Info("Codebase context: %d tokens, %d chars", tokenCount, len(codebaseContext))

        if tokenCount > 180000 {
            eventChan <- StreamEvent{
                Error:  "token_limit_exceeded",
                Detail: "Selected files are too large. Please select fewer files.",
            }
            return
        }

        // Step 2: Format prompt
        fullPrompt := a.formatPrompt(codebaseContext, req.Input.Question, req.Input.ChatHistory)
        utils.Log.Info("Prompt size: %d chars", len(fullPrompt))

        // Step 3: Call model
        utils.Log.Info("Calling OpenAI model...")
        modelStream, err := a.modelManager.StreamChat(ctx, fullPrompt)
        if err != nil {
            utils.Log.Error("Model stream error: %v", err)
            eventChan <- StreamEvent{
                Error:  "model_error",
                Detail: fmt.Sprintf("Model stream error: %v", err),
            }
            return
        }

        // Step 4: Forward chunks
        utils.Log.Info("Starting to forward model response chunks...")
        chunkCount := 0
        for chunk := range modelStream {
            if chunk.Error != nil {
                utils.Log.Error("Chunk error: %v", chunk.Error)
                select {
                case eventChan <- StreamEvent{
                    Error:  "stream_error",
                    Detail: chunk.Error.Error(),
                }:
                case <-ctx.Done():
                    utils.Log.Info("Agent context canceled")
                    return
                }
                return
            }

            if chunk.Content != "" {
                chunkCount++
                if chunkCount%10 == 0 {
                    utils.Log.Info("Forwarded %d chunks", chunkCount)
                }
                
                // Send to channel with select to avoid blocking
                select {
                case eventChan <- StreamEvent{Content: chunk.Content}:
                    // Successfully sent
                case <-ctx.Done():
                    utils.Log.Info("Context canceled, stopping chunk forwarding")
                    return
                }
            }
        }
        utils.Log.Info("Finished forwarding %d chunks", chunkCount)
    }()

    return eventChan, nil
}

func (a *Agent) buildCodebaseContext(files []string) (string, error) {
    userCodebaseDir := config.GetEnv(config.EnvUserCodebaseDir, ".")
    
    var builder strings.Builder
    fileCount := 0
    
    for _, filePath := range files {
        content, err := a.fileReader.ReadFile(userCodebaseDir, filePath)
        if err != nil {
            utils.Log.Warning("Skipping file %s: %v", filePath, err)
            continue
        }
        
        // Add file header and content
        builder.WriteString(fmt.Sprintf("File: %s\n", filePath))
        builder.WriteString(content)
        builder.WriteString("\n\n")
        fileCount++
    }
    
    utils.Log.Info("Successfully read %d files", fileCount)
    
    if fileCount == 0 {
        return "", fmt.Errorf("no valid files to analyze")
    }
    
    return builder.String(), nil
}

func (a *Agent) formatPrompt(codebase, question string, chatHistory [][]string) string {
    var prompt strings.Builder
    
    // System prompt
    prompt.WriteString(SystemPrompt)
    prompt.WriteString("\n\n")
    
    // Add codebase
    prompt.WriteString("Current codebase:\n")
    prompt.WriteString(codebase)
    prompt.WriteString("\n")
    
    // Add chat history if any
    if len(chatHistory) > 0 {
        prompt.WriteString("\nConversation history:\n")
        for _, exchange := range chatHistory {
            if len(exchange) >= 2 {
                prompt.WriteString(fmt.Sprintf("Human: %s\n", exchange[0]))
                prompt.WriteString(fmt.Sprintf("Assistant: %s\n\n", exchange[1]))
            }
        }
    }
    
    // Add current question
    prompt.WriteString(fmt.Sprintf("Human: %s\n", question))
    prompt.WriteString("Assistant: ")
    
    return prompt.String()
}

// GetCurrentModelInfo returns current model information
func (a *Agent) GetCurrentModelInfo() map[string]interface{} {
    return map[string]interface{}{
        "model_id":       config.GetEnv(config.EnvModel, "sonnet3.5-v2"),
        "endpoint":       config.GetEnv(config.EnvEndpoint, "bedrock"),
        "max_tokens":     200000,
        "context_window": 200000,
    }
}