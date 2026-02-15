package planexecute

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/biome/agent-core/packages/agent/core"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-core/packages/stream"
	"github.com/biome/agent-mind/provider"
)

// Default returns the plan-and-execute orchestrator. Opt-in only; do not register as default.
func Default() core.Orchestrator {
	return &PlanExecuteOrchestrator{}
}

// PlanExecuteOrchestrator runs one turn: plan (LLM) -> execute steps (tools) -> synthesize (LLM).
type PlanExecuteOrchestrator struct{}

// Run runs one plan-and-execute turn: planning call, execution loop, synthesis call.
func (o *PlanExecuteOrchestrator) Run(ctx context.Context, agent *core.Agent, userMessage types.UserMessage, eventStream *stream.EventStream[core.AgentEvent, []types.AgentMessage]) {
	_ = userMessage // already appended by Prompt() before Run()
	startTime := time.Now()
	state := agent.State()
	config := agent.Config()

	eventStream.Push(core.AgentEvent{
		Type:    core.EventTurnStart,
		Payload: core.TurnStartPayload{Timestamp: time.Now().UnixMilli()},
	})

	if config.Provider == nil {
		eventStream.EndWithError(fmt.Errorf("plan-and-execute: no provider configured"))
		return
	}

	// --- Planning phase ---
	agentContext := state.ToContext().Clone()
	providerMessages, err := buildProviderMessages(ctx, agentContext, config.Pipeline)
	if err != nil {
		eventStream.EndWithError(fmt.Errorf("plan-and-execute: build messages: %w", err))
		return
	}

	planningPrompt := buildPlanningPrompt(agentContext.SystemPrompt, config.Tools)
	planReq := provider.CompletionRequest{
		SystemPrompt: planningPrompt,
		Messages:     providerMessages,
		Temperature:  0.3,
		MaxTokens:    2000,
		Tools:        nil, // no tools for plan phase; we want JSON in text
	}

	planResp, err := config.Provider.Complete(ctx, planReq)
	if err != nil {
		agent.SetError(fmt.Sprintf("%v", err))
		eventStream.EndWithError(fmt.Errorf("plan-and-execute: planning call: %w", err))
		return
	}

	plan, err := parsePlan(planResp.Text)
	if err != nil {
		agent.SetError(fmt.Sprintf("failed to parse plan: %v", err))
		// Empty plan: skip execution, go to synthesis
		plan = &Plan{Steps: nil}
	}

	stepCount := len(plan.Steps)
	stepsInfo := make([]core.PlanStepInfo, 0, stepCount)
	for _, s := range plan.Steps {
		stepsInfo = append(stepsInfo, core.PlanStepInfo{Tool: s.Tool, Args: s.Args})
	}
	eventStream.Push(core.AgentEvent{
		Type:    core.EventPlanCreated,
		Payload: core.PlanCreatedPayload{StepCount: stepCount, Steps: stepsInfo},
	})

	// Append assistant "plan" message for context (optional; synthesis will see tool results)
	if planResp.Text != "" {
		providerName := ""
		if config.Provider != nil {
			providerName = config.Provider.Name()
		}
		modelUsed := planResp.Model
		if modelUsed == "" {
			modelUsed = "unknown"
		}
		state.Messages = append(state.Messages, types.AssistantMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: planResp.Text}},
			Provider:   providerName,
			Model:      modelUsed,
			StopReason: types.StopReasonStop,
		})
	}

	// --- Execution phase ---
	for i, step := range plan.Steps {
		if ctx.Err() != nil {
			eventStream.EndWithError(ctx.Err())
			return
		}
		toolCallId := fmt.Sprintf("plan-step-%d", i)
		toolCall := core.ToolCallRequest{
			ToolCallId: toolCallId,
			ToolName:   step.Tool,
			Args:       step.Args,
		}

		eventStream.Push(core.AgentEvent{
			Type: core.EventPlanStepStart,
			Payload: core.PlanStepStartPayload{
				Index:     i,
				StepCount: stepCount,
				Tool:      step.Tool,
				Args:      step.Args,
			},
		})
		eventStream.Push(core.AgentEvent{
			Type: core.EventToolCall,
			Payload: core.ToolCallPayload{
				ToolCallId: toolCall.ToolCallId,
				ToolName:   toolCall.ToolName,
				Args:       toolCall.Args,
			},
		})

		toolResult := agent.ExecuteTool(ctx, toolCall)

		eventStream.Push(core.AgentEvent{
			Type: core.EventToolResult,
			Payload: core.ToolResultPayload{
				ToolCallId: toolCall.ToolCallId,
				ToolName:   toolCall.ToolName,
				Result:     toolResult.Details,
				Error:      core.ToolResultError(toolResult),
			},
		})
		eventStream.Push(core.AgentEvent{
			Type: core.EventPlanStepEnd,
			Payload: core.PlanStepEndPayload{
				Index:     i,
				StepCount: stepCount,
				Tool:      step.Tool,
				Result:    toolResult.Details,
				Error:     core.ToolResultError(toolResult),
			},
		})

		state.Messages = append(state.Messages, toolResult)
	}

	// --- Synthesis phase ---
	synthContext := state.ToContext().Clone()
	synthMessages, err := buildProviderMessages(ctx, synthContext, config.Pipeline)
	if err != nil {
		eventStream.EndWithError(fmt.Errorf("plan-and-execute: synthesis messages: %w", err))
		return
	}

	synthPrompt := buildSynthesisPrompt(synthContext.SystemPrompt)
	synthReq := provider.CompletionRequest{
		SystemPrompt: synthPrompt,
		Messages:     synthMessages,
		Temperature:  0.5,
		MaxTokens:    1000,
		Tools:        nil,
	}

	synthResp, err := config.Provider.Complete(ctx, synthReq)
	if err != nil {
		agent.SetError(fmt.Sprintf("%v", err))
		eventStream.EndWithError(fmt.Errorf("plan-and-execute: synthesis call: %w", err))
		return
	}

	responseText := synthResp.Text
	if responseText == "" {
		responseText = "I've completed the steps."
	}

	state.IsStreaming = true
	words := strings.Split(responseText, " ")
	for i, word := range words {
		eventStream.Push(core.AgentEvent{
			Type: core.EventTextDelta,
			Payload: core.TextDeltaPayload{
				Text:  word + " ",
				Index: i,
			},
		})
	}
	state.IsStreaming = false
	state.StreamMessage = nil

	providerName := ""
	if config.Provider != nil {
		providerName = config.Provider.Name()
	}
	modelUsed := synthResp.Model
	if modelUsed == "" {
		modelUsed = "unknown"
	}
	assistantMessage := types.AssistantMessage{
		Content:    []types.ContentBlock{types.TextContent{Text: responseText}},
		Provider:   providerName,
		Model:      modelUsed,
		StopReason: types.StopReasonStop,
	}
	state.Messages = append(state.Messages, assistantMessage)

	duration := time.Since(startTime).Milliseconds()
	eventStream.Push(core.AgentEvent{
		Type: core.EventTurnEnd,
		Payload: core.TurnEndPayload{
			Message:  assistantMessage,
			Duration: duration,
		},
	})

	eventStream.End(state.Messages)
}

