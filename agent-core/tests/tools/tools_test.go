package tools_test

import (
	"context"
	"testing"

	examplestools "github.com/biome/agent-core/examples/tools"
	"github.com/biome/agent-core/packages/agent/tools"
)

func TestNewToolRegistry(t *testing.T) {
	registry := tools.NewToolRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	}

	if len(registry.All()) != 0 {
		t.Errorf("Expected empty registry, got %d tools", len(registry.All()))
	}
}

func TestToolRegistryRegister(t *testing.T) {
	registry := tools.NewToolRegistry()

	calc := &examplestools.CalculatorTool{}
	registry.Register(calc)

	tool, ok := registry.Get("calculator")
	if !ok {
		t.Error("Expected to find calculator tool")
	}

	if tool.Name() != "calculator" {
		t.Errorf("Expected name 'calculator', got %s", tool.Name())
	}
}

func TestToolRegistryGetNonExistent(t *testing.T) {
	registry := tools.NewToolRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected false for non-existent tool")
	}
}

func TestToolRegistryMultiple(t *testing.T) {
	registry := tools.NewToolRegistry()

	registry.Register(&examplestools.CalculatorTool{})
	registry.Register(&examplestools.GetTimeTool{})

	all := registry.All()
	if len(all) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(all))
	}

	_, ok1 := registry.Get("calculator")
	_, ok2 := registry.Get("get_current_time")

	if !ok1 || !ok2 {
		t.Error("Expected to find both tools")
	}
}

func TestCalculatorAdd(t *testing.T) {
	calc := &examplestools.CalculatorTool{}

	result, err := calc.Execute(
		context.Background(),
		map[string]interface{}{"expression": "2+2"},
	)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["result"] != 4.0 {
		t.Errorf("Expected 4, got %v", resultMap["result"])
	}
}

func TestCalculatorOperations(t *testing.T) {
	calc := &examplestools.CalculatorTool{}

	tests := []struct {
		expr     string
		expected float64
	}{
		{"2+2", 4},
		{"10-3", 7},
		{"5*6", 30},
		{"100/4", 25},
	}

	for _, tt := range tests {
		result, err := calc.Execute(
			context.Background(),
			map[string]interface{}{"expression": tt.expr},
		)

		if err != nil {
			t.Errorf("Unexpected error for %s: %v", tt.expr, err)
			continue
		}

		resultMap := result.(map[string]interface{})
		if resultMap["result"] != tt.expected {
			t.Errorf("For %s: expected %f, got %v", tt.expr, tt.expected, resultMap["result"])
		}
	}
}

func TestCalculatorDivisionByZero(t *testing.T) {
	calc := &examplestools.CalculatorTool{}

	_, err := calc.Execute(
		context.Background(),
		map[string]interface{}{"expression": "10/0"},
	)

	if err == nil {
		t.Error("Expected error for division by zero")
	}
}

func TestCalculatorInvalidExpression(t *testing.T) {
	calc := &examplestools.CalculatorTool{}

	_, err := calc.Execute(
		context.Background(),
		map[string]interface{}{"expression": "invalid"},
	)

	if err == nil {
		t.Error("Expected error for invalid expression")
	}
}

func TestCalculatorMissingArg(t *testing.T) {
	calc := &examplestools.CalculatorTool{}

	_, err := calc.Execute(
		context.Background(),
		map[string]interface{}{},
	)

	if err == nil {
		t.Error("Expected error for missing expression argument")
	}
}

func TestGetTimeTool(t *testing.T) {
	timeTool := &examplestools.GetTimeTool{}

	result, err := timeTool.Execute(
		context.Background(),
		map[string]interface{}{},
	)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	resultMap := result.(map[string]interface{})

	if _, ok := resultMap["timestamp"]; !ok {
		t.Error("Expected timestamp in result")
	}

	if _, ok := resultMap["datetime"]; !ok {
		t.Error("Expected datetime in result")
	}

	if _, ok := resultMap["timezone"]; !ok {
		t.Error("Expected timezone in result")
	}
}

func TestToolMetadata(t *testing.T) {
	calc := &examplestools.CalculatorTool{}

	if calc.Name() == "" {
		t.Error("Tool name should not be empty")
	}

	if calc.Description() == "" {
		t.Error("Tool description should not be empty")
	}

	params := calc.Parameters()
	if params.Type != "object" {
		t.Error("Parameters type should be 'object'")
	}

	if len(params.Properties) == 0 {
		t.Error("Expected at least one property")
	}

	if len(params.Required) == 0 {
		t.Error("Expected at least one required field")
	}
}

func TestGetTimeMetadata(t *testing.T) {
	timeTool := &examplestools.GetTimeTool{}

	if timeTool.Name() != "get_current_time" {
		t.Errorf("Expected name 'get_current_time', got %s", timeTool.Name())
	}

	if timeTool.Description() == "" {
		t.Error("Description should not be empty")
	}

	params := timeTool.Parameters()
	if params.Type != "object" {
		t.Error("Parameters type should be 'object'")
	}
}
