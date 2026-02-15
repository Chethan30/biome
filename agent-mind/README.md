# Agent-Mind 🧠

LLM provider abstraction for L3mma agent framework.

## Features

- ✅ Clean provider interface
- ✅ OpenRouter integration (200+ models)
- ✅ SSE streaming support
- ✅ OpenAI-compatible API
- ✅ Reasoning model support (o1, DeepSeek-R1, Nemotron, etc.)
- ✅ Easy to extend with new providers

## Quick Start

### 1. Get OpenRouter API Key

Sign up at [openrouter.ai](https://openrouter.ai/keys) and get a free API key.

### 2. Set Environment Variable

```bash
export OPENROUTER_API_KEY="your_key_here"
```

### 3. Run Demo

```bash
go run cmd/demo/main.go
```

## Usage

```go
import (
    "github.com/biome/agent-mind/openrouter"
    "github.com/biome/agent-mind/provider"
)

// Create provider
llm := openrouter.NewProvider(apiKey, "google/gemini-2.0-flash-exp:free")

// Stream response
stream, err := llm.Stream(ctx, provider.CompletionRequest{
    Messages: []types.Message{
        types.UserMessage{
            Content: []types.ContentBlock{
                types.TextContent{Text: "Hello!"},
            },
        },
    },
    Temperature: 0.7,
    MaxTokens: 200,
})

// Process events
for event := range stream {
    switch event.Type {
    case provider.EventTextDelta:
        fmt.Print(event.Delta)
    case provider.EventDone:
        fmt.Println("\nComplete!")
    }
}
```

## Available Models

OpenRouter provides access to:
- **OpenAI**: gpt-4o, gpt-4o-mini
- **Anthropic**: claude-3.5-sonnet
- **Google**: gemini-2.0-flash-exp (free!)
- **Meta**: llama-3.3-70b-instruct
- **Qwen**: qwen-2.5-72b-instruct
- And 200+ more!

## Project Structure

```
agent-mind/
├── provider/          - Provider interface & types
├── openrouter/        - OpenRouter implementation
└── cmd/demo/          - Demo application
```

## Next: Milestone 8

Integrate agent-mind into agent-core to bring your agent to life with real LLM intelligence!
