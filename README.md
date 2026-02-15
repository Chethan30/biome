# Biome

Biome is a Go monorepo for building LLM-driven agents with tool use, streaming events, and pluggable orchestration.

## Repositories

| Repository | Description |
|------------|-------------|
| **[agent-core](agent-core/README.md)** | Agent framework: state, turn loop, tools, event stream, and orchestrators. Defines the `Orchestrator` interface and hosts the default **agentic** loop plus **plan-and-execute**. Use agent-core to build an agent, register tools, and consume events. |
| **[agent-mind](agent-mind/README.md)** | LLM provider abstraction: OpenRouter (and future providers), streaming, tool-call handling. Use agent-mind to plug real models into agent-core via the `provider.Provider` interface. |

## How they fit together

- **agent-core** holds the agent: conversation state, tool registry, and the turn loop. It calls a **Provider** (from agent-mind) for completions and tool-call parsing. It does not implement HTTP or model APIs.
- **agent-mind** implements the Provider interface (e.g. OpenRouter). agent-core depends on agent-mind for the types and interface; agent-mind does not depend on agent-core.

## Quick links

- [agent-core README](agent-core/README.md) – architecture, state flow, extension points, quick start
- [agent-core packages/agent README](agent-core/packages/agent/README.md) – event flow, config, orchestrators
- [agent-mind README](agent-mind/README.md) – provider usage, OpenRouter, models

## Orchestrators

Orchestrators live under `agent-core/packages/agent/orchestrators/` and define how each turn runs (when to call the LLM, when to run tools, what events to emit).

- **[agentic](agent-core/packages/agent/orchestrators/agentic/README.md)** – Default. LLM decides per turn: respond (text) or steer (tool calls). Tools run one-by-one; steering and follow-up hooks available. Event pattern documented with diagrams.
- **[planexecute](agent-core/packages/agent/orchestrators/planexecute/README.md)** – Plan first (one LLM call → JSON plan), execute steps (run tools in order), then synthesize (one LLM call → final answer). Event pattern documented with diagrams.

## Building and running

From the repo root:

- Build all: `go build ./...` (in each module: `agent-core`, `agent-mind`)
- Demo (uses agent-core + agent-mind): `cd agent-core && go run ./cmd/demo`
- HTTP API: `cd agent-core && go run ./cmd/http-server`

See each repository’s README for details.
