package main

import (
	"github.com/biome/agent-core/packages/agent/core"
)

// RunPlanExecute runs examples for the plan-and-execute orchestrator: plan first (one LLM call), execute steps, then synthesize.
func RunPlanExecute(agent *core.Agent) {
	RunExample(agent, "What is 15 * 3?")
	RunExample(agent, "What time is it right now? Then calculate 10+5 and 20*2. Give me a short summary of the current time and both results.")
	RunExample(agent, "Get the current time, compute 100/4, and summarize the time and the quotient in one sentence.")
}
