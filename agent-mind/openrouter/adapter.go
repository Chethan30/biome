package openrouter

import (
	"context"
	"fmt"

	"github.com/biome/agent-mind/provider"
)

// Provider implements the provider.Provider interface for OpenRouter
type Provider struct {
	client *Client
	model  string
}

// NewProvider creates an OpenRouter provider
func NewProvider(apiKey, model string) *Provider {
	return &Provider{
		client: NewClient(apiKey),
		model:  model,
	}
}

// Stream implements provider.Provider
func (p *Provider) Stream(ctx context.Context, req provider.CompletionRequest) (<-chan provider.StreamEvent, error) {
	if p == nil || p.client == nil {
		// Create error channel
		errCh := make(chan provider.StreamEvent, 1)
		errCh <- provider.StreamEvent{
			Type:  provider.EventError,
			Error: fmt.Errorf("provider not initialized"),
		}
		close(errCh)
		return errCh, fmt.Errorf("provider not initialized")
	}
	return p.client.Stream(ctx, req, p.model)
}

// Complete implements provider.Provider
func (p *Provider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("provider not initialized")
	}
	return p.client.Complete(ctx, req, p.model)
}

// Name implements provider.Provider
func (p *Provider) Name() string {
	return "openrouter"
}

// Models implements provider.Provider
func (p *Provider) Models() []string {
	return []string{
		"openai/gpt-4o-mini",
		"openai/gpt-4o",
		"anthropic/claude-3.5-sonnet",
		"google/gemini-2.0-flash-exp:free",
		"meta-llama/llama-3.3-70b-instruct",
		"qwen/qwen-2.5-72b-instruct",
	}
}
