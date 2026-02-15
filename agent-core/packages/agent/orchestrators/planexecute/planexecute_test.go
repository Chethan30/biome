package planexecute_test

import (
	"context"
	"testing"

	examplestools "github.com/biome/agent-core/examples/tools"
	"github.com/biome/agent-core/packages/agent/core"
	"github.com/biome/agent-core/packages/agent/orchestrators/planexecute"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/provider"
)

// --- parsePlan (via parsePlan - we test via Run or export parsePlan for test)
// parsePlan is unexported; we test it indirectly via Run with mock that returns known JSON.
// We add a small test that verifies the orchestrator runs and emits expected events.

func TestParsePlanEmpty(t *testing.T) {
	// Test by running full orchestrator with mock that returns {"steps":[]}
	mock := &mockPlanExecuteProvider{
		planResponse:    `{"steps":[]}`,
		synthesisResponse: "No tools needed.",
	}
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful.",
		Provider:     mock,
		Tools:        registry,
		Orchestrator: planexecute.Default(),
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "What is 2+2?"}},
	})

	var turnStart, turnEnd, textDelta bool
	var toolCalls int
	for event := range stream.Events() {
		switch event.Type {
		case core.EventTurnStart:
			turnStart = true
		case core.EventTurnEnd:
			turnEnd = true
		case core.EventTextDelta:
			textDelta = true
		case core.EventToolCall:
			toolCalls++
		}
	}

	if !turnStart {
		t.Error("Expected turn_start event")
	}
	if !turnEnd {
		t.Error("Expected turn_end event")
	}
	if !textDelta {
		t.Error("Expected text_delta event")
	}
	if toolCalls != 0 {
		t.Errorf("Expected 0 tool calls for empty plan, got %d", toolCalls)
	}
	if mock.completeCount != 2 {
		t.Errorf("Expected 2 Complete calls (plan + synthesis), got %d", mock.completeCount)
	}
}

func TestParsePlanSingleStep(t *testing.T) {
	mock := &mockPlanExecuteProvider{
		planResponse:      `{"steps":[{"tool":"calculator","args":{"expression":"15*3"}}]}`,
		synthesisResponse: "15 times 3 is 45.",
	}
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful.",
		Provider:     mock,
		Tools:        registry,
		Orchestrator: planexecute.Default(),
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "What is 15 * 3?"}},
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

func TestParsePlanMultipleSteps(t *testing.T) {
	mock := &mockPlanExecuteProvider{
		planResponse: `{"steps":[
			{"tool":"calculator","args":{"expression":"1+1"}},
			{"tool":"calculator","args":{"expression":"2+2"}}
		]}`,
		synthesisResponse: "Done.",
	}
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful.",
		Provider:     mock,
		Tools:        registry,
		Orchestrator: planexecute.Default(),
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Calculate 1+1 and 2+2"}},
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
	if len(toolNames) >= 2 && (toolNames[0] != "calculator" || toolNames[1] != "calculator") {
		t.Errorf("Expected calculator calls, got %v", toolNames)
	}
}

func TestPlanStepEvents(t *testing.T) {
	mock := &mockPlanExecuteProvider{
		planResponse:      `{"steps":[{"tool":"calculator","args":{"expression":"5*2"}}]}`,
		synthesisResponse: "The result is 10.",
	}
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful.",
		Provider:     mock,
		Tools:        registry,
		Orchestrator: planexecute.Default(),
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "What is 5 * 2?"}},
	})

	var planCreated core.PlanCreatedPayload
	var gotPlanCreated bool
	var stepStarts []core.PlanStepStartPayload
	var stepEnds []core.PlanStepEndPayload
	for event := range stream.Events() {
		switch event.Type {
		case core.EventPlanCreated:
			planCreated = event.Payload.(core.PlanCreatedPayload)
			gotPlanCreated = true
		case core.EventPlanStepStart:
			stepStarts = append(stepStarts, event.Payload.(core.PlanStepStartPayload))
		case core.EventPlanStepEnd:
			stepEnds = append(stepEnds, event.Payload.(core.PlanStepEndPayload))
		}
	}

	if !gotPlanCreated {
		t.Fatal("Expected plan_created event")
	}
	if planCreated.StepCount != 1 {
		t.Errorf("Expected StepCount 1, got %d", planCreated.StepCount)
	}
	if len(planCreated.Steps) != 1 || planCreated.Steps[0].Tool != "calculator" {
		t.Errorf("Expected one calculator step in plan, got %+v", planCreated.Steps)
	}
	if len(stepStarts) != 1 {
		t.Errorf("Expected 1 plan_step_start, got %d", len(stepStarts))
	}
	if len(stepEnds) != 1 {
		t.Errorf("Expected 1 plan_step_end, got %d", len(stepEnds))
	}
	if len(stepStarts) > 0 {
		if stepStarts[0].Index != 0 || stepStarts[0].StepCount != 1 || stepStarts[0].Tool != "calculator" {
			t.Errorf("Unexpected plan_step_start: %+v", stepStarts[0])
		}
	}
	if len(stepEnds) > 0 {
		if stepEnds[0].Index != 0 || stepEnds[0].StepCount != 1 || stepEnds[0].Tool != "calculator" {
			t.Errorf("Unexpected plan_step_end: %+v", stepEnds[0])
		}
	}
}

