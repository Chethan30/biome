package types

// LastAssistantText returns the concatenated text from the last assistant message
// in the slice (by scanning from the end). Returns empty string if no assistant
// message is found or no text content blocks are present.
func LastAssistantText(messages []AgentMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if am, ok := msg.(AssistantMessage); ok {
			var text string
			for _, block := range am.Content {
				if tc, ok := block.(TextContent); ok {
					text += tc.Text
				}
			}
			return text
		}
	}
	return ""
}
