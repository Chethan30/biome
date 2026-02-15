package provider

import (
	"context"
)

// Provider defines the interface for LLM providers
type Provider interface {
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, error)
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Name() string
	Models() []string
}

type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
}
