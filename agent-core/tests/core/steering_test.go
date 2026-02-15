package core_test

import (
	"context"
	"testing"

	examplestools "github.com/biome/agent-core/examples/tools"
	"github.com/biome/agent-core/packages/agent/core"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/provider"
)

// --- Queue Tests ---

func TestNewFollowUpQueue(t *testing.T) {
	queue := core.NewFollowUpQueue()
	if queue == nil {
		t.Fatal("Expected queue to be created")
	}
	if !queue.IsEmpty() {
		t.Error("New queue should be empty")
	}
	if queue.Size() != 0 {
		t.Errorf("New queue size should be 0, got %d", queue.Size())
	}
}

func TestQueueEnqueueDequeue(t *testing.T) {
	queue := core.NewFollowUpQueue()
	queue.Enqueue(core.FollowUpItem{Type: "tool_call", Payload: "test1"})
	queue.Enqueue(core.FollowUpItem{Type: "think", Payload: "test2"})

	if queue.Size() != 2 {
		t.Errorf("Expected size 2, got %d", queue.Size())
	}

	first, ok := queue.Dequeue()
	if !ok {
		t.Error("Expected successful dequeue")
	}
	if first.Type != "tool_call" {
		t.Errorf("Expected type 'tool_call', got %s", first.Type)
	}
	if queue.Size() != 1 {
		t.Errorf("Expected size 1, got %d", queue.Size())
	}
}

func TestQueueDequeueEmpty(t *testing.T) {
	queue := core.NewFollowUpQueue()
	_, ok := queue.Dequeue()
	if ok {
		t.Error("Expected false when dequeuing from empty queue")
	}
}

func TestQueueClear(t *testing.T) {
	queue := core.NewFollowUpQueue()
	queue.Enqueue(core.FollowUpItem{Type: "test"})
	queue.Enqueue(core.FollowUpItem{Type: "test"})
	queue.Clear()

	if !queue.IsEmpty() {
		t.Error("Queue should be empty after clear")
	}
}

func TestQueueDrain(t *testing.T) {
	queue := core.NewFollowUpQueue()
	queue.Enqueue(core.FollowUpItem{Type: "a"})
	queue.Enqueue(core.FollowUpItem{Type: "b"})
	queue.Enqueue(core.FollowUpItem{Type: "c"})

	items := queue.Drain()
	if len(items) != 3 {
		t.Errorf("Expected 3 items from drain, got %d", len(items))
	}
	if !queue.IsEmpty() {
		t.Error("Queue should be empty after drain")
	}
}

// --- Steering Types Tests ---

func TestSteeringModeConstants(t *testing.T) {
	if core.SteeringModeRespond != "respond" {
		t.Errorf("Expected 'respond', got %s", core.SteeringModeRespond)
	}
	if core.SteeringModeSteer != "steer" {
		t.Errorf("Expected 'steer', got %s", core.SteeringModeSteer)
	}
}

func TestSteeringDecisionStruct(t *testing.T) {
	decision := core.SteeringDecision{
		Mode:         core.SteeringModeSteer,
		ToolCalls:    []core.ToolCallRequest{{ToolCallId: "1", ToolName: "calc", Args: map[string]interface{}{"x": 1}}},
		ThinkingText: "I'll calculate",
		Response:     "",
	}

	if decision.Mode != core.SteeringModeSteer {
		t.Error("Mode not set correctly")
	}
	if len(decision.ToolCalls) != 1 {
		t.Error("ToolCalls not set correctly")
	}
}

func TestToolCallRequest(t *testing.T) {
	req := core.ToolCallRequest{
		ToolCallId: "call_123",
		ToolName:   "calculator",
		Args:       map[string]interface{}{"expression": "2+2"},
	}

	if req.ToolCallId != "call_123" {
		t.Error("ToolCallId not set")
	}
	if req.ToolName != "calculator" {
		t.Error("ToolName not set")
	}
	if req.Args["expression"] != "2+2" {
		t.Error("Args not set")
	}
}

// --- Agent Config Tests ---

func TestAgentConfigMinimal(t *testing.T) {
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful",
	})

	if agent == nil {
		t.Fatal("Expected agent to be created")
	}
	if len(agent.Messages()) != 0 {
		t.Error("New agent should have no messages")
	}
}

func TestAgentResetClearsHistory(t *testing.T) {
	agent := core.NewAgent(core.AgentConfig{})

	// Add a message via Prompt
	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "test"}},
	})
	for range stream.Events() {
	}

	if len(agent.Messages()) == 0 {
		t.Error("Expected messages after prompt")
	}

	agent.Reset()

	if len(agent.Messages()) != 0 {
		t.Error("Agent should have no messages after reset")
	}
}

