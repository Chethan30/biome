package core

import (
	"context"

	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-core/packages/stream"
)

// Orchestrator drives one conversation turn: it reads/writes agent state and pushes events.
// Other arrangements (ReAct, plan-execute, etc.) implement this interface.
type Orchestrator interface {
	Run(ctx context.Context, agent *Agent, userMessage types.UserMessage, eventStream *stream.EventStream[AgentEvent, []types.AgentMessage])
}

// defaultOrchestrator is used when AgentConfig.Orchestrator is nil.
// Set via SetDefaultOrchestrator (e.g. from orchestrators/agentic init).
var defaultOrchestrator Orchestrator

// SetDefaultOrchestrator sets the orchestrator used when AgentConfig.Orchestrator is nil.
// Call from an orchestrator package's init() to register as the default (e.g. agentic).
func SetDefaultOrchestrator(o Orchestrator) {
	defaultOrchestrator = o
}
