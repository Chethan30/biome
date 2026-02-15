package provider

import (
	"github.com/biome/agent-core/packages/agent/types"
)

// StreamEvent represents a single event in the LLM stream
type StreamEvent struct {
	Type    string
	Delta   string
	Content interface{} // For EventToolDelta: *ToolCallStreamPayload; for EventToolCall: *ToolCallResponse
	Error   error
}

// ToolCallStreamPayload is sent with EventToolDelta for incremental tool-call args (partial JSON)
type ToolCallStreamPayload struct {
	Index     int
	ID        string
	Name      string
	Arguments map[string]interface{} // Best-effort parse of partial JSON
}

const (
	EventStart = "start"
	EventTextDelta = "text_delta"
	EventToolCall = "tool_call"
	EventToolDelta = "tool_delta"
	EventDone = "done"
	EventError = "error"
)

// CompletionRequest contains all parameters for an LLM completion
type CompletionRequest struct {
	Messages     []types.Message
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
	Tools        []Tool
}

// CompletionResponse represents a completed LLM response
type CompletionResponse struct {
	Text      string
	ToolCalls []ToolCallResponse // Structured tool calls from LLM
	Usage     UsageInfo
	// Model is the model identifier used for this completion (e.g. anthropic/claude-3-haiku). Empty if the provider does not report it.
	Model string
}

// ToolCallResponse represents a tool call from the LLM
type ToolCallResponse struct {
	ID        string                 // Unique ID for this tool call
	Name      string                 // Tool name
	Arguments map[string]interface{} // Parsed arguments
}

// UsageInfo tracks token usage
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Tool definition for function calling
type Tool struct {
	Name        string
	Description string
	Parameters  interface{}
}
