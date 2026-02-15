package transform

import (
	"context"

	"github.com/biome/agent-core/packages/agent/types"
)

type Pipeline struct{
	transformContext 	TransformFunc
	convertToLLM 		ConvertFunc
}

// doubt: is this equivaletnt to a constructor?
// doubt: can I pass a chain traisnformer to this pipeline?
func NewPipeline(transform TransformFunc, convert ConvertFunc) *Pipeline {
	return &Pipeline{
		transformContext: transform,
		convertToLLM: convert,
	}
}

// Transform runs the pipeline on the given messages and returns LLM-ready messages.
func (p *Pipeline) Transform(ctx context.Context, messages []types.AgentMessage) ([]types.Message, error) {
	transformed := messages
	if p.transformContext != nil {
		var err error
		transformed, err = p.transformContext(ctx, messages)
		if err != nil {
			return nil, err
		}
	}
	llmMessages := p.convertToLLM(transformed)
	return llmMessages, nil
}

// TransformContext runs the pipeline on an AgentContext snapshot and returns LLM-ready messages.
// Use this before LLM calls so transforms operate on an immutable snapshot.
func (p *Pipeline) TransformContext(ctx context.Context, agentContext types.AgentContext) ([]types.Message, error) {
	return p.Transform(ctx, agentContext.Messages)
}