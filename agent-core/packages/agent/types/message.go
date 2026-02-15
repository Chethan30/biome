package types

// Util types and constants
type StopReason string

const (
	StopReasonStop StopReason = "stop"
	StopReasonLength StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonAborted StopReason = "aborted"
	StopReasonError StopReason = "error"
)

type Cost struct {
	Input      float64
	Output     float64
	CacheRead  float64
	CacheWrite float64
	Total      float64
}

type UsageMetrics struct {
	Input       int
    Output      int
    CacheRead   int
    CacheWrite  int
    TotalTokens int
    Cost       Cost
}

// Interface
type Message interface {
	Role() string
	Timestamp() int64
}

// Messages Types
type UserMessage struct {
	Content []ContentBlock
	timestamp int64
}

type AssistantMessage struct {
	Content []ContentBlock
	API string
	Provider string
	Model string
	Usage UsageMetrics
	StopReason StopReason
	ErrorMessage *string
	timestamp int64
}

type ToolCallMessage struct {
	Content []ContentBlock
	ToolCallID string
	ToolName string
	Arguments interface{}
	timestamp int64
}

type ToolResultMessage struct {
	Content []ContentBlock
	ToolCallID string
	ToolName string
	Details interface{}
	IsError bool
	timestamp int64
}

// Impls
func (u UserMessage) Role() string { return "user" }
func (u UserMessage) Timestamp() int64 { return u.timestamp }

func (a AssistantMessage) Role() string { return "assistant" }
func (a AssistantMessage) Timestamp() int64 { return a.timestamp }

func (tc ToolCallMessage) Role() string { return "toolCall" }
func (tc ToolCallMessage) Timestamp() int64 { return tc.timestamp }

func (tr ToolResultMessage) Role() string { return "toolResult" }
func (tr ToolResultMessage) Timestamp() int64 { return tr.timestamp }
