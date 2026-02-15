# Examples

Simple examples showing how to use agent-core.

## Go: Calculator Tool

```go
package main

import (
    "context"
    "fmt"

    "github.com/biome/agent-core/packages/agent/core"
    "github.com/biome/agent-core/packages/agent/tools"
    "github.com/biome/agent-core/packages/agent/types"
    "github.com/biome/agent-mind/openrouter"
)

func main() {
    // Setup
    provider := openrouter.NewProvider(apiKey, "openai/gpt-4o-mini")
    registry := tools.NewToolRegistry()
    registry.Register(&tools.CalculatorTool{})

    agent := core.NewAgent(core.AgentConfig{
        SystemPrompt: "You are helpful. Use calculator for math.",
        Tools:        registry,
        Provider:     provider,
    })

    // Prompt
    stream := agent.Prompt(context.Background(), types.UserMessage{
        Content: []types.ContentBlock{types.TextContent{Text: "What is 15 * 3?"}},
    })

    // Handle events
    for event := range stream.Events() {
        switch event.Type {
        case core.EventToolCall:
            fmt.Println("[calling calculator]")
        case core.EventToolResult:
            p := event.Payload.(core.ToolResultPayload)
            fmt.Printf("[result: %v]\n", p.Result)
        case core.EventTextDelta:
            p := event.Payload.(core.TextDeltaPayload)
            fmt.Print(p.Text)
        }
    }
}
```

## HTTP API

Start the server:

```bash
go run cmd/http-server/main.go
```

### Single calculation

```bash
curl -X POST http://localhost:8080/agent/prompt \
  -H 'Content-Type: application/json' \
  -d '{
    "message": "What is 15 * 3?",
    "tools": ["calculator"]
  }'
```

### Two calculations

```bash
curl -X POST http://localhost:8080/agent/prompt \
  -H 'Content-Type: application/json' \
  -d '{
    "message": "Calculate 10+5 and 20-7",
    "tools": ["calculator"]
  }'
```

## Python Client

```python
import requests
import json

response = requests.post(
    "http://localhost:8080/agent/prompt",
    json={
        "message": "What is 15 * 3?",
        "tools": ["calculator"],
        "stream": True
    },
    stream=True
)

for line in response.iter_lines():
    if line and line.startswith(b'data: '):
        event = json.loads(line[6:])
        if event.get('type') == 'text_delta':
            print(event['payload']['Text'], end='')
        elif event.get('type') == 'tool_result':
            print(f"[result: {event['payload']['Result']}]")
```

## JavaScript Client

```javascript
const response = await fetch('http://localhost:8080/agent/prompt', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        message: 'What is 15 * 3?',
        tools: ['calculator']
    })
});

const data = await response.json();
console.log(data.events);
```