func TestParsePlanMarkdownBlock(t *testing.T) {
	// parsePlan is unexported; test via Run with response wrapped in ```json ... ```
	mock := &mockPlanExecuteProvider{
		planResponse: "Here is the plan:\n```json\n{\"steps\":[{\"tool\":\"calculator\",\"args\":{\"expression\":\"3+3\"}}]}\n```",
		synthesisResponse: "The result is 6.",
	}
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful.",
		Provider:     mock,
		Tools:        registry,
		Orchestrator: planexecute.Default(),
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "What is 3+3?"}},
	})

	var toolCalls int
	for event := range stream.Events() {
		if event.Type == core.EventToolCall {
			toolCalls++
		}
	}

	if toolCalls != 1 {
		t.Errorf("Expected 1 tool call when plan is in markdown block, got %d", toolCalls)
	}
}

func TestParsePlanInvalidJSONFallbackToEmpty(t *testing.T) {
	// When plan JSON is invalid, orchestrator treats as empty plan and continues to synthesis.
	mock := &mockPlanExecuteProvider{
		planResponse:      "I will use the calculator.",
		synthesisResponse: "I couldn't parse a plan, so here is a reply.",
	}
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful.",
		Provider:     mock,
		Tools:        registry,
		Orchestrator: planexecute.Default(),
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hello"}},
	})

	var toolCalls int
	for event := range stream.Events() {
		if event.Type == core.EventToolCall {
			toolCalls++
		}
	}

	if toolCalls != 0 {
		t.Errorf("Expected 0 tool calls when plan is invalid (fallback to empty), got %d", toolCalls)
	}
	_, err := stream.Result()
	if err != nil {
		t.Errorf("Expected no error when falling back to empty plan, got %v", err)
	}
}

func TestPlanExecuteNoProvider(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(&examplestools.CalculatorTool{})

	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful.",
		Provider:     nil,
		Tools:        registry,
		Orchestrator: planexecute.Default(),
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hi"}},
	})

	var errSeen bool
	for event := range stream.Events() {
		if event.Type == "error" || (event.Payload != nil && event.Type == core.EventTurnEnd) {
			// Stream may end with error
		}
		_ = event
	}
	_, err := stream.Result()
	if err != nil {
		errSeen = true
	}
	if !errSeen {
		t.Error("Expected stream to end with error when provider is nil")
	}
}

// mockPlanExecuteProvider returns plan JSON on first Complete, synthesis text on second.
type mockPlanExecuteProvider struct {
	completeCount     int
	planResponse     string
	synthesisResponse string
}

func (m *mockPlanExecuteProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	m.completeCount++
	if m.completeCount == 1 {
		return &provider.CompletionResponse{Text: m.planResponse, Model: "mock"}, nil
	}
	return &provider.CompletionResponse{Text: m.synthesisResponse, Model: "mock"}, nil
}

func (m *mockPlanExecuteProvider) Stream(ctx context.Context, req provider.CompletionRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent, 1)
	ch <- provider.StreamEvent{Type: provider.EventDone}
	close(ch)
	return ch, nil
}

func (m *mockPlanExecuteProvider) Name() string     { return "mockPlanExecute" }
func (m *mockPlanExecuteProvider) Models() []string { return nil }
