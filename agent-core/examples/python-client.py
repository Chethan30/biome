#!/usr/bin/env python3
"""Simple Python client for agent-core HTTP API."""

import requests
import json

BASE_URL = "http://localhost:8080"

def main():
    print("Python Client: Calculator Example")
    print("=" * 40)

    # Check health
    try:
        r = requests.get(f"{BASE_URL}/health")
        print(f"Server: {r.json()['status']}")
    except Exception as e:
        print(f"Server not running: {e}")
        print("Start with: go run cmd/http-server/main.go")
        return

    # Single calculation
    print("\nUser: What is 15 * 3?")
    print("Agent: ", end="", flush=True)

    response = requests.post(
        f"{BASE_URL}/agent/prompt",
        json={"message": "What is 15 * 3?", "tools": ["calculator"], "stream": True},
        stream=True
    )

    for line in response.iter_lines():
        if line and line.startswith(b'data: '):
            try:
                event = json.loads(line[6:])
                if event.get('type') == 'text_delta':
                    print(event['payload'].get('Text', ''), end="", flush=True)
                elif event.get('type') == 'tool_call':
                    print(f"\n  [calling {event['payload'].get('ToolName', '')}]", flush=True)
                elif event.get('type') == 'tool_result':
                    print(f"  [result: {event['payload'].get('Result', '')}]", flush=True)
            except json.JSONDecodeError:
                pass

    print("\n\nDone!")

if __name__ == "__main__":
    main()
