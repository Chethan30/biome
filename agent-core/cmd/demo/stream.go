package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/biome/agent-core/packages/agent/core"
	"github.com/biome/agent-core/packages/agent/types"
)

// ANSI color codes for terminal (metadata events; final message stays default).
const (
	ansiCyan  = "\033[36m"
	ansiReset = "\033[0m"
)

// RunExample sends a user message to the agent, prints tool calls/results and text deltas, then waits for the stream to finish.
// All streaming events except the final assistant text (EventTextDelta) are printed in cyan; the last message is default color.
func RunExample(agent *core.Agent, userText string) {
	fmt.Printf("\nUser: %s\n", userText)
	fmt.Println("Agent: ")

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: userText}},
	})

	for event := range stream.Events() {
		switch event.Type {
		case core.EventPlanCreated:
			p := event.Payload.(core.PlanCreatedPayload)
			fmt.Printf("%s  [plan: %d step(s)]\n", ansiCyan, p.StepCount)
			for i, s := range p.Steps {
				fmt.Printf("    %d. %s %v\n", i+1, s.Tool, s.Args)
			}
			fmt.Print(ansiReset)
		case core.EventPlanStepStart:
			p := event.Payload.(core.PlanStepStartPayload)
			fmt.Printf("%s  [step %d/%d: %s]\n%s", ansiCyan, p.Index+1, p.StepCount, p.Tool, ansiReset)
		case core.EventPlanStepEnd:
			p := event.Payload.(core.PlanStepEndPayload)
			if p.Error != "" {
				fmt.Printf("%s  [step %d/%d done: error %s]\n%s", ansiCyan, p.Index+1, p.StepCount, p.Error, ansiReset)
			} else {
				fmt.Printf("%s  [step %d/%d done: %v]\n%s", ansiCyan, p.Index+1, p.StepCount, p.Result, ansiReset)
			}
		case core.EventToolCall:
			p := event.Payload.(core.ToolCallPayload)
			fmt.Printf("%s  [tool: %s]\n%s", ansiCyan, p.ToolName, ansiReset)
		case core.EventToolResult:
			p := event.Payload.(core.ToolResultPayload)
			if m, ok := p.Result.(map[string]interface{}); ok {
				if thinking, _ := m["thinking"].(string); thinking != "" {
					const maxShow = 500
					show := thinking
					if len(show) > maxShow {
						show = show[:maxShow] + "..."
					}
					lines := strings.Split(show, "\n")
					fmt.Printf("%s  [thinking]\n%s", ansiCyan, ansiReset)
					for _, line := range lines {
						fmt.Printf("%s    %s\n%s", ansiCyan, line, ansiReset)
					}
				}
			}
			fmt.Printf("%s  [result: %v]\n%s", ansiCyan, p.Result, ansiReset)
		case core.EventTextDelta:
			p := event.Payload.(core.TextDeltaPayload)
			fmt.Print(p.Text)
		}
	}

	fmt.Println()
}
