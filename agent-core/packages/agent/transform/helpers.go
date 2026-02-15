package transform

import (
	"context"

	"github.com/biome/agent-core/packages/agent/types"
)

// keeps only last N most recent messages
func KeepRecentMessages(n int) TransformFunc { // doubt: why are we writing a function that returns another function? what is the end output? how can this be used?
	return func(ctx context.Context, messages []types.AgentMessage) ([]types.AgentMessage, error) {
		if len(messages) <= n {
			return messages, nil
		}

		return messages[len(messages)-n:], nil
	}
}

// keep first and last N most recent messages
func KeepFirstandRecentMessages(n int) TransformFunc {
	return func(ctx context.Context, messages []types.AgentMessage) ([]types.AgentMessage, error) {
		if len(messages) <= n {
			return messages, nil
		}

		first := messages[0]
		recent := messages[len(messages)-n:] // doubt: difference between := and =

		result := make([]types.AgentMessage, 0, n+1)
		result = append(result, first)
		result = append(result, recent...) // doubt: recent... is spread?

		return result, nil
	}
}

// combines multiple transforms
func ChainTrainsform(transforms ...TransformFunc) TransformFunc { // doubt: what is ...TransformFunc?
	return func(ctx context.Context, messages []types.AgentMessage) ([]types.AgentMessage, error) {
		result := messages
		var err error

		for _, transform := range transforms {
			result, err = transform(ctx, result)
			if err != nil {
				return nil, err
			}
		}

		return result, nil
	}
}