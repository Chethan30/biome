package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/provider"
)

// makeSteeringDecision uses the LLM to decide whether to respond or use tools.
// It takes an AgentContext snapshot so transforms and LLM see immutable state.
// Tools are always included when toolRegistry is non-nil.
func makeSteeringDecision(
	ctx context.Context,
	llm provider.Provider,
	agentContext types.AgentContext,
	pipeline *transform.Pipeline,
	toolRegistry *tools.ToolRegistry,
	isFollowUp bool,
	initialSteeringInstruction string,
) (SteeringDecision, error) {
	steeringPrompt := buildSteeringPrompt(agentContext.SystemPrompt, isFollowUp, initialSteeringInstruction)

	var providerMessages []types.Message
	if pipeline != nil {
		var err error
		providerMessages, err = pipeline.TransformContext(ctx, agentContext)
		if err != nil {
			return SteeringDecision{}, fmt.Errorf("pipeline transform: %w", err)
		}
	} else {
		providerMessages = make([]types.Message, len(agentContext.Messages))
		for i, msg := range agentContext.Messages {
			providerMessages[i] = msg
		}
	}

	var providerTools []provider.Tool
	if toolRegistry != nil {
		providerTools = convertToolsToProvider(toolRegistry)
		fmt.Printf("[STEERING] Sending %d tool(s) to provider: ", len(providerTools))
		if len(providerTools) == 0 {
			fmt.Println("(none)")
		} else {
			for i, t := range providerTools {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(t.Name)
			}
			fmt.Println()
		}
	}

	// Flow: system prompt first, then messages history; last message(s) are the current query (user message or tool results).
	req := provider.CompletionRequest{
		SystemPrompt: steeringPrompt,
		Messages:     providerMessages,
		Temperature:  0.7,
		MaxTokens:    1000,
		Tools:        providerTools,
	}

	fmt.Printf("[DEBUG] Req for steering mode to provider: \n SystemPrompt(len=%d), \n Messages(%d), \n Temperature=%.2f, \n MaxTokens=%d, \n Tools=%d\n",
		len(steeringPrompt), len(providerMessages), req.Temperature, req.MaxTokens, len(providerTools))

	// Get response from LLM
	resp, err := llm.Complete(ctx, req)
	if err != nil {
		return SteeringDecision{}, fmt.Errorf("steering decision failed: %w", err)
	}

	fmt.Printf("LLM Response for Steering Decision: %+v\n", resp)

	// Check if LLM wants to use tools (structured)
	if len(resp.ToolCalls) > 0 {
		// Convert to agent ToolCallRequests
		toolCalls := make([]ToolCallRequest, 0, len(resp.ToolCalls))
		toolNames := make([]string, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			toolCalls = append(toolCalls, ToolCallRequest{
				ToolCallId: tc.ID,
				ToolName:   tc.Name,
				Args:       tc.Arguments,
			})
			toolNames = append(toolNames, tc.Name)
		}

		// Create descriptive thinking text
		thinkingText := fmt.Sprintf("I'll use %s to help answer this.", strings.Join(toolNames, ", "))

		return SteeringDecision{
			Mode:         SteeringModeSteer,
			ThinkingText: thinkingText,
			ToolCalls:    toolCalls,
			Model:        resp.Model,
		}, nil
	}

	// Default to direct response
	return SteeringDecision{
		Mode:     SteeringModeRespond,
		Response: resp.Text,
		Model:    resp.Model,
	}, nil
}

// convertToolsToProvider converts tool registry to provider tools
func convertToolsToProvider(registry *tools.ToolRegistry) []provider.Tool {
	if registry == nil {
		return nil
	}

	providerTools := []provider.Tool{}

	// Get all registered tools
	for _, tool := range registry.All() {
		params := tool.Parameters()

		// Convert tool parameters to provider format
		properties := make(map[string]interface{})
		for propName, prop := range params.Properties {
			properties[propName] = map[string]interface{}{
				"type":        prop.Type,
				"description": prop.Description,
			}
		}

		providerTools = append(providerTools, provider.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters: map[string]interface{}{
				"type":       params.Type,
				"properties": properties,
				"required":   params.Required,
			},
		})
	}

	return providerTools
}

// buildSteeringPrompt creates system prompt for intelligent steering.
// initialSteeringInstruction is optional (from AgentConfig); when set, used for the initial (non-followUp) case instead of the default generic instruction.
func buildSteeringPrompt(basePrompt string, isFollowUp bool, initialSteeringInstruction string) string {
	if basePrompt == "" {
		basePrompt = "You are a helpful AI assistant."
	}

	if isFollowUp {
		// Follow-up prompt after tool execution: encourage completing tool intent before final answer
		return basePrompt +
			`You have just received tool results. Based on these results, you must decide:
		1. If the task requires MORE tool calls (e.g. user asked for multiple items and you have results for only some), call tools for the remaining items. 
		Do NOT reply with a final answer until you have tool results for all requested items.
		2. If you have ENOUGH information from tools, provide a final natural language response to the user.
		3. If you cannot complete the task with the tools or results you have, respond clearly that you don't have enough information—do not invent or guess data.
		CRITICAL: Do NOT repeat the same tool call if you already have its results. 
		Do NOT hallucinate: only use real tool results or say you couldn't complete the request.`
	}

	// Initial steering prompt: generic default or user-provided instruction from config
	if initialSteeringInstruction != "" {
		return basePrompt + "\n\n" + initialSteeringInstruction
	}
	return basePrompt + "\n\nWhen the user's query can be answered using the available tools, you MUST call the appropriate tool(s) first—do not answer with text alone. After you have tool results, provide a concise final response. For other queries, provide helpful, concise responses."
}
