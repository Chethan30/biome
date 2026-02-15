package tools

import (
	"context"
	"fmt"
	"net/http"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() ToolParameters
	Execute(ctx context.Context, args map[string]interface{})	(interface{}, error)
}

type ToolParameters struct {
	Type string	`json:"type"`
	Properties map[string]Property	`json:"properties"`
	Required []string	`json:"required,omitempty"`
}

type Property struct {
	Type string	`json:"type"`
	Description	string	`json:"description"`
	Enum []string	`json:"enum,omitempty"`
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry{
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetTool returns a tool by name (for single return value)
func (r *ToolRegistry) GetTool(name string) Tool {
	return r.tools[name]
}

// ListTools returns list of registered tool names
func (r *ToolRegistry) ListTools() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

func (r *ToolRegistry) All() []Tool {
	toolList := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		toolList = append(toolList, tool)
	}
	return toolList
}

// NewRegistryFromConfig builds a ToolRegistry from tool configs. Only "http" type
// is implemented; "mcp" and "command" return an error for now. If client is nil,
// the default HTTP client is used for http tools.
func NewRegistryFromConfig(configs []ToolConfig, client *http.Client) (*ToolRegistry, error) {
	r := NewToolRegistry()
	for _, cfg := range configs {
		if cfg.Name == "" {
			continue
		}
		var tool Tool
		var err error
		switch cfg.Type {
		case "http":
			tool, err = NewHTTPTool(cfg, client)
		case "mcp", "command":
			err = fmt.Errorf("tool type %q not yet implemented", cfg.Type)
		default:
			err = fmt.Errorf("unknown tool type %q", cfg.Type)
		}
		if err != nil {
			return nil, fmt.Errorf("tool %q: %w", cfg.Name, err)
		}
		r.Register(tool)
	}
	return r, nil
}