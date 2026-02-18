package main

import (
	"github.com/biome/agent-core/packages/agent/core"
)

// RunDelegation runs examples where the master agent can delegate to a sub-agent (persona + optional tools).
// The master uses the agentic orchestrator and has the delegate tool; these prompts encourage using it.
func RunDelegation(agent *core.Agent) {
	RunExample(agent, "Delegate to a math expert: ask a sub-agent with system prompt 'You are a math expert. Be concise.' to compute 15 * 3. Use tool_names ['calculator'].")
	RunExample(agent, "Use the delegate tool: task is 'What time is it?', system_prompt is 'You are a helpful time assistant.', and give the sub-agent the get_current_time tool.")
}
