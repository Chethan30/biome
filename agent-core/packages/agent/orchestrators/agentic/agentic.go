package agentic

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/biome/agent-core/packages/agent/core"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-core/packages/stream"
)

func init() {
	core.SetDefaultOrchestrator(Default())
}

// Default returns the default agentic orchestrator (steering + tools + respond).
func Default() core.Orchestrator {
	return &AgenticOrchestrator{}
}

// AgenticOrchestrator implements the default turn loop: steering decision, optional tool execution, response, and follow-up.
type AgenticOrchestrator struct{}

// Run runs one agentic turn: steering, tools, respond, follow-up.
func (o *AgenticOrchestrator) Run(ctx context.Context, agent *core.Agent, userMessage types.UserMessage, eventStream *stream.EventStream[core.AgentEvent, []types.AgentMessage]) {
	_ = userMessage // already appended by Prompt() before Run()
	startTime := time.Now()
	state := agent.State()
	config := agent.Config()

	// Emit turn start
	eventStream.Push(core.AgentEvent{
		Type:    core.EventTurnStart,
		Payload: core.TurnStartPayload{Timestamp: time.Now().UnixMilli()},
	})

	firstTurn := true
	for {
		if !firstTurn {
			eventStream.Push(core.AgentEvent{
				Type:    core.EventTurnStart,
				Payload: core.TurnStartPayload{Timestamp: time.Now().UnixMilli()},
			})
		}

		if ctx.Err() != nil {
			eventStream.EndWithError(ctx.Err())
			return
		}

		decision, err := agent.SteeringDecision(ctx, !firstTurn)
		if err != nil {
			agent.SetError(fmt.Sprintf("%v", err))
			decision = core.SteeringDecision{
				Mode:     core.SteeringModeRespond,
				Response: fmt.Sprintf("I encountered an error: %v", err),
			}
		}

		eventStream.Push(core.AgentEvent{
			Type: core.EventSteeringMode,
			Payload: core.SteeringModePayload{
				Mode:      string(decision.Mode),
				QueueSize: len(decision.ToolCalls),
			},
		})

		var responseText string

		for decision.Mode == core.SteeringModeSteer {
			if ctx.Err() != nil {
				eventStream.EndWithError(ctx.Err())
				return
			}

			if decision.ThinkingText != "" {
				eventStream.Push(core.AgentEvent{
					Type:    core.EventThinking,
					Payload: core.ThinkingPayload{Text: decision.ThinkingText},
				})
			}

			assistantBlocks := make([]types.ContentBlock, 0, len(decision.ToolCalls))
			for _, tc := range decision.ToolCalls {
				assistantBlocks = append(assistantBlocks, types.ToolCallContent{
					ID:        tc.ToolCallId,
					Name:      tc.ToolName,
					Arguments: tc.Args,
				})
			}
			modelUsed := decision.Model
			if modelUsed == "" {
				modelUsed = "unknown"
			}
			providerNameForTurn := ""
			if config.Provider != nil {
				providerNameForTurn = config.Provider.Name()
			}
			assistantWithToolCalls := types.AssistantMessage{
				Content:    assistantBlocks,
				Provider:   providerNameForTurn,
				Model:      modelUsed,
				StopReason: types.StopReasonToolUse,
			}
			state.Messages = append(state.Messages, assistantWithToolCalls)

			calls := decision.ToolCalls
			for _, tc := range calls {
				state.PendingToolCalls[tc.ToolCallId] = true
			}

			// Run all tool calls in parallel; collect results in invocation order.
			results := make([]types.ToolResultMessage, len(calls))
			var wg sync.WaitGroup
			for i := range calls {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					results[i] = agent.ExecuteTool(ctx, calls[i])
				}(i)
			}
			wg.Wait()

			// Emit events and append results in order.
			for _, tc := range calls {
				eventStream.Push(core.AgentEvent{
					Type: core.EventToolCall,
					Payload: core.ToolCallPayload{
						ToolCallId: tc.ToolCallId,
						ToolName:   tc.ToolName,
						Args:       tc.Args,
					},
				})
			}
			for i, tc := range calls {
				res := results[i]
				eventStream.Push(core.AgentEvent{
					Type: core.EventToolResult,
					Payload: core.ToolResultPayload{
						ToolCallId: tc.ToolCallId,
						ToolName:   tc.ToolName,
						Result:     res.Details,
						Error:      core.ToolResultError(res),
					},
				})
				delete(state.PendingToolCalls, tc.ToolCallId)
				state.Messages = append(state.Messages, res)
			}

			if config.GetSteeringMessages != nil {
				if steering := config.GetSteeringMessages(); len(steering) > 0 {
					for _, m := range steering {
						state.Messages = append(state.Messages, m)
					}
				}
			}

			// Delegation summary and control message to keep the agent going.
			var summaryLines []string
			for _, tc := range calls {
				if tc.ToolName == "delegate" {
					summaryLines = append(summaryLines, fmt.Sprintf("Task delegated via tool_call_id %s.", tc.ToolCallId))
				}
			}
			controlText := "All requested tool calls have completed. What would you like to do next?"
			if len(summaryLines) > 0 {
				controlText = strings.Join(summaryLines, " ") + " " + controlText
			}
			state.Messages = append(state.Messages, types.ControlMessage{
				Content: []types.ContentBlock{types.TextContent{Text: controlText}},
			})

			decision, err = agent.SteeringDecision(ctx, true)
			if err != nil {
				agent.SetError(fmt.Sprintf("%v", err))
				responseText = fmt.Sprintf("I encountered an error: %v", err)
				break
			}

			eventStream.Push(core.AgentEvent{
				Type: core.EventSteeringMode,
				Payload: core.SteeringModePayload{
					Mode:      string(decision.Mode),
					QueueSize: len(decision.ToolCalls),
				},
			})

			if decision.Mode == core.SteeringModeRespond {
				responseText = decision.Response
				break
			}
		}

		if decision.Mode == core.SteeringModeRespond && responseText == "" {
			responseText = decision.Response
		}

		if responseText != "" {
			state.IsStreaming = true
			words := strings.Split(responseText, " ")
			for i, word := range words {
				eventStream.Push(core.AgentEvent{
					Type: core.EventTextDelta,
					Payload: core.TextDeltaPayload{
						Text:  word + " ",
						Index: i,
					},
				})
			}
			state.IsStreaming = false
			state.StreamMessage = nil
		}

		providerName := ""
		if config.Provider != nil {
			providerName = config.Provider.Name()
		}
		modelUsed := decision.Model
		if modelUsed == "" {
			modelUsed = "unknown"
		}
		assistantMessage := types.AssistantMessage{
			Content:    []types.ContentBlock{types.TextContent{Text: responseText}},
			Provider:   providerName,
			Model:      modelUsed,
			StopReason: types.StopReasonStop,
		}
		state.Messages = append(state.Messages, assistantMessage)

		duration := time.Since(startTime).Milliseconds()
		eventStream.Push(core.AgentEvent{
			Type: core.EventTurnEnd,
			Payload: core.TurnEndPayload{
				Message:  assistantMessage,
				Duration: duration,
			},
		})

		if config.GetFollowUpMessages == nil {
			break
		}
		followUp := config.GetFollowUpMessages()
		if len(followUp) == 0 {
			break
		}
		for _, m := range followUp {
			state.Messages = append(state.Messages, m)
		}
		firstTurn = false
	}

	eventStream.End(state.Messages)
}
