package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	examplestools "github.com/biome/agent-core/examples/tools"
	"github.com/biome/agent-core/packages/agent/core"
	_ "github.com/biome/agent-core/packages/agent/orchestrators/agentic"
	"github.com/biome/agent-core/packages/agent/orchestrators/planexecute"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/openrouter"
)

func main() {
	orchFlag := flag.String("orchestrator", "", "orchestrator: agentic or planexecute (overrides AGENT_ORCHESTRATOR env)")
	flag.Parse()

	fmt.Println("Agent Demo: Calculator & Get Time Tools")
	fmt.Println("========================================")
	fmt.Println("Use -orchestrator=planexecute or AGENT_ORCHESTRATOR=planexecute for plan-and-execute; otherwise agentic.")

	// llmModel := "anthropic/claude-3-haiku"
	// llmModel := "openai/gpt-5-nano"
	llmModel := "qwen/qwen3-235b-a22b-2507"

	// Setup provider
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("Set OPENROUTER_API_KEY to run with a real LLM")
		return
	}

	provider := openrouter.NewProvider(apiKey, llmModel)

	// Tools from config: AGENT_TOOLS_JSON (array of tool configs) or demo-local example tools
	registry := buildToolRegistry()

	// Orchestrator: -orchestrator flag or AGENT_ORCHESTRATOR env; otherwise default (agentic)
	orch := strings.ToLower(strings.TrimSpace(*orchFlag))
	if orch == "" {
		orch = strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_ORCHESTRATOR")))
	}
	config := core.AgentConfig{
		SystemPrompt: "You are a helpful assistant. Use the calculator for math and get_current_time for the current time.",
		Tools:        registry,
		Provider:     provider,
	}
	if orch == "planexecute" {
		config.Orchestrator = planexecute.Default()
		fmt.Println("Using plan-and-execute orchestrator")
	} else {
		fmt.Println("Using agentic orchestrator")
	}

	agent := core.NewAgent(config)

	if config.Orchestrator != nil {
		runPlanExecuteDemo(agent)
	} else {
		runAgenticDemo(agent)
	}
}

// runAgenticDemo runs examples suited to the agentic orchestrator (LLM decides tool-by-tool).
func runAgenticDemo(agent *core.Agent) {
	runExample(agent, "What is 15 * 3?")
	runExample(agent, "What time is it right now, and what is 10 + 5?")
}

// runPlanExecuteDemo runs examples suited to plan-and-execute (plan first, then execute steps, then synthesize).
func runPlanExecuteDemo(agent *core.Agent) {
	runExample(agent, "What is 15 * 3?")
	runExample(agent, "What time is it right now? Then calculate 10+5 and 20*2. Give me a short summary of the current time and both results.")
}

// ansi color codes for terminal (metadata events; final message stays default)
const (
	ansiCyan  = "\033[36m"
	ansiReset = "\033[0m"
)

// runExample sends a user message, prints tool calls/results and text deltas, then waits for the stream to finish.
// All streaming events except the final assistant text (EventTextDelta) are printed in cyan; the last message is default color.
func runExample(agent *core.Agent, userText string) {
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
			fmt.Printf("%s  [result: %v]\n%s", ansiCyan, p.Result, ansiReset)
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
