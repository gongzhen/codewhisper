package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gongzhen/codewhisper-go/internal/agent"
	"github.com/gongzhen/codewhisper-go/internal/utils"
	"github.com/gongzhen/codewhisper-go/pkg/config"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Server represents the CodeWhisper HTTP server
type Server struct {
	router     *mux.Router
	httpServer *http.Server
	port       int
	agent      *agent.Agent
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	s := &Server{
		router: mux.NewRouter(),
		port:   port,
	}

	// Setup routes
	s.setupRoutes()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(s.router)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
		// Set timeouts to 0 to disable them for streaming
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	utils.Log.Info("Server starting on http://localhost:%d", s.port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// API routes that the React frontend expects
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/folders", s.handleGetFolders).Methods("GET")
	api.HandleFunc("/current-model", s.handleGetCurrentModel).Methods("GET")
	api.HandleFunc("/model-id", s.handleGetModelID).Methods("GET")
	api.HandleFunc("/available-models", s.handleGetAvailableModels).Methods("GET")
	api.HandleFunc("/token-count", s.handleTokenCount).Methods("POST")
	api.HandleFunc("/default-included-folders", s.handleDefaultIncludedFolders).Methods("GET")
	api.HandleFunc("/set-model", s.handleSetModel).Methods("POST")
	api.HandleFunc("/model-settings", s.handleModelSettings).Methods("GET", "POST")
	api.HandleFunc("/model-capabilities", s.handleModelCapabilities).Methods("GET")

	// Streaming endpoints (critical for chat)
	s.router.HandleFunc("/codewhisper/stream", s.handleStreamChatLog).Methods("POST")
	s.router.HandleFunc("/codewhisper/stream_log", s.handleStreamChatLog).Methods("POST")

	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Add this to setupRoutes()
	s.router.HandleFunc("/test-sse", s.handleTestSSE).Methods("GET")

	// For production (after npm run build)
	templatesDir := "./templates"
	if _, err := os.Stat(filepath.Join(templatesDir, "index.html")); err == nil {
		// Serve static files
		s.router.PathPrefix("/static/").Handler(
			http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(templatesDir, "static")))),
		)

		// Serve testcases
		s.router.PathPrefix("/testcases/").Handler(
			http.StripPrefix("/testcases/", http.FileServer(http.Dir(filepath.Join(templatesDir, "testcases")))),
		)

		// Catch-all for React Router - must be last!
		s.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(templatesDir, "index.html"))
		})
	} else {
		// Fallback
		s.router.HandleFunc("/", s.handleRoot).Methods("GET")
	}
}

// Response types
type FolderStructure struct {
	TokenCount int                         `json:"token_count"`
	Children   map[string]FolderStructure `json:"children,omitempty"`
}

type ModelInfo struct {
	ModelID  string `json:"model_id"`
	Endpoint string `json:"endpoint"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Handlers

// Add this handler method
func (s *Server) handleTestSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send test messages
	for i := 0; i < 5; i++ {
		msg := map[string]interface{}{
			"ops": []map[string]interface{}{
				{
					"op":    "add",
					"path":  "/streamed_output_str/-",
					"value": fmt.Sprintf("Test message %d ", i),
				},
			},
		}

		data, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		time.Sleep(500 * time.Millisecond)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	// For now, just return a simple message
	// Later we'll serve the actual HTML template
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
        <html>
        <head><title>CodeWhisper</title></head>
        <body>
            <h1>CodeWhisper Assistant</h1>
            <p>Server is running on port %d</p>
        </body>
        </html>
    `, s.port)
}

func (s *Server) handleGetFolders(w http.ResponseWriter, r *http.Request) {
	userCodebaseDir := config.GetEnv(config.EnvUserCodebaseDir, ".")
	maxDepth := config.GetEnvInt(config.EnvMaxDepth, 15)

	// Get ignored patterns
	ignoredPatterns := utils.GetIgnoredPatterns(userCodebaseDir)

	// Build folder structure
	structure := s.getFolderStructure(userCodebaseDir, ignoredPatterns, maxDepth)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(structure)
}

