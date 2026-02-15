package types

// ToolLister provides read-only tool names for context snapshots.
// Implemented by e.g. *tools.ToolRegistry; avoids types depending on tools package.
type ToolLister interface {
	ListTools() []string
}

// AgentContext is an immutable snapshot of state for LLM calls and transforms.
type AgentContext struct {
	SystemPrompt string
	Messages     []AgentMessage // Copied in Clone(); caller must clone before mutating
	Tools        ToolLister     // Optional; reference only, not deep-copied
}

// Clone returns a deep copy of the context (messages copied; Tools reference shared).
func (c AgentContext) Clone() AgentContext {
	msgs := make([]AgentMessage, len(c.Messages))
	copy(msgs, c.Messages)
	return AgentContext{
		SystemPrompt: c.SystemPrompt,
		Messages:     msgs,
		Tools:        c.Tools, // registry is immutable for snapshot purposes
	}
}

// AgentState is the mutable runtime state of an agent.
type AgentState struct {
	// Configuration (set at init, unchanged by Reset)
	SystemPrompt string
	Tools        ToolLister

	// Conversation history
	Messages []AgentMessage

	// Runtime flags
	IsStreaming     bool
	StreamMessage   *AgentMessage   // Pointer to allow nil
	PendingToolCalls map[string]bool
	Error           *string
}

// ToContext returns an immutable snapshot for LLM/transform use. Call Clone() if the snapshot will be held across mutations.
func (s *AgentState) ToContext() AgentContext {
	return AgentContext{
		SystemPrompt: s.SystemPrompt,
		Messages:     s.Messages,
		Tools:        s.Tools,
	}
}