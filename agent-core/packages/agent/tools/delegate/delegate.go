package delegate

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/biome/agent-core/packages/agent/core"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/provider"
)

const (
	toolName       = "delegate"
	maxThinkingLen = 80
)

// Tool is a delegation tool that creates a sub-agent with a given persona and
// optional tool subset, runs one turn with the given task, and returns the
// sub-agent's final response. On failure, returns error and thinking trace so
// the master can reason and retry.
type Tool struct {
	provider provider.Provider
	pipeline *transform.Pipeline
	pool     *tools.ToolRegistry
}

// New returns a new delegation tool. The pool is the registry from which
// sub-agent tools are chosen (by name). If tool_names is empty when the tool
// runs, the sub-agent gets all tools from the pool.
func New(provider provider.Provider, pipeline *transform.Pipeline, pool *tools.ToolRegistry) *Tool {
	return &Tool{
		provider: provider,
		pipeline: pipeline,
		pool:     pool,
	}
}

// Name implements tools.Tool.
func (t *Tool) Name() string {
	return toolName
}

// Description implements tools.Tool.
func (t *Tool) Description() string {
	return "Delegate a task to a sub-agent with a dedicated persona (system prompt) and an optional set of tools. Use when a specialized agent should perform a focused task. Returns the sub-agent's response."
}

// Parameters implements tools.Tool.
func (t *Tool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.Property{
			"task": {
				Type:        "string",
				Description: "The instruction to send to the sub-agent as the single user message.",
			},
			"system_prompt": {
				Type:        "string",
				Description: "Persona or system context for the sub-agent (e.g. 'You are a math expert.').",
			},
			"tool_names": {
				Type:        "array",
				Description: "Optional. Names of tools the sub-agent can use. If empty or omitted, the sub-agent gets all tools from the pool.",
			},
			"context_excerpt": {
				Type:        "string",
				Description: "Optional. Additional context to append to the task for the sub-agent.",
			},
		},
		Required: []string{"task", "system_prompt"},
	}
}

// Execute implements tools.Tool.
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	task, _ := args["task"].(string)
	systemPrompt, _ := args["system_prompt"].(string)
	if task == "" || systemPrompt == "" {
		return errResult("task and system_prompt are required", ""), nil
	}

	contextExcerpt, _ := args["context_excerpt"].(string)
	if contextExcerpt != "" {
		task = strings.TrimSpace(task) + "\n\n" + contextExcerpt
	}

	var toolNames []string
	if raw, ok := args["tool_names"]; ok && raw != nil {
		switch v := raw.(type) {
		case []string:
			toolNames = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					toolNames = append(toolNames, s)
				}
			}
		}
	}

	subRegistry := t.buildSubRegistry(toolNames)

	config := core.AgentConfig{
		SystemPrompt: systemPrompt,
		Pipeline:     t.pipeline,
		Tools:        subRegistry,
		Provider:     t.provider,
		Orchestrator: nil,
	}
	subAgent := core.NewAgent(config)

	userMsg := types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: task}},
	}
	stream := subAgent.Prompt(ctx, userMsg)

	var mu sync.Mutex
	var lines []string
	go func() {
		for e := range stream.Events() {
			line := compactEvent(e)
			if line == "" {
				continue
			}
			mu.Lock()
			lines = append(lines, line)
			mu.Unlock()
		}
	}()

	messages, err := stream.Result()
	mu.Lock()
	thinking := strings.Join(lines, "\n")
	mu.Unlock()

	if err != nil {
		return errResult(fmt.Sprintf("sub-agent failed: %v", err), thinking), nil
	}

	lastText := types.LastAssistantText(messages)
	if lastText == "" {
		return errResult("sub-agent produced no assistant text", thinking), nil
	}

	out := map[string]interface{}{
		"response": lastText,
		"error":    "",
	}
	if thinking != "" {
		out["thinking"] = thinking
	}
	return out, nil
}

func (t *Tool) buildSubRegistry(toolNames []string) *tools.ToolRegistry {
	r := tools.NewToolRegistry()
	if t.pool == nil {
		return r
	}
	if len(toolNames) == 0 {
		for _, tool := range t.pool.All() {
			r.Register(tool)
		}
		return r
	}
	for _, name := range toolNames {
		if tool, ok := t.pool.Get(name); ok {
			r.Register(tool)
		}
	}
	return r
}

// errResult returns a map with error; when thinking is non-empty it is included so the master can reason and retry.
func errResult(msg, thinking string) map[string]interface{} {
	out := map[string]interface{}{
		"response": "",
		"error":    msg,
	}
	if thinking != "" {
		out["thinking"] = thinking
	}
	return out
}

// compactEvent formats a sub-agent stream event as a single line for thinking content.
func compactEvent(e core.AgentEvent) string {
	switch e.Type {
	case core.EventTurnStart:
		return "turn_start"
	case core.EventSteeringMode:
		if p, ok := e.Payload.(core.SteeringModePayload); ok {
			return p.Mode
		}
		return "steering_mode"
	case core.EventToolCall:
		if p, ok := e.Payload.(core.ToolCallPayload); ok {
			return "tool: " + p.ToolName
		}
		return "tool_call"
	case core.EventToolResult:
		if p, ok := e.Payload.(core.ToolResultPayload); ok {
			if p.Error != "" {
				return "result: " + p.ToolName + " (error)"
			}
			return "result: " + p.ToolName
		}
		return "tool_result"
	case core.EventThinking:
		if p, ok := e.Payload.(core.ThinkingPayload); ok {
			s := p.Text
			if len(s) > maxThinkingLen {
				s = s[:maxThinkingLen] + "..."
			}
			return "thinking: " + s
		}
		return "thinking"
	case core.EventTextDelta:
		return "output"
	case core.EventTurnEnd:
		return "turn_end"
	case core.EventPlanCreated:
		if p, ok := e.Payload.(core.PlanCreatedPayload); ok {
			return fmt.Sprintf("plan_created: %d step(s)", p.StepCount)
		}
		return "plan_created"
	case core.EventPlanStepStart:
		if p, ok := e.Payload.(core.PlanStepStartPayload); ok {
			return fmt.Sprintf("plan_step: %s", p.Tool)
		}
		return "plan_step_start"
	case core.EventPlanStepEnd:
		if p, ok := e.Payload.(core.PlanStepEndPayload); ok {
			if p.Error != "" {
				return fmt.Sprintf("plan_step_done: %s (error)", p.Tool)
			}
			return fmt.Sprintf("plan_step_done: %s", p.Tool)
		}
		return "plan_step_end"
	default:
		return ""
	}
}
