package models

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"

	"github.com/gongzhen/codewhisper-go/internal/utils"
	"github.com/gongzhen/codewhisper-go/pkg/config"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
    client  *openai.Client
    modelID string
}

// Model mapping for OpenAI
var openAIModelMap = map[string]string{
    "gpt4":          "gpt-4-turbo-preview",
    "gpt-4":         "gpt-4-turbo-preview",
    "gpt-4-turbo":   "gpt-4-turbo-preview",
    "gpt-3.5":       "gpt-3.5-turbo",
    "gpt-3.5-turbo": "gpt-3.5-turbo",
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() (*OpenAIProvider, error) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set. " +
            "Please set it in your environment or create a .env file with:\n" +
            "OPENAI_API_KEY=sk-your-api-key-here")
    }
    
    // Get model from config
    modelAlias := config.GetEnv(config.EnvModel, "gpt-4-turbo")
    modelID, exists := openAIModelMap[modelAlias]
    if !exists {
        modelID = modelAlias // Use as-is if not in map
    }
    
    utils.Log.Info("Using OpenAI model: %s", modelID)
    var client *openai.Client
    apiBase := os.Getenv("OPENAI_API_BASE")
    if apiBase != "" {
        utils.Log.Info("Using custom API base: %s", apiBase)
        config := openai.DefaultConfig(apiKey)
        config.BaseURL = apiBase
        client = openai.NewClientWithConfig(config)
    } else {
        client = openai.NewClient(apiKey)
    }
    
    return &OpenAIProvider{
        client:  client,
        modelID: modelID,
    }, nil
}

// StreamChat implements streaming chat for OpenAI
func (o *OpenAIProvider) StreamChat(ctx context.Context, prompt string) (<-chan StreamChunk, error) {
    streamChan := make(chan StreamChunk, 100)
    
    go func() {
        defer close(streamChan)
        var fullResponse strings.Builder
        defer func() {
            utils.Log.Info("Full OpenAI response: %s", fullResponse.String())
        }()        
        
        // Parse the prompt to extract system message
        messages := o.parsePrompt(prompt)

        // EXTRA SAFETY: Filter out any empty messages before sending
        var validMessages []openai.ChatCompletionMessage
        for _, msg := range messages {
            if strings.TrimSpace(msg.Content) != "" {
                validMessages = append(validMessages, msg)
            } else {
                utils.Log.Warning("Removing empty %s message before API call", msg.Role)
            }
        }
        messages = validMessages

        // Make sure we have at least one message
        if len(messages) == 0 {
            utils.Log.Error("No valid messages after filtering")
            streamChan <- StreamChunk{Error: fmt.Errorf("no valid messages to send")}
            return
        }        
        
        // Log the parsed messages for debugging
        utils.Log.Info("Parsed %d messages for OpenAI", len(messages))
        for i, msg := range messages {
            utils.Log.Info("Message %d - Role: %s, Content length: %d", i, msg.Role, len(msg.Content))
        }
        
        // Get parameters from config
        temperature := float32(0.7)
        if tempStr := config.GetEnv(config.EnvTemperature, "0.7"); tempStr != "" {
            if temp, err := strconv.ParseFloat(tempStr, 32); err == nil {
                temperature = float32(temp)
            }
        }
        
        maxTokens := config.GetEnvInt(config.EnvMaxOutputTokens, 4096)
        
        // Create chat completion request
        req := openai.ChatCompletionRequest{
            Model:       o.modelID,
            Messages:    messages,
            Temperature: temperature,
            MaxTokens:   maxTokens,
            Stream:      true,
        }
        
        utils.Log.Info("Creating OpenAI stream with model: %s, temperature: %.2f, maxTokens: %d", 
            o.modelID, temperature, maxTokens)
        

        // In StreamChat function, right before CreateChatCompletionStream
        utils.Log.Info("=== DEBUG: Sending request to API ===")
        utils.Log.Info("Model: %s", req.Model)
        utils.Log.Info("Number of messages: %d", len(req.Messages))
        for i, msg := range req.Messages {
            utils.Log.Info("Message %d:", i)
            utils.Log.Info("  Role: %s", msg.Role)
            utils.Log.Info("  Content length: %d", len(msg.Content))
            utils.Log.Info("  Content preview: %.100s...", msg.Content) // First 100 chars
            if msg.Content == "" {
                utils.Log.Error("  WARNING: Empty content in message!")
            }
        }
        utils.Log.Info("=== END DEBUG ===")            
        // Create stream
        stream, err := o.client.CreateChatCompletionStream(ctx, req)
        if err != nil {
            utils.Log.Error("Failed to create OpenAI stream: %v", err)
            streamChan <- StreamChunk{Error: fmt.Errorf("failed to create stream: %w", err)}
            return
        }
        defer stream.Close()
        
        utils.Log.Info("OpenAI stream created successfully, starting to receive...")
        // Process stream
        chunkCount := 0
        for {
            response, err := stream.Recv()
            if errors.Is(err, io.EOF) {
                utils.Log.Info("OpenAI stream completed after %d chunks", chunkCount)
                break
            }
            
            if err != nil {
                utils.Log.Error("OpenAI stream error: %v", err)
                streamChan <- StreamChunk{Error: fmt.Errorf("stream error: %w", err)}
                return
            }
            
            // Extract content from response
            if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
                chunkCount++
                content := response.Choices[0].Delta.Content
                fullResponse.WriteString(content)
                if chunkCount <= 5 || chunkCount % 10 == 0 {
                    utils.Log.Info("Chunk %d: %q", chunkCount, content)
                }
                streamChan <- StreamChunk{Content: content}
            }
        }
    }()
    
    return streamChan, nil
}

