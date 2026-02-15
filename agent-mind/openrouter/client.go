package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/provider"
)

const (
	DefaultBaseURL = "https://openrouter.ai/api/v1"
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{},
	}
}

// (OpenAI-compatible)
type chatRequest struct {
	Model             string        `json:"model"`
	Messages          []chatMessage `json:"messages"`
	Temperature       float64       `json:"temperature,omitempty"`
	MaxTokens         int           `json:"max_tokens,omitempty"`
	Stream            bool          `json:"stream"`
	Tools             []toolDef     `json:"tools,omitempty"`
	ParallelToolCalls bool          `json:"parallel_tool_calls,omitempty"`
}

// chatMessage is OpenAI-compatible: assistant may have tool_calls; tool result has role "tool" and tool_call_id.
type chatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string, or omitempty when tool_calls present
	ToolCalls  []toolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// contentBlock represents a content block in a message
type contentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`        // For text blocks
	ToolUseID string `json:"tool_use_id,omitempty"` // For tool_result blocks
	Content   string `json:"content,omitempty"`     // For tool_result content
}

type toolDef struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type chatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// streamChunk represents SSE chunk from OpenRouter
type streamChunk struct {
	ID      string        `json:"id"`
	Model   string        `json:"model"`
	Choices []deltaChoice `json:"choices"`
}

type deltaChoice struct {
	Index int   `json:"index"`
	Delta delta `json:"delta"`
}

type delta struct {
	Role      string                `json:"role,omitempty"`
	Content   string                `json:"content,omitempty"`
	Reasoning string                `json:"reasoning,omitempty"` // For reasoning models (o1, DeepSeek-R1, etc.)
	ToolCalls []streamToolCallDelta `json:"tool_calls,omitempty"`
}

// streamToolCallDelta is the OpenAI/OpenRouter streaming delta for one tool call
type streamToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Function *struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

// createRequest builds HTTP request with auth
func (c *Client) createRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/biome/agent-mind")

	return req, nil
}

// convertMessages converts agent-core messages to OpenRouter format
func convertMessages(messages []types.Message) []chatMessage {
	result := make([]chatMessage, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role() {
		case "user":
			if userMsg, ok := msg.(types.UserMessage); ok {
				text := ""
				for _, block := range userMsg.Content {
					if textBlock, ok := block.(types.TextContent); ok {
						text += textBlock.Text
					}
				}
				result = append(result, chatMessage{
					Role:    "user",
					Content: text,
				})
			}
		case "assistant":
			if assistantMsg, ok := msg.(types.AssistantMessage); ok {
				text := ""
				var toolCalls []toolCall
				for _, block := range assistantMsg.Content {
					switch b := block.(type) {
					case types.TextContent:
						text += b.Text
					case types.ToolCallContent:
						argsStr := ""
						if b.Arguments != nil {
							if j, err := json.Marshal(b.Arguments); err == nil {
								argsStr = string(j)
							}
						}
						toolCalls = append(toolCalls, toolCall{
							ID:   b.ID,
							Type: "function",
							Function: toolCallFunction{
								Name:      b.Name,
								Arguments: argsStr,
							},
						})
					}
				}
				m := chatMessage{Role: "assistant", Content: text}
				if len(toolCalls) > 0 {
					m.ToolCalls = toolCalls
				}
				result = append(result, m)
			}
		case "toolResult":
			// OpenAI/OpenRouter: role "tool", content = result string, tool_call_id links to assistant tool_calls
			if toolResultMsg, ok := msg.(types.ToolResultMessage); ok {
				text := ""
				for _, block := range toolResultMsg.Content {
					if textBlock, ok := block.(types.TextContent); ok {
						text += textBlock.Text
					}
				}
				if text == "" && toolResultMsg.Details != nil {
					if j, err := json.Marshal(toolResultMsg.Details); err == nil {
						text = string(j)
					}
				}
				result = append(result, chatMessage{
					Role:       "tool",
					Content:    text,
					ToolCallID: toolResultMsg.ToolCallID,
				})
			}
		}
	}

	return result
}

// convertTools converts provider tools to OpenRouter format
func convertTools(tools []provider.Tool) []toolDef {
	result := make([]toolDef, 0, len(tools))

	for _, t := range tools {
		result = append(result, toolDef{
			Type: "function",
			Function: toolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	return result
}
