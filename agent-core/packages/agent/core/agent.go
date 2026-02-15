package core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-core/packages/stream"
	"github.com/biome/agent-mind/provider"
)

// GetSteeringMessagesFunc is called after each tool execution. If it returns non-empty
// messages, remaining queued tool calls are skipped and those messages are injected.
type GetSteeringMessagesFunc func() []types.AgentMessage

// GetFollowUpMessagesFunc is called when the agent would otherwise stop (LLM returned text).
// If it returns non-empty messages, they are added and another turn begins.
type GetFollowUpMessagesFunc func() []types.AgentMessage

// AgentConfig configures the agent.
type AgentConfig struct {
	SystemPrompt        string
	Pipeline            *transform.Pipeline
	Tools               *tools.ToolRegistry
	Provider            provider.Provider
	GetSteeringMessages GetSteeringMessagesFunc
	GetFollowUpMessages GetFollowUpMessagesFunc
	// SteeringInstruction is optional. When set, used as the initial steering instruction (instead of the default generic one). Use this to tell the model when/how to use tools without hardcoding tool names.
	SteeringInstruction string
	// Orchestrator drives the turn loop. Nil = default agentic loop (steering + tools + respond). Set to use another arrangement (e.g. ReAct, plan-execute).
	Orchestrator Orchestrator
}

// Agent manages conversation state and tool execution.
type Agent struct {
	config AgentConfig
	state  *types.AgentState
}

// newAgentState creates initial state from config.
func newAgentState(config AgentConfig) *types.AgentState {
	return &types.AgentState{
		SystemPrompt:      config.SystemPrompt,
		Tools:             config.Tools, // ToolRegistry implements types.ToolLister
		Messages:          []types.AgentMessage{},
		IsStreaming:       false,
		StreamMessage:     nil,
		PendingToolCalls:  make(map[string]bool),
		Error:             nil,
	}
}

// NewAgent creates a new agent with the given configuration.
func NewAgent(config AgentConfig) *Agent {
	return &Agent{
		config: config,
		state:  newAgentState(config),
	}
}

// Config returns the agent configuration (for use by orchestrators).
func (a *Agent) Config() AgentConfig {
	return a.config
}

// SteeringDecision asks the LLM for a steering decision (respond or steer with tool calls).
// Orchestrators call this with the current agent state; isFollowUp is true for iterations after the first in a turn.
func (a *Agent) SteeringDecision(ctx context.Context, isFollowUp bool) (SteeringDecision, error) {
	if a.config.Provider == nil {
		return SteeringDecision{Mode: SteeringModeRespond, Response: ""}, nil
	}
	snapshot := a.state.ToContext().Clone()
	return makeSteeringDecision(ctx, a.config.Provider, snapshot, a.config.Pipeline, a.config.Tools, isFollowUp, a.config.SteeringInstruction)
}

// SetError sets the agent state error (for use by orchestrators).
func (a *Agent) SetError(s string) {
	a.state.Error = &s
}

// Prompt starts a new conversation turn with the given user message.
// Returns an EventStream for consuming events and the final result.
func (a *Agent) Prompt(
	ctx context.Context,
	userMessage types.UserMessage,
) *stream.EventStream[AgentEvent, []types.AgentMessage] {

	eventStream := stream.NewEventStream[AgentEvent, []types.AgentMessage]()

	// Append user message before delegating to orchestrator
	a.state.Messages = append(a.state.Messages, userMessage)

	orch := a.config.Orchestrator
	if orch == nil {
		orch = defaultOrchestrator
	}
	if orch == nil {
		go func() {
			eventStream.EndWithError(fmt.Errorf("no orchestrator configured: set AgentConfig.Orchestrator or import github.com/biome/agent-core/packages/agent/orchestrators/agentic for the default agentic loop"))
		}()
		return eventStream
	}

	go func() {
		orch.Run(ctx, a, userMessage, eventStream)
	}()

	return eventStream
}

// ExecuteTool runs a single tool and returns the result message. Used by orchestrators.
func (a *Agent) ExecuteTool(ctx context.Context, toolCall ToolCallRequest) types.ToolResultMessage {
	if a.config.Tools == nil {
		return types.ToolResultMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: "tool registry not configured"}},
			ToolCallID: toolCall.ToolCallId,
			ToolName:   toolCall.ToolName,
			IsError:    true,
		}
	}

	tool, ok := a.config.Tools.Get(toolCall.ToolName)
	if !ok {
		return types.ToolResultMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: "tool not found: " + toolCall.ToolName}},
			ToolCallID: toolCall.ToolCallId,
			ToolName:   toolCall.ToolName,
			IsError:    true,
		}
	}

	result, err := tool.Execute(ctx, toolCall.Args)
	if err != nil {
		return types.ToolResultMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: err.Error()}},
			ToolCallID: toolCall.ToolCallId,
			ToolName:   toolCall.ToolName,
			IsError:    true,
		}
	}

	resultJSON, _ := json.Marshal(result)
	return types.ToolResultMessage{
		Content:    []types.ContentBlock{types.TextContent{Text: string(resultJSON)}},
		ToolCallID: toolCall.ToolCallId,
		ToolName:   toolCall.ToolName,
		Details:    result,
		IsError:    false,
	}
}

// ToolResultError extracts error text from a tool result if present. Used by orchestrators.
func ToolResultError(tr types.ToolResultMessage) string {
	if !tr.IsError {
		return ""
	}
	for _, block := range tr.Content {
		if tb, ok := block.(types.TextContent); ok {
			return tb.Text
		}
	}
	return "unknown error"
}

// Messages returns the current conversation history.
func (a *Agent) Messages() []types.AgentMessage {
	return a.state.Messages
}

// State returns the current agent state (for observability or testing).
func (a *Agent) State() *types.AgentState {
	return a.state
}

// Reset clears the conversation history and runtime flags; keeps system prompt.
func (a *Agent) Reset() {
	a.state = newAgentState(a.config)
}

