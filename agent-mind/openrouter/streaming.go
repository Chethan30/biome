package openrouter

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/biome/agent-mind/provider"
)

// parseStreamingJson does best-effort parse of partial JSON (e.g. streaming tool arguments).
// Returns a non-nil map on success; returns empty map on parse error so callers can still use partial args.
func parseStreamingJson(s string) map[string]interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return map[string]interface{}{}
	}
	if out == nil {
		return map[string]interface{}{}
	}
	return out
}

// Stream creates a streaming request to OpenRouter
func (c *Client) Stream(ctx context.Context, req provider.CompletionRequest, model string) (<-chan provider.StreamEvent, error) {
	convertedTools := convertTools(req.Tools)
	fmt.Printf("[OPENROUTER] Stream tools: %d – ", len(convertedTools))
	if len(convertedTools) == 0 {
		fmt.Println("(none)")
	} else {
		for i, t := range convertedTools {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(t.Function.Name)
		}
		fmt.Println()
	}
	// Build OpenRouter request
	chatReq := chatRequest{
		Model:             model,
		Messages:          convertMessages(req.Messages),
		Temperature:       req.Temperature,
		MaxTokens:         req.MaxTokens,
		Stream:            true,
		Tools:             convertedTools,
		ParallelToolCalls: len(convertedTools) > 0,
	}

	// Add system prompt as first message if present
	if req.SystemPrompt != "" {
		chatReq.Messages = append([]chatMessage{
			{Role: "system", Content: req.SystemPrompt},
		}, chatReq.Messages...)
	}

	// Create HTTP request
	httpReq, err := c.createRequest(ctx, "POST", "/chat/completions", chatReq)
	if err != nil {
		return nil, err
	}

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Create event channel
	events := make(chan provider.StreamEvent, 10)

	// Start SSE parser goroutine
	go c.parseSSE(ctx, resp.Body, events)

	return events, nil
}

// streamToolCallState holds accumulated tool call for one index during streaming
type streamToolCallState struct {
	ID          string
	Name        string
	PartialArgs string
}

// parseSSE parses Server-Sent Events from response
func (c *Client) parseSSE(ctx context.Context, body io.ReadCloser, events chan<- provider.StreamEvent) {
	defer close(events)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	// Accumulate tool calls by index (OpenAI/OpenRouter stream tool_calls with index)
	toolCallByIndex := make(map[int]*streamToolCallState)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			events <- provider.StreamEvent{
				Type:  provider.EventError,
				Error: ctx.Err(),
			}
			return
		default:
		}

		line := scanner.Text()

		// SSE format: "data: {json}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// End of stream: emit completed tool calls then done
		if data == "[DONE]" {
			for idx, state := range toolCallByIndex {
				if state.ID == "" && state.Name == "" && state.PartialArgs == "" {
					continue
				}
				args := parseStreamingJson(state.PartialArgs)
				events <- provider.StreamEvent{
					Type: provider.EventToolCall,
					Content: &provider.ToolCallResponse{
						ID:        state.ID,
						Name:      state.Name,
						Arguments: args,
					},
				}
				_ = idx
			}
			events <- provider.StreamEvent{
				Type: provider.EventDone,
			}
			return
		}

		// Parse JSON chunk
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Extract delta
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta

			// Text delta (regular content)
			if delta.Content != "" {
				events <- provider.StreamEvent{
					Type:  provider.EventTextDelta,
					Delta: delta.Content,
				}
			}

			// Reasoning delta (for reasoning models like o1, DeepSeek-R1)
			if delta.Reasoning != "" {
				events <- provider.StreamEvent{
					Type:  provider.EventTextDelta,
					Delta: delta.Reasoning,
				}
			}

			// Tool call deltas: accumulate by index and emit incremental payloads (streaming tool-call parsing)
			if len(delta.ToolCalls) > 0 {
				fmt.Printf("[OPENROUTER] Tool call delta: %d chunk(s)\n", len(delta.ToolCalls))
			}
			for _, tc := range delta.ToolCalls {
				if tc.Index < 0 {
					continue
				}
				if toolCallByIndex[tc.Index] == nil {
					toolCallByIndex[tc.Index] = &streamToolCallState{}
				}
				st := toolCallByIndex[tc.Index]
				if tc.ID != "" {
					st.ID = tc.ID
				}
				if tc.Function != nil {
					if tc.Function.Name != "" {
						st.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						st.PartialArgs += tc.Function.Arguments
					}
				}
				args := parseStreamingJson(st.PartialArgs)
				events <- provider.StreamEvent{
					Type: provider.EventToolDelta,
					Content: &provider.ToolCallStreamPayload{
						Index:     tc.Index,
						ID:        st.ID,
						Name:      st.Name,
						Arguments: args,
					},
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		events <- provider.StreamEvent{
			Type:  provider.EventError,
			Error: fmt.Errorf("stream error: %w", err),
		}
	}
}

// Complete makes a non-streaming request
func (c *Client) Complete(ctx context.Context, req provider.CompletionRequest, model string) (*provider.CompletionResponse, error) {
	// Build request
	convertedTools := convertTools(req.Tools)
	chatReq := chatRequest{
		Model:             model,
		Messages:          convertMessages(req.Messages),
		Temperature:       req.Temperature,
		MaxTokens:         req.MaxTokens,
		Stream:            false,
		Tools:             convertedTools,
		ParallelToolCalls: len(convertedTools) > 0,
	}
	// Verify tools sent to OpenRouter
	fmt.Printf("[OPENROUTER] Tools in request: %d – ", len(convertedTools))
	if len(convertedTools) == 0 {
		fmt.Println("(none)")
	} else {
		names := make([]string, len(convertedTools))
		for i, t := range convertedTools {
			names[i] = t.Function.Name
		}
		fmt.Println(names)
	}

	// Add system prompt
	if req.SystemPrompt != "" {
		chatReq.Messages = append([]chatMessage{
			{Role: "system", Content: req.SystemPrompt},
		}, chatReq.Messages...)
	}

	// DEBUG: Print messages being sent
	// fmt.Println("\n[DEBUG] Messages being sent to API:")
	// for i, msg := range chatReq.Messages {
	// 	msgJSON, _ := json.MarshalIndent(msg, "  ", "  ")
	// 	fmt.Printf("  Message %d:\n%s\n", i, string(msgJSON))
	// }
	// fmt.Println()

	// Create HTTP request
	httpReq, err := c.createRequest(ctx, "POST", "/chat/completions", chatReq)
	if err != nil {
		return nil, err
	}

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp chatResponse
	bodyBytes, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract response
	text := ""
	var toolCalls []provider.ToolCallResponse

	if len(chatResp.Choices) > 0 {
		choice := chatResp.Choices[0]
		text = choice.Message.Content

		// Parse tool calls if present
		if len(choice.Message.ToolCalls) > 0 {
			toolCalls = make([]provider.ToolCallResponse, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				// Parse arguments JSON
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					// If parsing fails, skip this tool call
					continue
				}

				toolCalls = append(toolCalls, provider.ToolCallResponse{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: args,
				})
			}
		}
	}

	modelUsed := chatResp.Model
	if modelUsed == "" {
		modelUsed = model
	}
	return &provider.CompletionResponse{
		Text:      text,
		ToolCalls: toolCalls,
		Usage: provider.UsageInfo{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
		Model: modelUsed,
	}, nil
}