func (s *Server) handleGetCurrentModel(w http.ResponseWriter, r *http.Request) {
	modelInfo := ModelInfo{
		ModelID:  config.GetEnv(config.EnvModel, "sonnet3.5-v2"),
		Endpoint: config.GetEnv(config.EnvEndpoint, "bedrock"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelInfo)
}

func (s *Server) handleGetModelID(w http.ResponseWriter, r *http.Request) {
	modelID := config.GetEnv(config.EnvModel, "sonnet3.5-v2")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"model_id": modelID})
}

func (s *Server) handleGetAvailableModels(w http.ResponseWriter, r *http.Request) {
	// Simplified for now - return bedrock models
	models := []map[string]string{
		{"id": "sonnet3.5-v2", "name": "Claude 3.5 Sonnet v2"},
		{"id": "sonnet3.5", "name": "Claude 3.5 Sonnet"},
		{"id": "opus", "name": "Claude 3 Opus"},
		{"id": "sonnet", "name": "Claude 3 Sonnet"},
		{"id": "haiku", "name": "Claude 3 Haiku"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func (s *Server) handleTokenCount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request"})
		return
	}

	// Use the tokenizer utility
	tokenCount := utils.CountTokens(req.Text)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"token_count": tokenCount})
}

func (s *Server) handleDefaultIncludedFolders(w http.ResponseWriter, r *http.Request) {
	// Return empty array for now, we'll implement folder selection later
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{
		"defaultIncludedFolders": []string{},
	})
}

func (s *Server) handleSetModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelID  string `json:"model_id"`
		Endpoint string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request"})
		return
	}

	// Update environment variables
	config.SetEnv(config.EnvModel, req.ModelID)
	config.SetEnv(config.EnvEndpoint, req.Endpoint)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleModelSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		settings := map[string]interface{}{
			"temperature":       config.GetEnv(config.EnvTemperature, "0.7"),
			"max_output_tokens": config.GetEnvInt(config.EnvMaxOutputTokens, 4096),
			"top_k":             config.GetEnvInt(config.EnvTopK, 40),
			"thinking_mode":     config.GetEnvBool(config.EnvThinkingMode, false),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	} else if r.Method == "POST" {
		var settings map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Update settings in environment
		if temp, ok := settings["temperature"].(float64); ok {
			config.SetEnv(config.EnvTemperature, fmt.Sprintf("%f", temp))
		}
		// ... handle other settings

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func (s *Server) handleStreamChatLog(w http.ResponseWriter, r *http.Request) {
	utils.Log.Info("Stream chat log request received")

	// Buffer the request body to prevent "read on closed body" errors
	body, err := io.ReadAll(r.Body)
    // Add this after reading the body
    utils.Log.Debug("Raw request body: %s", string(body))
    
	if err != nil {
		utils.Log.Error("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body)) // Restore the body for potential re-reads

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.Log.Error("Streaming not supported")
		return
	}

	flusher.Flush()

	if s.agent == nil {
		var err error
		s.agent, err = agent.NewAgent()
		if err != nil {
			fmt.Fprintf(w, "data: {\"error\": \"initialization_error\", \"detail\": \"Failed to initialize agent\"}\n\n")
			flusher.Flush()
			return
		}
	}

	var req agent.ChatRequest
	// Use the buffered body for decoding
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		utils.Log.Error("Failed to decode request: %v", err)
		fmt.Fprintf(w, "data: {\"error\": \"invalid_request\", \"detail\": \"Invalid request format\"}\n\n")
		flusher.Flush()
		return
	}

	utils.Log.Info("Processing stream request for question: %s", req.Input.Question)

	agentCtx := r.Context()

	eventChan, err := s.agent.StreamChat(agentCtx, req)
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\": \"stream_error\", \"detail\": \"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}

	messageCount := 0

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				utils.Log.Info("Event channel closed after %d messages", messageCount)
				return
			}

			if event.Error != "" {
				errorData := map[string]string{
					"error":  event.Error,
					"detail": event.Detail,
				}
				data, _ := json.Marshal(errorData)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				return
			}

			if event.Content != "" {
				messageCount++

				msg := map[string]interface{}{
					"ops": []map[string]interface{}{
						{
							"op":    "add",
							"path":  "/streamed_output_str/-",
							"value": event.Content,
						},
					},
				}

				data, _ := json.Marshal(msg)
				_, err := fmt.Fprintf(w, "data: %s\n\n", data)
				if err != nil {
					utils.Log.Error("Error writing to response: %v", err)
					return
				}
				flusher.Flush()

				if messageCount%50 == 0 {
					utils.Log.Info("Sent %d messages to client", messageCount)
				}
			}

		case <-agentCtx.Done():
			utils.Log.Info("Client disconnected after %d messages", messageCount)
			return
		}
	}
}

