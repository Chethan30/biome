package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPTool implements Tool by calling an HTTP endpoint. Used when ToolConfig.Type == "http".
type HTTPTool struct {
	name         string
	description  string
	endpoint     string
	method       string
	parameters   map[string]interface{}
	auth         *HTTPAuthConfig
	httpClient   *http.Client
	responsePath string
	responseMap  map[string]string
}

// NewHTTPTool builds a Tool from an HTTP ToolConfig. If client is nil, http.DefaultClient is used.
func NewHTTPTool(cfg ToolConfig, client *http.Client) (Tool, error) {
	if cfg.Type != "http" {
		return nil, fmt.Errorf("ToolConfig type is %q, not http", cfg.Type)
	}
	if cfg.Name == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("http tool requires name and endpoint")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPTool{
		name:         cfg.Name,
		description:  cfg.Description,
		endpoint:     cfg.Endpoint,
		method:       cfg.Method,
		parameters:   cfg.Parameters,
		auth:         cfg.Auth,
		httpClient:   client,
		responsePath: cfg.ResponsePath,
		responseMap:  cfg.ResponseMap,
	}, nil
}

func (t *HTTPTool) Name() string        { return t.name }
func (t *HTTPTool) Description() string { return t.description }

func (t *HTTPTool) Parameters() ToolParameters {
	props := make(map[string]Property)
	required := []string{}
	if t.parameters != nil {
		if propsMap, ok := t.parameters["properties"].(map[string]interface{}); ok {
			for key, val := range propsMap {
				if propMap, ok := val.(map[string]interface{}); ok {
					prop := Property{}
					if typ, ok := propMap["type"].(string); ok {
						prop.Type = typ
					}
					if desc, ok := propMap["description"].(string); ok {
						prop.Description = desc
					}
					props[key] = prop
				}
			}
		}
		if reqList, ok := t.parameters["required"].([]interface{}); ok {
			for _, r := range reqList {
				if rStr, ok := r.(string); ok {
					required = append(required, rStr)
				}
			}
		}
	}
	return ToolParameters{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

func (t *HTTPTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	params := t.Parameters()
	for _, req := range params.Required {
		if _, ok := args[req]; !ok {
			return nil, fmt.Errorf("missing required parameter: %s", req)
		}
	}
	method := t.method
	if method == "" {
		method = "POST"
	}
	body, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, t.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.auth != nil {
		switch t.auth.Type {
		case "bearer", "apikey":
			h := "Authorization"
			if t.auth.Header != "" {
				h = t.auth.Header
			}
			req.Header.Set(h, "Bearer "+t.auth.APIKey)
		}
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bs, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(bs))
	}
	dec := json.NewDecoder(resp.Body)
	var out interface{}
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	if t.responsePath != "" {
		out = extractPath(out, t.responsePath)
	}
	if len(t.responseMap) > 0 && out != nil {
		out = applyResponseMap(out, t.responseMap)
	}
	return out, nil
}

func extractPath(v interface{}, path string) interface{} {
	// Simple path like "results[0]" or "data"
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '.' || r == '[' || r == ']' })
	for _, p := range parts {
		if p == "" {
			continue
		}
		switch m := v.(type) {
		case map[string]interface{}:
			v = m[p]
		case []interface{}:
			var i int
			if _, err := fmt.Sscanf(p, "%d", &i); err == nil && i >= 0 && i < len(m) {
				v = m[i]
			} else {
				return nil
			}
		default:
			return v
		}
	}
	return v
}

func applyResponseMap(v interface{}, m map[string]string) interface{} {
	vm, ok := v.(map[string]interface{})
	if !ok {
		return v
	}
	out := make(map[string]interface{})
	for outKey, inKey := range m {
		if val, ok := vm[inKey]; ok {
			out[outKey] = val
		}
	}
	return out
}