// buildProviderMessages returns messages for the provider from agent context, using pipeline if set.
func buildProviderMessages(ctx context.Context, agentContext types.AgentContext, pipeline *transform.Pipeline) ([]types.Message, error) {
	if pipeline != nil {
		return pipeline.TransformContext(ctx, agentContext)
	}
	return types.ConvertToLLM(agentContext.Messages), nil
}

// buildPlanningPrompt returns the system prompt for the planning call (tool list + JSON format).
func buildPlanningPrompt(basePrompt string, toolRegistry *tools.ToolRegistry) string {
	if basePrompt == "" {
		basePrompt = "You are a helpful AI assistant."
	}
	toolList := "No tools available."
	if toolRegistry != nil {
		names := toolRegistry.ListTools()
		if len(names) > 0 {
			var descs []string
			for _, t := range toolRegistry.All() {
				descs = append(descs, t.Name()+": "+t.Description())
			}
			toolList = strings.Join(descs, "\n")
		}
	}
	return basePrompt + "\n\nYou must output a JSON plan for the user's request using ONLY the following tools:\n" +
		toolList + "\n\n" +
		`Output a single JSON object with this exact format (no other text):
		{"steps":[{"tool":"<tool_name>","args":{...}}, ...]}
		Use only the tool names listed above. If no tools are needed, use: {"steps":[]}.`
}

// buildSynthesisPrompt returns the system prompt for the synthesis call.
func buildSynthesisPrompt(basePrompt string) string {
	if basePrompt == "" {
		basePrompt = "You are a helpful AI assistant."
	}
	return basePrompt + "\n\nYou have executed a plan and received tool results. Provide a concise, natural language summary for the user based on the conversation and the tool results. Do not repeat raw JSON or tool internals."
}

var jsonBlockRE = regexp.MustCompile("(?s)\\s*```(?:json)?\\s*([\\s\\S]*?)```\\s*")

// parsePlan extracts a Plan from the LLM response text (JSON only; optional markdown code block stripped).
func parsePlan(text string) (*Plan, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return &Plan{Steps: nil}, nil
	}
	// Strip markdown code block if present
	if m := jsonBlockRE.FindStringSubmatch(trimmed); len(m) > 1 {
		trimmed = strings.TrimSpace(m[1])
	}
	var plan Plan
	if err := json.Unmarshal([]byte(trimmed), &plan); err != nil {
		return nil, err
	}
	if plan.Steps == nil {
		plan.Steps = []PlanStep{}
	}
	return &plan, nil
}
