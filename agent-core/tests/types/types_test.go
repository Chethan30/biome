package types_test

import (
	"testing"

	"github.com/biome/agent-core/packages/agent/types"
)

func TestContentBlockInterface(t *testing.T) {
	tests := []struct {
		name     string
		block    types.ContentBlock
		expected string
	}{
		{"TextContent", types.TextContent{Text: "hello"}, "text"},
		{"ImageContent", types.ImageContent{Data: "base64data", MimeType: "image/png"}, "image"},
		{"ThinkingContent", types.ThinkingContent{Thinking: "reasoning..."}, "thinking"},
		{"ToolCallContent", types.ToolCallContent{ID: "1", Name: "tool", Arguments: nil}, "toolCall"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.block.ContentType(); got != tt.expected {
				t.Errorf("ContentType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestContentBlockPolymorphism(t *testing.T) {
	blocks := []types.ContentBlock{
		types.TextContent{Text: "hello"},
		types.ImageContent{Data: "data", MimeType: "image/jpeg"},
		types.ThinkingContent{Thinking: "thinking"},
		types.ToolCallContent{ID: "1", Name: "test", Arguments: map[string]string{"key": "value"}},
	}

	if len(blocks) != 4 {
		t.Errorf("Expected 4 blocks, got %d", len(blocks))
	}

	expectedTypes := []string{"text", "image", "thinking", "toolCall"}
	for i, block := range blocks {
		if got := block.ContentType(); got != expectedTypes[i] {
			t.Errorf("Block %d: ContentType() = %v, want %v", i, got, expectedTypes[i])
		}
	}
}

func TestStopReasonConstants(t *testing.T) {
	tests := []struct {
		name     string
		reason   types.StopReason
		expected string
	}{
		{"Stop", types.StopReasonStop, "stop"},
		{"Length", types.StopReasonLength, "length"},
		{"ToolUse", types.StopReasonToolUse, "toolUse"},
		{"Aborted", types.StopReasonAborted, "aborted"},
		{"Error", types.StopReasonError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.reason) != tt.expected {
				t.Errorf("StopReason = %v, want %v", tt.reason, tt.expected)
			}
		})
	}
}

func TestUserMessage(t *testing.T) {
	content := []types.ContentBlock{
		types.TextContent{Text: "Hello, world!"},
	}
	msg := types.UserMessage{
		Content: content,
	}

	if msg.Role() != "user" {
		t.Errorf("Role() = %v, want user", msg.Role())
	}

	if len(msg.Content) != 1 {
		t.Errorf("Content length = %v, want 1", len(msg.Content))
	}
}

func TestAssistantMessage(t *testing.T) {
	errorMsg := "test error"
	msg := types.AssistantMessage{
		Content:      []types.ContentBlock{types.TextContent{Text: "Response"}},
		API:          "v1",
		Provider:     "openai",
		Model:        "gpt-4",
		Usage:        types.UsageMetrics{Input: 10, Output: 20, TotalTokens: 30},
		StopReason:   types.StopReasonStop,
		ErrorMessage: &errorMsg,
	}

	if msg.Role() != "assistant" {
		t.Errorf("Role() = %v, want assistant", msg.Role())
	}

	if msg.Provider != "openai" {
		t.Errorf("Provider = %v, want openai", msg.Provider)
	}

	if msg.StopReason != types.StopReasonStop {
		t.Errorf("StopReason = %v, want stop", msg.StopReason)
	}

	if msg.ErrorMessage == nil || *msg.ErrorMessage != "test error" {
		t.Errorf("ErrorMessage = %v, want 'test error'", msg.ErrorMessage)
	}
}

func TestToolResultMessage(t *testing.T) {
	details := map[string]interface{}{"status": "success"}
	msg := types.ToolResultMessage{
		Content:    []types.ContentBlock{types.TextContent{Text: "Result"}},
		ToolCallID: "call_123",
		ToolName:   "read_file",
		Details:    details,
		IsError:    false,
	}

	if msg.Role() != "toolResult" {
		t.Errorf("Role() = %v, want toolResult", msg.Role())
	}

	if msg.ToolCallID != "call_123" {
		t.Errorf("ToolCallID = %v, want call_123", msg.ToolCallID)
	}

	if msg.IsError {
		t.Errorf("IsError = %v, want false", msg.IsError)
	}
}

func TestMessageInterface(t *testing.T) {
	var messages []types.Message

	messages = append(messages, types.UserMessage{})
	messages = append(messages, types.AssistantMessage{})
	messages = append(messages, types.ToolResultMessage{})

	expectedRoles := []string{"user", "assistant", "toolResult"}

	for i, msg := range messages {
		if msg.Role() != expectedRoles[i] {
			t.Errorf("Message %d: Role() = %v, want %v", i, msg.Role(), expectedRoles[i])
		}
	}
}

func TestAgentMessageInterface(t *testing.T) {
	var agentMessages []types.AgentMessage

	agentMessages = append(agentMessages, types.UserMessage{})
	agentMessages = append(agentMessages, types.AssistantMessage{})
	agentMessages = append(agentMessages, types.ToolResultMessage{})

	if len(agentMessages) != 3 {
		t.Errorf("Expected 3 agent messages, got %d", len(agentMessages))
	}
}

func TestConvertToLLM(t *testing.T) {
	messages := []types.AgentMessage{
		types.UserMessage{Content: []types.ContentBlock{types.TextContent{Text: "User msg"}}},
		types.AssistantMessage{Content: []types.ContentBlock{types.TextContent{Text: "Assistant msg"}}},
		types.ToolResultMessage{Content: []types.ContentBlock{types.TextContent{Text: "Tool result"}}, ToolCallID: "1"},
	}

	llmMessages := types.ConvertToLLM(messages)

	if len(llmMessages) != 3 {
		t.Errorf("ConvertToLLM: Expected 3 messages, got %d", len(llmMessages))
	}

	expectedRoles := []string{"user", "assistant", "toolResult"}
	for i, msg := range llmMessages {
		if msg.Role() != expectedRoles[i] {
			t.Errorf("Message %d: Role() = %v, want %v", i, msg.Role(), expectedRoles[i])
		}
	}
}

// Custom message for testing filtering
type CustomNotificationMessage struct {
	text string
}

func (c CustomNotificationMessage) Role() string     { return "notification" }
func (c CustomNotificationMessage) Timestamp() int64 { return 0 }

func TestConvertToLLMFiltersCustomMessages(t *testing.T) {
	messages := []types.AgentMessage{
		types.UserMessage{Content: []types.ContentBlock{types.TextContent{Text: "User"}}},
		CustomNotificationMessage{text: "Notification"},
		types.AssistantMessage{Content: []types.ContentBlock{types.TextContent{Text: "Assistant"}}},
	}

	llmMessages := types.ConvertToLLM(messages)

	if len(llmMessages) != 2 {
		t.Errorf("Expected 2 LLM messages (custom filtered), got %d", len(llmMessages))
	}
}

func TestAgentContextClone(t *testing.T) {
	original := types.AgentContext{
		SystemPrompt: "You are helpful",
		Messages: []types.AgentMessage{
			types.UserMessage{},
			types.AssistantMessage{},
		},
	}

	cloned := original.Clone()

	if cloned.SystemPrompt != original.SystemPrompt {
		t.Errorf("Cloned SystemPrompt = %v, want %v", cloned.SystemPrompt, original.SystemPrompt)
	}

	if len(cloned.Messages) != len(original.Messages) {
		t.Errorf("Cloned Messages length = %v, want %v", len(cloned.Messages), len(original.Messages))
	}

	// Verify deep copy
	cloned.Messages = append(cloned.Messages, types.ToolResultMessage{})

	if len(original.Messages) != 2 {
		t.Errorf("Original Messages modified! Length = %v, want 2", len(original.Messages))
	}
}

func TestCompleteConversationFlow(t *testing.T) {
	state := types.AgentState{
		SystemPrompt: "You are a helpful assistant",
		Messages:     []types.AgentMessage{},
		IsStreaming:  false,
	}

	// User sends message
	userMsg := types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hello!"}},
	}
	state.Messages = append(state.Messages, userMsg)

	// Assistant responds
	assistantMsg := types.AssistantMessage{
		Content:    []types.ContentBlock{types.TextContent{Text: "Hi there!"}},
		StopReason: types.StopReasonStop,
	}
	state.Messages = append(state.Messages, assistantMsg)

	// Convert to LLM messages
	ctx := state.ToContext()
	llmMessages := types.ConvertToLLM(ctx.Messages)

	if len(llmMessages) != 2 {
		t.Errorf("Expected 2 LLM messages, got %d", len(llmMessages))
	}

	if llmMessages[0].Role() != "user" {
		t.Errorf("First message should be user, got %v", llmMessages[0].Role())
	}
	if llmMessages[1].Role() != "assistant" {
		t.Errorf("Second message should be assistant, got %v", llmMessages[1].Role())
	}
}
