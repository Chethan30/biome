package core

type SteeringMode string

const (
	SteeringModeRespond SteeringMode = "respond"
	SteeringModeSteer   SteeringMode = "steer"
)

type SteeringDecision struct {
	Mode         SteeringMode
	ToolCalls    []ToolCallRequest
	ThinkingText string
	Response     string
	// Model is the model identifier used for this steering call (e.g. anthropic/claude-3-haiku). Empty if the provider did not report it.
	Model string
}

type ToolCallRequest struct {
	ToolCallId string
	ToolName   string
	Args       map[string]interface{}
}