// --- Mock Provider for Agent Tests ---

type mockRespondProvider struct{}

func (m *mockRespondProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return &provider.CompletionResponse{Text: "Hello!"}, nil
}

func (m *mockRespondProvider) Stream(ctx context.Context, req provider.CompletionRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent, 1)
	ch <- provider.StreamEvent{Type: provider.EventDone}
	close(ch)
	return ch, nil
}

func (m *mockRespondProvider) Name() string     { return "mockRespond" }
func (m *mockRespondProvider) Models() []string { return nil }

func TestAgentRespondMode(t *testing.T) {
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "Test",
		Provider:     &mockRespondProvider{},
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hi"}},
	})

	var gotTurnStart, gotTurnEnd, gotSteering bool
	for event := range stream.Events() {
		switch event.Type {
		case core.EventTurnStart:
			gotTurnStart = true
		case core.EventTurnEnd:
			gotTurnEnd = true
		case core.EventSteeringMode:
			gotSteering = true
			p := event.Payload.(core.SteeringModePayload)
			if p.Mode != "respond" {
				t.Errorf("Expected respond mode, got %s", p.Mode)
			}
		}
	}

	if !gotTurnStart {
		t.Error("Missing turn_start event")
	}
	if !gotTurnEnd {
		t.Error("Missing turn_end event")
	}
	if !gotSteering {
		t.Error("Missing steering_mode event")
	}
}

// --- Mock Provider with Tool Calls ---

type mockToolProvider struct {
	callCount int
}

func (m *mockToolProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	m.callCount++
	if m.callCount == 1 {
		return &provider.CompletionResponse{
			ToolCalls: []provider.ToolCallResponse{
				{ID: "call_1", Name: "calculator", Arguments: map[string]interface{}{"expression": "2+2"}},
			},
		}, nil
	}
	return &provider.CompletionResponse{Text: "The answer is 4"}, nil
}

func (m *mockToolProvider) Stream(ctx context.Context, req provider.CompletionRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent, 1)
	ch <- provider.StreamEvent{Type: provider.EventDone}
	close(ch)
	return ch, nil
}

func (m *mockToolProvider) Name() string     { return "mockTool" }
func (m *mockToolProvider) Models() []string { return nil }

func TestAgentToolExecution(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "Test",
		Provider:     &mockToolProvider{},
		Tools:        registry,
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Calculate 2+2"}},
	})

	var toolCalls, toolResults int
	for event := range stream.Events() {
		switch event.Type {
		case core.EventToolCall:
			toolCalls++
		case core.EventToolResult:
			toolResults++
		}
	}

	if toolCalls != 1 {
		t.Errorf("Expected 1 tool call, got %d", toolCalls)
	}
	if toolResults != 1 {
		t.Errorf("Expected 1 tool result, got %d", toolResults)
	}
}

func TestAgentSequentialToolCalls(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	mock := &mockMultiToolProvider{}
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "Test",
		Provider:     mock,
		Tools:        registry,
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Calculate"}},
	})

	var toolNames []string
	for event := range stream.Events() {
		if event.Type == core.EventToolCall {
			p := event.Payload.(core.ToolCallPayload)
			toolNames = append(toolNames, p.ToolName)
		}
	}

	if len(toolNames) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(toolNames))
	}
}

type mockMultiToolProvider struct {
	callCount int
}

func (m *mockMultiToolProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	m.callCount++
	if m.callCount == 1 {
		return &provider.CompletionResponse{
			ToolCalls: []provider.ToolCallResponse{
				{ID: "c1", Name: "calculator", Arguments: map[string]interface{}{"expression": "1+1"}},
				{ID: "c2", Name: "calculator", Arguments: map[string]interface{}{"expression": "2+2"}},
			},
		}, nil
	}
	return &provider.CompletionResponse{Text: "Done"}, nil
}

func (m *mockMultiToolProvider) Stream(ctx context.Context, req provider.CompletionRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent, 1)
	ch <- provider.StreamEvent{Type: provider.EventDone}
	close(ch)
	return ch, nil
}

func (m *mockMultiToolProvider) Name() string     { return "mockMulti" }
func (m *mockMultiToolProvider) Models() []string { return nil }

// --- Event Types Tests ---

func TestEventTypeConstants(t *testing.T) {
	events := []string{
		core.EventTurnStart,
		core.EventTextDelta,
		core.EventThinking,
		core.EventSteeringMode,
		core.EventToolCall,
		core.EventToolResult,
		core.EventTurnEnd,
	}

	for _, e := range events {
		if e == "" {
			t.Error("Event type constant is empty")
		}
	}
}
