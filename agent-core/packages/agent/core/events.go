package core

import "github.com/biome/agent-core/packages/agent/types"

const (
	EventTurnStart    = "turn_start"
	EventTextDelta    = "text_delta"
	EventThinking     = "thinking"
	EventSteeringMode = "steering_mode"
	EventToolCall     = "tool_call"
	EventToolResult   = "tool_result"
	EventTurnEnd      = "turn_end"
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
