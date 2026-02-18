package main

import (
	"encoding/json"
	"fmt"
	"os"

	examplestools "github.com/biome/agent-core/examples/tools"
	"github.com/biome/agent-core/packages/agent/tools"
	"github.com/biome/agent-core/packages/agent/tools/delegate"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-mind/provider"
)

// buildToolRegistry builds a registry from AGENT_TOOLS_JSON env, or uses example tools for demo.
// When using example tools, the delegate tool is registered so the master can create sub-agents.
// prov is required for the delegate tool when it runs sub-agents.
func buildToolRegistry(prov provider.Provider) *tools.ToolRegistry {
	if s := os.Getenv("AGENT_TOOLS_JSON"); s != "" {
		var configs []tools.ToolConfig
		if err := json.Unmarshal([]byte(s), &configs); err != nil {
			fmt.Printf("Warning: invalid AGENT_TOOLS_JSON: %v, using example tools\n", err)
			return exampleToolRegistry(prov)
		}
		r, err := tools.NewRegistryFromConfig(configs, nil)
		if err != nil {
			fmt.Printf("Warning: tool config error: %v, using example tools\n", err)
			return exampleToolRegistry(prov)
		}
		return registerDelegate(prov, r)
	}
	return exampleToolRegistry(prov)
}

// registerDelegate adds the delegate tool to the master registry. The pool passed to the
// delegate is the given registry (sub-agents get a subset of these tools, never the delegate itself).
func registerDelegate(prov provider.Provider, pool *tools.ToolRegistry) *tools.ToolRegistry {
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	delegateTool := delegate.New(prov, pipeline, pool)
	r := tools.NewToolRegistry()
	for _, t := range pool.All() {
		r.Register(t)
	}
	r.Register(delegateTool)
	return r
}

// exampleToolRegistry returns a registry with example tools and the delegate tool (demo-local tools + framework delegate).
func exampleToolRegistry(prov provider.Provider) *tools.ToolRegistry {
	pool := tools.NewToolRegistry()
	pool.Register(&examplestools.CalculatorTool{})
	pool.Register(&examplestools.GetTimeTool{})
	return registerDelegate(prov, pool)
}
