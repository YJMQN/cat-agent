package types

import "time"

// Role 定义消息角色
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message 表示对话消息
type Message struct {
	Role       Role            `json:"role"`
	Content    string          `json:"content"`
	Name       string          `json:"name,omitempty"`        // 工具调用时的名称
	ToolCallID string          `json:"tool_call_id,omitempty"` // 工具调用 ID
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`   // 助手发起的工具调用
	Metadata   map[string]any  `json:"metadata,omitempty"`     // 元数据
	Timestamp  time.Time       `json:"timestamp"`
}

// ToolCall 表示工具调用请求
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用详情
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// ToolResult 工具执行结果
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

// StreamChunk 流式输出块
type StreamChunk struct {
	Content      string     `json:"content,omitempty"`
	ToolCall     *ToolCall  `json:"tool_call,omitempty"`
	IsFinished   bool       `json:"is_finished"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

// Session 会话上下文
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Messages     []Message `json:"messages"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MaxTokens    int       `json:"max_tokens"`
	TokenCount   int       `json:"token_count"`
}

// UserPreferences 用户偏好（长期记忆）
type UserPreferences struct {
	UserID        string            `json:"user_id"`
	Language      string            `json:"language,omitempty"`
	Timezone      string            `json:"timezone,omitempty"`
	PreferredTools []string         `json:"preferred_tools,omitempty"`
	CustomSettings map[string]any   `json:"custom_settings,omitempty"`
	Summary       string            `json:"summary,omitempty"` // 历史对话摘要
}

// AgentRequest Agent 请求
type AgentRequest struct {
	SessionID      string   `json:"session_id"`
	UserID         string   `json:"user_id"`
	Message        string   `json:"message"`
	Stream         bool     `json:"stream,omitempty"`
	MaxIterations int      `json:"max_iterations,omitempty"` // 最大 ReAct 迭代次数
}

// AgentResponse Agent 响应
type AgentResponse struct {
	SessionID      string        `json:"session_id"`
	Response       string        `json:"response"`
	ToolCalls      []ToolCall    `json:"tool_calls,omitempty"`
	Iterations     int           `json:"iterations"`
	FinishReason   string        `json:"finish_reason"`
	Error          *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
