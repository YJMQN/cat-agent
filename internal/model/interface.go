package model

import (
	"context"
)

// Message 模型消息格式
type Message struct {
	Role       string     `json:"role"` // system, user, assistant, tool
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDef 工具定义 (传给模型)
type ToolDef struct {
	Type     string          `json:"type"`
	Function ToolDefFunction `json:"function"`
}

// ToolDefFunction 工具函数定义
type ToolDefFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ChatRequest 模型请求
type ChatRequest struct {
	Model       string     `json:"model"`
	Messages    []Message  `json:"messages"`
	Tools       []ToolDef  `json:"tools,omitempty"`
	MaxTokens   int        `json:"max_tokens,omitempty"`
	Temperature float64    `json:"temperature,omitempty"`
	Stream      bool       `json:"stream"`
}

// ChatResponse 模型响应
type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Tokens    int        `json:"tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Delta      string     `json:"delta"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Done       bool       `json:"done"`
	Tokens     int        `json:"tokens"`
}

// Model 模型接口
type Model interface {
	// Name 返回模型名称
	Name() string
	// Chat 同步对话
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	// StreamChat 流式对话，返回chunk通道
	StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)
}

// ModelProvider 模型提供者工厂
type ModelProvider interface {
	Create(modelName string) (Model, error)
	ProviderName() string
}

// ProviderConfig 提供者配置（从数据库加载）
type ProviderConfig struct {
	Provider   string `json:"provider"`    // openai, local
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key,omitempty"`
	DefaultModel string `json:"default_model"`
}
