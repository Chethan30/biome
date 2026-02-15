package core_test

import (
	"context"
	"testing"

	"github.com/biome/agent-core/packages/agent/core"
	_ "github.com/biome/agent-core/packages/agent/orchestrators/agentic"
	"github.com/biome/agent-core/packages/agent/transform"
	"github.com/biome/agent-core/packages/agent/types"
)

func TestNewAgent(t *testing.T) {
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful",
		Pipeline:     pipeline,
	})

	if agent == nil {
		t.Fatal("Expected agent to be created")
	}

	if len(agent.Messages()) != 0 {
		t.Errorf("Expected 0 initial messages, got %d", len(agent.Messages()))
	}
}

func TestAgentBasicPrompt(t *testing.T) {
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful",
		Pipeline:     pipeline,
	})

	userMsg := types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hello"}},
	}

	stream := agent.Prompt(context.Background(), userMsg)

	events := []core.AgentEvent{}
	for event := range stream.Events() {
		events = append(events, event)
	}

	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}

	if events[0].Type != core.EventTurnStart {
		t.Errorf("First event should be turn_start, got %s", events[0].Type)
	}

	lastEvent := events[len(events)-1]
	if lastEvent.Type != core.EventTurnEnd {
		t.Errorf("Last event should be turn_end, got %s", lastEvent.Type)
	}

	messages, err := stream.Result()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestAgentEventStreaming(t *testing.T) {
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful",
		Pipeline:     pipeline,
	})

	userMsg := types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hi"}},
	}

	stream := agent.Prompt(context.Background(), userMsg)

	turnStarts := 0
	turnEnds := 0

	for event := range stream.Events() {
		switch event.Type {
		case core.EventTurnStart:
			turnStarts++
		case core.EventTurnEnd:
			turnEnds++
		}
	}

	if turnStarts != 1 {
		t.Errorf("Expected 1 turn start, got %d", turnStarts)
	}
	if turnEnds != 1 {
		t.Errorf("Expected 1 turn end, got %d", turnEnds)
	}
}

func TestAgentMultipleTurns(t *testing.T) {
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful",
		Pipeline:     pipeline,
	})

	// Turn 1
	stream1 := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hello"}},
	})
	for range stream1.Events() {
	}
	messages1, _ := stream1.Result()

	if len(messages1) != 2 {
		t.Errorf("After turn 1: expected 2 messages, got %d", len(messages1))
	}

	// Turn 2
	stream2 := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "How are you?"}},
	})
	for range stream2.Events() {
	}
	messages2, _ := stream2.Result()

	if len(messages2) != 4 {
		t.Errorf("After turn 2: expected 4 messages, got %d", len(messages2))
	}
}

func TestAgentReset(t *testing.T) {
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful",
		Pipeline:     pipeline,
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Hello"}},
	})
	for range stream.Events() {
	}
	stream.Result()

	if len(agent.Messages()) == 0 {
		t.Error("Expected messages after prompt")
	}

	agent.Reset()

	if len(agent.Messages()) != 0 {
		t.Errorf("Expected 0 messages after reset, got %d", len(agent.Messages()))
	}
}

func TestAgentEventTiming(t *testing.T) {
	pipeline := transform.NewPipeline(nil, transform.DefaultConvertToLLM)
	agent := core.NewAgent(core.AgentConfig{
		SystemPrompt: "You are helpful",
		Pipeline:     pipeline,
	})

	stream := agent.Prompt(context.Background(), types.UserMessage{
		Content: []types.ContentBlock{types.TextContent{Text: "Test"}},
	})

	var gotTurnEnd bool
	for event := range stream.Events() {
		if event.Type == core.EventTurnEnd {
			gotTurnEnd = true
			payload := event.Payload.(core.TurnEndPayload)
			if payload.Duration < 0 {
				t.Errorf("Expected non-negative duration, got %dms", payload.Duration)
			}
		}
	}

	if !gotTurnEnd {
		t.Error("Expected turn_end event")
	}
}
