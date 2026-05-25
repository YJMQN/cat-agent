package tool

import (
	"context"
	"encoding/json"
)

// ToolMeta 工具元数据
type ToolMeta struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
	Required    []string               `json:"required,omitempty"`
}

// Tool 工具接口 - 所有工具必须实现此接口
type Tool interface {
	// Meta 返回工具元数据
	Meta() *ToolMeta

	// Execute 执行工具
	Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)

	// ValidatePermission 权限校验
	ValidatePermission(userID string, args json.RawMessage) error
}

// ToolResult 工具执行结果
type ToolResult struct {
	Content string      `json:"content"`
	Data    interface{} `json:"data,omitempty"`
	IsError bool        `json:"is_error,omitempty"`
}

// BaseTool 基础工具结构，提供通用功能
type BaseTool struct {
	meta *ToolMeta
}

// NewBaseTool 创建基础工具
func NewBaseTool(name, description string, params map[string]interface{}) *BaseTool {
	return &BaseTool{
		meta: &ToolMeta{
			Name:        name,
			Description: description,
			Parameters:  params,
		},
	}
}

// Meta 返回工具元数据
func (b *BaseTool) Meta() *ToolMeta {
	return b.meta
}

// SetRequired 设置必需参数
func (b *BaseTool) SetRequired(required ...string) {
	b.meta.Required = required
}

// Registry 工具注册中心
type Registry struct {
	tools       map[string]Tool
	permissions map[string][]string // toolName -> allowed userIDs
}

// NewRegistry 创建工具注册中心
func NewRegistry() *Registry {
	return &Registry{
		tools:       make(map[string]Tool),
		permissions: make(map[string][]string),
	}
}

// Register 注册工具
func (r *Registry) Register(tool Tool) error {
	meta := tool.Meta()
	if _, exists := r.tools[meta.Name]; exists {
		return ErrToolAlreadyRegistered
	}
	r.tools[meta.Name] = tool
	return nil
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, ErrToolNotFound
	}
	return tool, nil
}

// List 列出所有可用工具
func (r *Registry) List() []*ToolMeta {
	metas := make([]*ToolMeta, 0, len(r.tools))
	for _, tool := range r.tools {
		metas = append(metas, tool.Meta())
	}
	return metas
}

// SetPermission 设置工具权限
func (r *Registry) SetPermission(toolName string, userIDs []string) {
	r.permissions[toolName] = userIDs
}

// CheckPermission 检查权限
func (r *Registry) CheckPermission(toolName, userID string) error {
	allowed, exists := r.permissions[toolName]
	if !exists {
		// 没有配置权限的工具默认允许所有用户
		return nil
	}

	for _, id := range allowed {
		if id == "*" || id == userID {
			return nil
		}
	}

	return ErrPermissionDenied
}

// Execute 执行工具
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage, userID string) (*ToolResult, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// 权限校验
	if err := r.CheckPermission(name, userID); err != nil {
		return &ToolResult{
			Content: "Permission denied",
			IsError: true,
		}, nil
	}

	// 工具自身的权限校验
	if err := tool.ValidatePermission(userID, args); err != nil {
		return &ToolResult{
			Content: err.Error(),
			IsError: true,
		}, nil
	}

	// 执行工具
	return tool.Execute(ctx, args)
}

// ToLLMTools 转换为 LLM 工具定义格式
func (r *Registry) ToLLMTools() []interface{} {
	tools := make([]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		meta := tool.Meta()
		tools = append(tools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        meta.Name,
				"description": meta.Description,
				"parameters":  meta.Parameters,
			},
		})
	}
	return tools
}

// 错误定义
var (
	ErrToolAlreadyRegistered = &ToolError{Code: "TOOL_ALREADY_REGISTERED", Message: "工具已注册"}
	ErrToolNotFound          = &ToolError{Code: "TOOL_NOT_FOUND", Message: "工具不存在"}
	ErrPermissionDenied      = &ToolError{Code: "PERMISSION_DENIED", Message: "权限不足"}
)

// ToolError 工具错误
type ToolError struct {
	Code    string
	Message string
	Err     error
}

func (e *ToolError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + " - " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *ToolError) Unwrap() error {
	return e.Err
}

// ParseArgs 解析工具参数辅助函数
func ParseArgs[T any](args json.RawMessage) (*T, error) {
	var result T
	if err := json.Unmarshal(args, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
