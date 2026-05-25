package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

// CalculatorTool 计算器工具 - 执行数学计算
type CalculatorTool struct {
	*BaseTool
}

// CalculatorArgs 计算器参数
type CalculatorArgs struct {
	Expression string `json:"expression"`
}

// NewCalculatorTool 创建计算器工具
func NewCalculatorTool() *CalculatorTool {
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "数学表达式，支持 +, -, *, /, ^, sqrt, sin, cos, tan 等",
			},
		},
		"required": []string{"expression"},
	}

	return &CalculatorTool{
		BaseTool: NewBaseTool("calculator", "执行数学计算", params),
	}
}

// Execute 执行计算
func (t *CalculatorTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var reqArgs CalculatorArgs
	if err := json.Unmarshal(args, &reqArgs); err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Invalid arguments: %v", err),
			IsError: true,
		}, nil
	}

	// 安全检查：防止注入
	if !isValidExpression(reqArgs.Expression) {
		return &ToolResult{
			Content: "Invalid expression: contains forbidden characters",
			IsError: true,
		}, nil
	}

	// 简单实现：实际生产环境应使用安全的表达式求值库
	result := simpleEvaluate(reqArgs.Expression)

	return &ToolResult{
		Content: fmt.Sprintf("Result: %v", result),
		Data: map[string]interface{}{
			"expression": reqArgs.Expression,
			"result":     result,
		},
	}, nil
}

// ValidatePermission 权限校验
func (t *CalculatorTool) ValidatePermission(userID string, args json.RawMessage) error {
	return nil
}

// isValidExpression 简单的表达式校验
func isValidExpression(expr string) bool {
	// 只允许数字、基本运算符和括号
	for _, c := range expr {
		if !((c >= '0' && c <= '9') || c == '+' || c == '-' || c == '*' || c == '/' || 
			c == '(' || c == ')' || c == '.' || c == ' ' || c == '^') {
			return false
		}
	}
	return true
}

// simpleEvaluate 简单表达式求值（仅支持基础运算）
func simpleEvaluate(expr string) float64 {
	// 这是一个简化实现，实际应使用成熟的表达式求值库
	// 例如：github.com/Knetic/govaluate
	return 0 // TODO: 实现完整的表达式求值
}
