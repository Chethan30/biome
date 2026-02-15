package transform_test

import (
	"context"
	"testing"

	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-core/packages/agent/types"
)

// Custom message type for testing
type CustomNotification struct {
	text string
}

func (c CustomNotification) Role() string     { return "notification" }
func (c CustomNotification) Timestamp() int64 { return 0 }

func TestDefaultConvertToLLM(t *testing.T) {
	messages := []types.AgentMessage{
		types.UserMessage{
			Content: []types.ContentBlock{types.TextContent{Text: "Hello"}},
		},
		CustomNotification{text: "Typing..."}, // Should be filtered
		types.AssistantMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: "Hi"}},
			StopReason: types.StopReasonStop,
		},
		CustomNotification{text: "Another"}, // Should be filtered
		types.ToolResultMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: "result"}},
			ToolCallID: "123",
		},
	}

	result := transform.DefaultConvertToLLM(messages)

	if len(result) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(result))
	}

	expectedRoles := []string{"user", "assistant", "toolResult"}
	for i, msg := range result {
		if msg.Role() != expectedRoles[i] {
			t.Errorf("Message %d: expected role %s, got %s", i, expectedRoles[i], msg.Role())
		}
	}
}

func TestKeepRecentMessages(t *testing.T) {
	messages := make([]types.AgentMessage, 100)
	for i := 0; i < 100; i++ {
		messages[i] = types.UserMessage{
			Content: []types.ContentBlock{types.TextContent{Text: "msg"}},
		}
	}

	tr := transform.KeepRecentMessages(10)
	result, err := tr(context.Background(), messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(result))
	}
}

func TestKeepRecentMessagesUnderLimit(t *testing.T) {
	messages := []types.AgentMessage{
		types.UserMessage{Content: []types.ContentBlock{types.TextContent{Text: "1"}}},
		types.UserMessage{Content: []types.ContentBlock{types.TextContent{Text: "2"}}},
		types.UserMessage{Content: []types.ContentBlock{types.TextContent{Text: "3"}}},
	}

	tr := transform.KeepRecentMessages(10)
	result, err := tr(context.Background(), messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 messages (no pruning), got %d", len(result))
	}
}

func TestKeepFirstAndRecentMessages(t *testing.T) {
	messages := make([]types.AgentMessage, 100)
	for i := 0; i < 100; i++ {
		messages[i] = types.UserMessage{
			Content: []types.ContentBlock{types.TextContent{Text: "msg"}},
		}
	}

	tr := transform.KeepFirstandRecentMessages(10)
	result, err := tr(context.Background(), messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 11 {
		t.Errorf("Expected 11 messages, got %d", len(result))
	}
}

func TestChainTransform(t *testing.T) {
	messages := make([]types.AgentMessage, 200)
	for i := 0; i < 200; i++ {
		messages[i] = types.UserMessage{
			Content: []types.ContentBlock{types.TextContent{Text: "msg"}},
		}
	}

	tr := transform.ChainTrainsform(
		transform.KeepRecentMessages(100),
		transform.KeepFirstandRecentMessages(10),
	)

	result, err := tr(context.Background(), messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 11 {
		t.Errorf("Expected 11 messages, got %d", len(result))
	}
}

func TestPipelineNoTransform(t *testing.T) {
	messages := []types.AgentMessage{
		types.UserMessage{Content: []types.ContentBlock{types.TextContent{Text: "msg"}}},
		CustomNotification{text: "notif"},
		types.AssistantMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: "response"}},
			StopReason: types.StopReasonStop,
		},
	}

	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	result, err := pipeline.Transform(context.Background(), messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result))
	}
}

func TestPipelineWithTransform(t *testing.T) {
	messages := make([]types.AgentMessage, 50)
	for i := 0; i < 50; i++ {
		messages[i] = types.UserMessage{
			Content: []types.ContentBlock{types.TextContent{Text: "msg"}},
		}
	}

	pipeline := transform.NewPipeline(
		transform.KeepRecentMessages(10),
		transform.DefaultConvertToLLM,
	)

	result, err := pipeline.Transform(context.Background(), messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(result))
	}
}

func TestPipelineEmptyMessages(t *testing.T) {
	messages := []types.AgentMessage{}

	pipeline := transform.NewPipeline(
		transform.KeepRecentMessages(10),
		transform.DefaultConvertToLLM,
	)

	result, err := pipeline.Transform(context.Background(), messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(result))
	}
}