// Helper methods

func (s *Server) getFolderStructure(dir string, patterns []utils.PatternSource, maxDepth int) map[string]interface{} {
	shouldIgnore := utils.ParseGitignorePatterns(patterns)

	result := make(map[string]interface{})
	s.buildFolderStructureRecursive(dir, dir, result, shouldIgnore, 0, maxDepth)

	return result
}

func (s *Server) buildFolderStructureRecursive(baseDir, currentDir string, result map[string]interface{}, shouldIgnore utils.IgnoreMatcher, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(currentDir, entry.Name())

		if shouldIgnore(fullPath) {
			continue
		}

		if entry.IsDir() {
			dirNode := map[string]interface{}{
				"token_count": 0,
				"children":    make(map[string]interface{}),
			}

			s.buildFolderStructureRecursive(baseDir, fullPath, dirNode["children"].(map[string]interface{}), shouldIgnore, depth+1, maxDepth)

			tokenCount := s.calculateDirTokenCount(dirNode["children"].(map[string]interface{}))
			dirNode["token_count"] = tokenCount

			result[entry.Name()] = dirNode
		} else {
			if !utils.IsBinaryFile(fullPath) && !utils.IsImageFile(fullPath) {
				tokenCount := utils.CountTokensInFile(fullPath)
				result[entry.Name()] = map[string]interface{}{
					"token_count": tokenCount,
				}
			}
		}
	}
}

func (s *Server) calculateDirTokenCount(children map[string]interface{}) int {
	total := 0
	for _, child := range children {
		if node, ok := child.(map[string]interface{}); ok {
			if tokenCount, exists := node["token_count"]; exists {
				if count, ok := tokenCount.(int); ok {
					total += count
				}
			}
		}
	}
	return total
}

func (s *Server) handleModelCapabilities(w http.ResponseWriter, r *http.Request) {
	endpoint := r.URL.Query().Get("endpoint")

	capabilities := map[string]interface{}{
		"supportsStreaming":    true,
		"supportsSystemPrompt": true,
		"maxTokens":            200000,
		"supportsFunctions":    false,
		"supportsVision":       false,
	}

	switch endpoint {
	case "bedrock", "":
		capabilities["models"] = []map[string]interface{}{
			{"id": "sonnet3.5-v2", "name": "Claude 3.5 Sonnet v2", "maxTokens": 200000},
			{"id": "sonnet3.5", "name": "Claude 3.5 Sonnet", "maxTokens": 200000},
			{"id": "opus", "name": "Claude 3 Opus", "maxTokens": 200000},
			{"id": "sonnet", "name": "Claude 3 Sonnet", "maxTokens": 200000},
			{"id": "haiku", "name": "Claude 3 Haiku", "maxTokens": 200000},
		}
	case "openai":
		capabilities["models"] = []map[string]interface{}{
			{"id": "gpt-4", "name": "GPT-4", "maxTokens": 8192},
			{"id": "gpt-3.5-turbo", "name": "GPT-3.5 Turbo", "maxTokens": 4096},
		}
	case "google":
		capabilities["models"] = []map[string]interface{}{
			{"id": "gemini-pro", "name": "Gemini Pro", "maxTokens": 32768},
		}
	case "deepseek":
		capabilities["models"] = []map[string]interface{}{
			{"id": "deepseek-coder", "name": "DeepSeek Coder", "maxTokens": 16384},
		}
	default:
		capabilities["models"] = []map[string]interface{}{
			{"id": "default", "name": "Default Model", "maxTokens": 4096},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capabilities)
}