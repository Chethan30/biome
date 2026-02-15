package examplestools

import (
	"context"
	"time"

	"github.com/biome/agent-core/packages/agent/tools"
)

// GetTimeTool is an example tool that returns the current time.
type GetTimeTool struct{}

func (g *GetTimeTool) Name() string {
	return "get_current_time"
}

func (g *GetTimeTool) Description() string {
	return "Returns the current date and time"
}

func (g *GetTimeTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type:       "object",
		Properties: map[string]tools.Property{},
		Required:   []string{},
	}
}

func (g *GetTimeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	now := time.Now()
	return map[string]interface{}{
		"timestamp": now.Unix(),
		"datetime":  now.Format(time.RFC3339),
		"timezone":  now.Location().String(),
	}, nil
}
