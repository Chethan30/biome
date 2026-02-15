package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	examplestools "github.com/biome/agent-core/examples/tools"
	"github.com/biome/agent-core/packages/agent/core"
	_ "github.com/biome/agent-core/packages/agent/orchestrators/agentic"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/openrouter"
)

func main() {
	fmt.Println("Agent Demo: Calculator & Get Time Tools")
	fmt.Println("========================================")

	// llmModel := "anthropic/claude-3-haiku"
	// llmModel := "openai/gpt-5-nano"
	llmModel := "qwen/qwen3-235b-a22b-2507"

	// Setup provider
	// apiKey := os.Getenv("OPENROUTER_API_KEY")
	apiKey := "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"
	if apiKey == "" {
		fmt.Println("Set OPENROUTER_API_KEY to run with a real LLM")
		return
	}

	provider := openrouter.NewProvider(apiKey, llmModel)

	// Tools from config: AGENT_TOOLS_JSON (array of tool configs) or demo-local example tools
	registry := buildToolRegistry()

	// Create agent
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are a helpful assistant. Use the calculator for math and get_current_time for the current time.",
		Tools:        registry,
		Provider:     provider,
	})

	// Example 1: single tool (calculator)
	runExample(agent, "What is 15 * 3?")

	// Example 2: multiple tools in one turn (time + calculator)
	runExample(agent, "What time is it right now, and what is 10 + 5?")
}

// runExample sends a user message, prints tool calls/results and text deltas, then waits for the stream to finish.
func runExample(agent *core.Agent, userText string) {
	fmt.Printf("\nUser: %s\n", userText)
	fmt.Println("Agent: ")

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: userText}},
	})

	for event := range stream.Events() {
		switch event.Type {
		case core.EventToolCall:
			p := event.Payload.(core.ToolCallPayload)
			fmt.Printf("  [tool: %s]\n", p.ToolName)
		case core.EventToolResult:
			p := event.Payload.(core.ToolResultPayload)
			fmt.Printf("  [result: %v]\n", p.Result)
		case core.EventTextDelta:
			p := event.Payload.(core.TextDeltaPayload)
			fmt.Print(p.Text)
		}
	}

	fmt.Println()
}

// buildToolRegistry builds a registry from AGENT_TOOLS_JSON env, or uses example tools for demo.
func buildToolRegistry() *tools.ToolRegistry {
	if s := os.Getenv("AGENT_TOOLS_JSON"); s != "" {
		var configs []tools.ToolConfig
		if err := json.Unmarshal([]byte(s), &configs); err != nil {
			fmt.Printf("Warning: invalid AGENT_TOOLS_JSON: %v, using example tools\n", err)
			return exampleToolRegistry()
		}
		r, err := tools.NewRegistryFromConfig(configs, nil)
		if err != nil {
			fmt.Printf("Warning: tool config error: %v, using example tools\n", err)
			return exampleToolRegistry()
		}
		return r
	}
	return exampleToolRegistry()
}

// exampleToolRegistry returns a registry with example tools (demo-local, not part of framework).
func exampleToolRegistry() *tools.ToolRegistry {
	r := tools.NewToolRegistry()
	r.Register(&examplestools.CalculatorTool{})
	r.Register(&examplestools.GetTimeTool{})
	return r
}
