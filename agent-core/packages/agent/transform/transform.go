package transform

import (
	"context"

	"github.com/biome/agent-core/packages/agent/types"
)

type TransformFunc func (
	ctx context.Context,
	messages []types.AgentMessage,
) ([]types.AgentMessage, error)

type ConvertFunc func(
	messages []types.AgentMessage,
) []types.Message