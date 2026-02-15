package types

type AgentMessage interface {
    Role() string
    Timestamp() int64
}

// ConvertToLLM filters AgentMessages to only include standard LLM-compatible messages
func ConvertToLLM(messages []AgentMessage) []Message {
    llmMessages := make([]Message, 0, len(messages))

	for _, msg := range messages {
		switch m := msg.(type) {
		case UserMessage:
			llmMessages = append(llmMessages, m)
		case AssistantMessage:
			llmMessages = append(llmMessages, m)
		case ToolCallMessage:
			llmMessages = append(llmMessages, m)
		case ToolResultMessage:
			llmMessages = append(llmMessages, m)
		default:
			// skip
		}
	}

	return llmMessages
}