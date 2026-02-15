package core

import "github.com/biome/agent-core/packages/agent/types"

const (
	EventTurnStart     = "turn_start"
	EventTextDelta     = "text_delta"
	EventThinking      = "thinking"
	EventSteeringMode  = "steering_mode"
	EventToolCall      = "tool_call"
	EventToolResult    = "tool_result"
	EventTurnEnd       = "turn_end"
	EventPlanCreated   = "plan_created"
	EventPlanStepStart = "plan_step_start"
	EventPlanStepEnd   = "plan_step_end"
)

type AgentEvent struct {
	Type	string
	Payload	interface{}
}

type TurnStartPayload struct {
	Timestamp 	int64
}

type TextDeltaPayload struct {
	Text	string
	Index	int
}

type ToolCallPayload struct {
	ToolCallId	string
	ToolName 	string
	Args		map[string]interface{}
}

type ToolResultPayload struct {
	ToolCallId	string
	ToolName 	string
	Result		interface{}
	Error		string
}

type ThinkingPayload struct {
	Text string
}

type SteeringModePayload struct {
	Mode       string
	QueueSize  int
	NextAction string
}

type TurnEndPayload struct {
	Message		types.AssistantMessage
	Duration 	int64
}

// PlanCreatedPayload is emitted when the plan-and-execute orchestrator has produced a plan.
type PlanCreatedPayload struct {
	StepCount	int
	Steps		[]PlanStepInfo
}

// PlanStepInfo describes one step in a plan (for UI/observability).
type PlanStepInfo struct {
	Tool	string
	Args	map[string]interface{}
}

// PlanStepStartPayload is emitted when a plan step execution starts.
type PlanStepStartPayload struct {
	Index     int
	StepCount int
	Tool      string
	Args      map[string]interface{}
}

// PlanStepEndPayload is emitted when a plan step execution finishes.
type PlanStepEndPayload struct {
	Index     int
	StepCount int
	Tool      string
	Result    interface{}
	Error     string
}
