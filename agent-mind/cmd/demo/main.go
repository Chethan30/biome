package main

import (
	"context"
	"fmt"
	"os"

	"github.com/biome/agent-core/packages/agent/types"
	"github.com/biome/agent-mind/openrouter"
	"github.com/biome/agent-mind/provider"
)

func main() {
	fmt.Println("🧠 Agent-Mind OpenRouter Demo")
	fmt.Println("================================")
	
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ OPENROUTER_API_KEY not set")
		fmt.Println("Get one at: https://openrouter.ai/keys")
		fmt.Println("\nUsage:")
		fmt.Println("  export OPENROUTER_API_KEY=\"your_key_here\"")
		fmt.Println("  go run cmd/demo/main.go")
		os.Exit(1)
	}
	
	// Create provider (free Nvidia reasoning model)
	llm := openrouter.NewProvider(apiKey, "nvidia/nemotron-3-nano-30b-a3b:free")
	
	fmt.Printf("Provider: %s\n", llm.Name())
	fmt.Printf("Model: nvidia/nemotron-3-nano-30b-a3b:free\n\n")
	
	// Create request
	req := provider.CompletionRequest{
		Messages: []types.Message{
			types.UserMessage{
				Content: []types.ContentBlock{
					types.TextContent{Text: "Write a haiku about Go programming."},
				},
			},
		},
		Temperature: 0.8,
		MaxTokens:   200,
	}
	
	fmt.Println("👤 User: Write a haiku about Go programming.")
	fmt.Print("🤖 Assistant: ")
	
	// Stream response
	stream, err := llm.Stream(context.Background(), req)
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		os.Exit(1)
	}
	
	fullText := ""
	for event := range stream {
		switch event.Type {
		case provider.EventTextDelta:
			fmt.Print(event.Delta)
			fullText += event.Delta
		case provider.EventError:
			fmt.Printf("\n❌ Error: %v\n", event.Error)
			os.Exit(1)
		case provider.EventDone:
			fmt.Println("\n\n✅ Complete!")
		}
	}
	
	fmt.Printf("\n📊 Total characters: %d\n", len(fullText))
}
