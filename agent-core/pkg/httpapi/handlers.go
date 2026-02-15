package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/biome/agent-core/packages/agent/core"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-core/packages/stream"
	"github.com/biome/agent-mind/provider"
)

// Server wraps the agent and provider. Tools are supplied per request via tool configs
// in POST /agent/prompt; the framework does not ship prebuilt tools.
type Server struct {
	defaultProvider provider.Provider
	httpClient      *http.Client
}

// NewServer creates a new HTTP API server. Tools are passed per request in the prompt body (Tools field).
func NewServer(defaultProvider provider.Provider) *Server {
	return &Server{
		defaultProvider: defaultProvider,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// toolDefinitionToConfig converts legacy ToolDefinition to tools.ToolConfig for backward compatibility.
func toolDefinitionToConfig(def ToolDefinition) tools.ToolConfig {
	cfg := tools.ToolConfig{
		Type:         "http",
		Name:         def.Name,
		Description:  def.Description,
		Parameters:   def.Parameters,
		Endpoint:     def.Endpoint,
		Method:       def.Method,
		ResponsePath: def.ResponsePath,
		ResponseMap:  def.ResponseMap,
	}
	if def.Auth != nil {
		cfg.Auth = &tools.HTTPAuthConfig{
			Type:   def.Auth.Type,
			APIKey: def.Auth.APIKey,
			Header: def.Auth.Header,
		}
	}
	return cfg
}

// AuthConfig represents authentication configuration for HTTP tools
type AuthConfig struct {
	Type   string `json:"type"`   // "apikey", "bearer", "basic"
	APIKey string `json:"apikey"` // For apikey or bearer
	Header string `json:"header"` // Header name for apikey (e.g., "X-API-Key")
}

// ToolDefinition represents a complete HTTP tool definition
type ToolDefinition struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Parameters   map[string]interface{} `json:"parameters"`   // JSON Schema
	Endpoint     string                 `json:"endpoint"`    // HTTP endpoint to call
	Method       string                 `json:"method"`     // HTTP method (default: POST)
	Auth         *AuthConfig            `json:"auth"`        // Authentication config
	ResponsePath string                 `json:"response_path,omitempty"` // e.g. "results[0]" to extract from API response
	ResponseMap  map[string]string      `json:"response_map,omitempty"`   // output key -> input key, e.g. {"country":"country_code"}
}

// PromptRequest represents a prompt request. Tools are supplied as tool configs (this agent has access to these tools).
type PromptRequest struct {
	Message      string                   `json:"message"`
	Stream       bool                     `json:"stream"`
	SystemPrompt string                   `json:"system_prompt,omitempty"`
	Messages     []map[string]interface{} `json:"messages,omitempty"`
	Temperature  float64                  `json:"temperature,omitempty"`
	MaxTokens    int                      `json:"max_tokens,omitempty"`
	Model        string                   `json:"model,omitempty"`
	Tools        []tools.ToolConfig       `json:"tools,omitempty"`
	// CustomTools is deprecated: use Tools with type "http" instead. Kept for backward compatibility.
	CustomTools []ToolDefinition `json:"custom_tools,omitempty"`
}

// RegisterToolRequest for POST /tools/register
type RegisterToolRequest struct {
	Tool ToolDefinition `json:"tool"`
}

// RegisterToolHandler handles POST /tools/register.
// Deprecated: tools are supplied per request in POST /agent/prompt via the "tools" field (tool configs).
// This endpoint is kept for compatibility but no longer registers tools; pass tools in each prompt request instead.
func (s *Server) RegisterToolHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusGone)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   "deprecated: use POST /agent/prompt with \"tools\" array (tool configs) instead of registering tools",
	})
}

// ListToolsHandler handles GET /tools. Tools are supplied per request in POST /agent/prompt; no global registry.
func (s *Server) ListToolsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "tools are supplied per request in POST /agent/prompt via the \"tools\" array (tool configs)",
		"tools":   []string{},
	})
}

// PromptHandler handles POST /agent/prompt
func (s *Server) PromptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Build tool configs: primary Tools field + legacy CustomTools converted to http ToolConfig
	configs := make([]tools.ToolConfig, 0, len(req.Tools)+len(req.CustomTools))
	configs = append(configs, req.Tools...)
	for _, def := range req.CustomTools {
		configs = append(configs, toolDefinitionToConfig(def))
	}
	toolRegistry, err := tools.NewRegistryFromConfig(configs, s.httpClient)
	if err != nil {
		http.Error(w, "invalid tool config: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build agent config
	agentConfig := s.buildAgentConfig(req, toolRegistry)

	// Create agent
	agent := core.NewAgent(agentConfig)

	// Create user message
	userMsg := types.UserMessage{
		Content: []types.ContentBlock{
			types.TextContent{Text: req.Message},
		},
	}

	// Get event stream
	eventStream := agent.Prompt(context.Background(), userMsg)

	// Stream or collect
	if req.Stream {
		s.streamEvents(w, eventStream)
	} else {
		s.collectEvents(w, eventStream)
	}
}

// buildAgentConfig creates agent configuration from request and the registry built from tool configs.
func (s *Server) buildAgentConfig(req PromptRequest, toolRegistry *tools.ToolRegistry) core.AgentConfig {
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful AI assistant."
	}
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	return core.AgentConfig{
		SystemPrompt: systemPrompt,
		Pipeline:     pipeline,
		Tools:        toolRegistry,
		Provider:     s.defaultProvider,
	}
}

// streamEvents streams via SSE
func (s *Server) streamEvents(w http.ResponseWriter, eventStream *stream.EventStream[core.AgentEvent, []types.AgentMessage]) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for event := range eventStream.Events() {
		data, err := json.Marshal(map[string]interface{}{
			"type":    event.Type,
			"payload": event.Payload,
		})
		if err != nil {
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: {\"type\":\"done\"}\n\n")
	flusher.Flush()
}

// collectEvents collects and returns all events
func (s *Server) collectEvents(w http.ResponseWriter, eventStream *stream.EventStream[core.AgentEvent, []types.AgentMessage]) {
	events := []map[string]interface{}{}

	for event := range eventStream.Events() {
		events = append(events, map[string]interface{}{
			"type":    event.Type,
			"payload": event.Payload,
		})
	}

	messages, _ := eventStream.Result()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events":   events,
		"messages": messages,
	})
}

// HealthHandler handles GET /health
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"agent":  "ready",
	})
}

// CORS middleware
func (s *Server) CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
