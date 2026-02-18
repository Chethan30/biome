package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/biome/agent-core/packages/agent/core"
	_ "github.com/biome/agent-core/packages/agent/orchestrators/agentic"
	"github.com/biome/agent-core/packages/agent/orchestrators/planexecute"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-mind/openrouter"
)

const (
	demoAgentic     = "agentic"
	demoPlanExecute = "planexecute"
	demoDelegate    = "delegate"
	demoAll         = "all"
)

func main() {
	demoFlag := flag.String("demo", demoAll, "demo: agentic, planexecute, delegate, or all (runs all orchestration scenarios)")
	orchFlag := flag.String("orchestrator", "", "orchestrator: agentic or planexecute (overrides -demo when set, for backward compatibility)")
	flag.Parse()

	demo := strings.ToLower(strings.TrimSpace(*demoFlag))
	orch := strings.ToLower(strings.TrimSpace(*orchFlag))
	if orch != "" {
		if orch == demoPlanExecute {
			demo = demoPlanExecute
		} else {
			demo = demoAgentic
		}
		fmt.Printf("Using -orchestrator=%s (running %s scenario)\n", orch, demo)
	}

	fmt.Println("Agent Demo: Orchestration scenarios")
	fmt.Println("===================================")
	fmt.Println("Scenarios: agentic (tool-by-tool), planexecute (plan then execute), delegate (master/sub-agent).")
	fmt.Println("Use -demo=agentic|planexecute|delegate|all (default: all). Legacy: -orchestrator=agentic|planexecute.")

	llmModel := "qwen/qwen3-235b-a22b-2507"
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("Set OPENROUTER_API_KEY to run with a real LLM")
		return
	}

	prov := openrouter.NewProvider(apiKey, llmModel)
	registry := buildToolRegistry(prov)
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)

	systemPromptBase := "You are a helpful assistant. Use the calculator for math and get_current_time for the current time."
	systemPromptWithDelegate := systemPromptBase + " You may delegate specialized tasks to a sub-agent via the delegate tool (give it a task, a system_prompt persona, and optional tool_names). When a delegate tool result includes a sub-agent trace, use it to explain failures to the user and to decide whether to retry (e.g. with different task or tools)."

	runAll := demo == demoAll

	if runAll || demo == demoAgentic {
		fmt.Printf("\n--- %s scenario ---\n", demoAgentic)
		config := core.AgentConfig{
			SystemPrompt: systemPromptBase,
			Pipeline:     pipeline,
			Tools:        registry,
			Provider:     prov,
		}
		agent := core.NewAgent(config)
		RunAgentic(agent)
	}

	if runAll || demo == demoPlanExecute {
		fmt.Printf("\n--- %s scenario ---\n", demoPlanExecute)
		config := core.AgentConfig{
			SystemPrompt: systemPromptBase,
			Pipeline:     pipeline,
			Tools:        registry,
			Provider:     prov,
			Orchestrator: planexecute.Default(),
		}
		agent := core.NewAgent(config)
		RunPlanExecute(agent)
	}

	if runAll || demo == demoDelegate {
		fmt.Printf("\n--- %s scenario (master/sub-agent) ---\n", demoDelegate)
		config := core.AgentConfig{
			SystemPrompt: systemPromptWithDelegate,
			Pipeline:     pipeline,
			Tools:        registry,
			Provider:     prov,
		}
		agent := core.NewAgent(config)
		RunDelegation(agent)
	}
}
