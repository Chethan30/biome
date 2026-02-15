package tools

// ToolConfig describes one tool in a backend-agnostic way. The agent has access
// only to tools built from config (no prebuilt tools in the framework).
// Type determines which backend-specific fields are used.
type ToolConfig struct {
	Type        string                 `json:"type"`                  // "http", "mcp", "command"
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // JSON Schema for LLM

	// HTTP backend
	Endpoint     string            `json:"endpoint,omitempty"`
	Method       string            `json:"method,omitempty"`
	Auth         *HTTPAuthConfig   `json:"auth,omitempty"`
	ResponsePath string            `json:"response_path,omitempty"`
	ResponseMap  map[string]string `json:"response_map,omitempty"`

	// MCP backend: server to connect to (e.g. stdio command or SSE URL)
	MCPCommand string   `json:"mcp_command,omitempty"` // e.g. "npx -y @modelcontextprotocol/server-xyz"
	MCPArgs    []string `json:"mcp_args,omitempty"`
	MCPURL     string   `json:"mcp_url,omitempty"` // SSE URL if not stdio

	// Command backend: run a subprocess (any language)
	Command string   `json:"command,omitempty"` // e.g. "node", "python"
	Args    []string `json:"args,omitempty"`    // e.g. ["script.js"]
	Env     []string `json:"env,omitempty"`
}

// HTTPAuthConfig is authentication for HTTP tools.
type HTTPAuthConfig struct {
	Type   string `json:"type"`   // "apikey", "bearer", "basic"
	APIKey string `json:"apikey,omitempty"`
	Header string `json:"header,omitempty"`
}
