// Package examplestools provides example Tool implementations for use with agent-core.
// These are not part of the framework; move or copy them into your app as needed.
package examplestools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/biome/agent-core/packages/agent/tools"
)

// CalculatorTool is an example tool that evaluates simple arithmetic expressions.
type CalculatorTool struct{}

func (c *CalculatorTool) Name() string {
	return "calculator"
}

func (c *CalculatorTool) Description() string {
	return "Performs basic arithmetic. Pass exactly one argument: 'expression' (required). Supports +, -, *, / (e.g. '2+2', '10*5'). Do not use 'input' or other parameter names."
}

func (c *CalculatorTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.Property{
			"expression": {
				Type:        "string",
				Description: "Required. The math expression to evaluate. Examples: '2+2', '15*3', '20-7', '10/2'.",
			},
		},
		Required: []string{"expression"},
	}
}

func (c *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	expr, ok := args["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("expression must be a string")
	}
	result, err := evaluate(expr)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"result": result}, nil
}

func evaluate(expr string) (float64, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	var left, right float64
	var op rune
	for i, r := range expr {
		if r == '+' || r == '-' || r == '*' || r == '/' {
			leftStr := expr[:i]
			rightStr := expr[i+1:]
			var err error
			left, err = strconv.ParseFloat(leftStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid left operand: %s", leftStr)
			}
			right, err = strconv.ParseFloat(rightStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid right operand: %s", rightStr)
			}
			op = r
			break
		}
	}
	switch op {
	case '+':
		return left + right, nil
	case '-':
		return left - right, nil
	case '*':
		return left * right, nil
	case '/':
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	default:
		return 0, fmt.Errorf("no operator found")
	}
}