// parsePrompt converts our prompt format to OpenAI messages
func (o *OpenAIProvider) parsePrompt(prompt string) []openai.ChatCompletionMessage {
    messages := []openai.ChatCompletionMessage{}
    
    // Check if prompt contains system message
    if strings.Contains(prompt, "You are CodeWhisper") {
        // Split on the codebase marker
        parts := strings.SplitN(prompt, "\n\nCurrent codebase:", 2)
        if len(parts) == 2 {
            // Add system message
            messages = append(messages, openai.ChatCompletionMessage{
                Role:    openai.ChatMessageRoleSystem,
                Content: strings.TrimSpace(parts[0]),
            })
            
            // Parse the rest for user message and chat history
            rest := parts[1]
            
            // Check for chat history
            if strings.Contains(rest, "\nConversation history:\n") {
                historyParts := strings.SplitN(rest, "\nConversation history:\n", 2)
                
                // Add codebase as first user message
                codebasePart := strings.TrimSpace(historyParts[0])
                
                if len(historyParts) == 2 {
                    // Parse conversation history
                    convParts := strings.Split(historyParts[1], "\nHuman: ")
                    
                    // Add the codebase context as the first message
                    messages = append(messages, openai.ChatCompletionMessage{
                        Role:    openai.ChatMessageRoleUser,
                        Content: "Current codebase:" + codebasePart,
                    })
                    
                    // Process each conversation turn
                    for _, part := range convParts {
                        if part == "" {
                            continue
                        }
                        
                        if strings.Contains(part, "\nAssistant: ") {
                            humanAssistant := strings.SplitN(part, "\nAssistant: ", 2)
                            if len(humanAssistant) == 2 {
                                humanContent := strings.TrimSpace(humanAssistant[0])
                                assistantContent := strings.TrimSpace(humanAssistant[1])

                                // Add human message if not empty
                                if humanContent != "" && humanContent != "Human:" {
                                    messages = append(messages, 
                                        openai.ChatCompletionMessage{
                                            Role:    openai.ChatMessageRoleUser,
                                            Content: humanContent,
                                        },
                                    )
                                }
                                
                                // IMPORTANT: Only add assistant message if it has real content
                                // Skip if it's empty, just "Assistant:", or ends with "Assistant:"
                                if assistantContent != "" && 
                                   assistantContent != "Assistant:" &&
                                   !strings.HasSuffix(assistantContent, "Assistant:") {
                                    messages = append(messages, 
                                        openai.ChatCompletionMessage{
                                            Role:    openai.ChatMessageRoleAssistant,
                                            Content: assistantContent,
                                        },
                                    )
                                }                                

                                // // Skip the closing Assistant: prompt at the end
                                // if !strings.HasSuffix(humanAssistant[1], "Assistant: ") {
                                //     messages = append(messages, 
                                //         openai.ChatCompletionMessage{
                                //             Role:    openai.ChatMessageRoleUser,
                                //             Content: strings.TrimSpace(humanAssistant[0]),
                                //         },
                                //         openai.ChatCompletionMessage{
                                //             Role:    openai.ChatMessageRoleAssistant,
                                //             Content: strings.TrimSpace(humanAssistant[1]),
                                //         },
                                //     )
                                // } else {
                                //     // This is the current question
                                //     messages = append(messages, openai.ChatCompletionMessage{
                                //         Role:    openai.ChatMessageRoleUser,
                                //         Content: strings.TrimSpace(humanAssistant[0]),
                                //     })
                                // }
                            }
                        } else {
                            content := strings.TrimSpace(part)
                            if content != "" && content != "Human:" {
                                messages = append(messages, openai.ChatCompletionMessage{
                                    Role:    openai.ChatMessageRoleUser,
                                    Content: content,
                                })                                
                            }
                        }
                    }
                } else {
                    // No history, just add the question
                    if strings.Contains(rest, "\nHuman: ") {
                        parts := strings.SplitN(rest, "\nHuman: ", 2)
                        if len(parts) == 2 {
                            content := strings.TrimSuffix(parts[1], "\nAssistant: ")
                            content = strings.TrimSpace(content)
                            if content != "" {
                                messages = append(messages, openai.ChatCompletionMessage{
                                    Role:    openai.ChatMessageRoleUser,
                                    Content: "Current codebase:" + parts[0] + "\n\n" + content,
                                })
                            }
                        }
                    }
                }
            } else {
                // Simple format without history
                if strings.Contains(rest, "\nHuman: ") {
                    parts := strings.SplitN(rest, "\nHuman: ", 2)
                    if len(parts) == 2 {
                        content := strings.TrimSuffix(parts[1], "\nAssistant: ")
                        content = strings.TrimSpace(content)
                        messages = append(messages, openai.ChatCompletionMessage{
                            Role:    openai.ChatMessageRoleUser,
                            Content: "Current codebase:" + parts[0] + "\n\n" + content,
                        })
                    }
                }
            }
        }
    } else {
        // Simple prompt without system message
        messages = append(messages, openai.ChatCompletionMessage{
            Role:    openai.ChatMessageRoleUser,
            Content: prompt,
        })
    }

    // SAFETY NET: Filter out any empty messages that might have slipped through
    var filteredMessages []openai.ChatCompletionMessage
    for _, msg := range messages {
        if strings.TrimSpace(msg.Content) != "" {
            filteredMessages = append(filteredMessages, msg)
        } else {
            utils.Log.Warning("Filtering out empty %s message in parsePrompt", msg.Role);
        }
    }
    
    return filteredMessages
}

// ValidateAuth validates OpenAI API key
func (o *OpenAIProvider) ValidateAuth() error {
    // Test with a minimal request
    ctx := context.TODO()
    
    req := openai.ChatCompletionRequest{
        Model: o.modelID,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: "Hi",
            },
        },
        MaxTokens: 5,
    }
    
    _, err := o.client.CreateChatCompletion(ctx, req)
    if err != nil {
        return fmt.Errorf("OpenAI authentication failed: %w", err)
    }
    
    utils.Log.Info("OpenAI authentication successful")
    return nil
}

// GetModelInfo returns information about the current model
func (o *OpenAIProvider) GetModelInfo() ModelInfo {
    maxTokens := 4096
    if o.modelID == "gpt-4-turbo-preview" || o.modelID == "gpt-4-1106-preview" {
        maxTokens = 128000
    }
    
    return ModelInfo{
        ModelID:   o.modelID,
        Endpoint:  "openai",
        MaxTokens: maxTokens,
    }
}