package types

// Interface
type ContentBlock interface {
	ContentType() string
}

// Types
type TextContent struct {
	Text string
}

type ImageContent struct {
	Data string
	MimeType string
}

type ThinkingContent struct {
	Thinking string
}

type ToolCallContent struct {
	ID string
	Name string
	Arguments interface{}
}

// Impls
func (tc TextContent) ContentType() string { return "text" }
func (ic ImageContent) ContentType() string { return "image" }
func (thc ThinkingContent) ContentType() string { return "thinking" }
func (tcc ToolCallContent) ContentType() string { return "toolCall" }



