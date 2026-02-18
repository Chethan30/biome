package main

import (
	"github.com/biome/agent-core/packages/agent/core"
)

// RunAgentic runs examples for the agentic orchestrator: the LLM decides per turn whether to respond or use tools (tool-by-tool).
func RunAgentic(agent *core.Agent) {
	RunExample(agent, "What is 15 * 3?")
	RunExample(agent, "What time is it right now, and what is 10 + 5?")
	RunExample(agent, "Calculate 20 - 7 and then tell me the current time.")
}
