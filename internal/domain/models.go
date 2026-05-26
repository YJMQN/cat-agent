package domain

import (
	"time"
)

// User 用户模型
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:64;not null"`
	Password  string    `json:"-" gorm:"size:256;not null"`
	Role      string    `json:"role" gorm:"size:32;default:user"` // admin, operator, user
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	Name                 string    `json:"name" gorm:"size:128;not null"`
	Description          string    `json:"description" gorm:"size:512"`
	ModelProvider        string    `json:"model_provider" gorm:"size:32"`
	ModelName            string    `json:"model_name" gorm:"size:128"`
	UseGlobalModelConfig bool      `json:"use_global_model_config" gorm:"default:false"`
	SystemPrompt         string    `json:"system_prompt" gorm:"type:text"`
	MaxTokens            int       `json:"max_tokens" gorm:"default:4096"`
	Temperature          float64   `json:"temperature" gorm:"default:0.7"`
	ToolIDs              string    `json:"tool_ids" gorm:"type:text"`             // JSON array of tool IDs
	Status               string    `json:"status" gorm:"size:16;default:stopped"` // running, stopped
	CreatedBy            uint      `json:"created_by"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Tool 工具定义
type Tool struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"uniqueIndex;size:64;not null"`
	DisplayName string `json:"display_name" gorm:"size:128"`
	Description string `json:"description" gorm:"size:512"`
	ToolType    string `json:"tool_type" gorm:"size:32;not null"` // builtin, http, script
	// 参数Schema (JSON Schema格式)
	ParamsSchema string `json:"params_schema" gorm:"type:text"`
	// HTTP工具配置
	HTTPEndpoint string `json:"http_endpoint" gorm:"size:512"`
	HTTPMethod   string `json:"http_method" gorm:"size:16"`
	HTTPHeaders  string `json:"http_headers" gorm:"type:text"`
	// 脚本工具配置
	ScriptLang string `json:"script_lang" gorm:"size:32"`
	ScriptCode string `json:"script_code" gorm:"type:text"`
	// 状态
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Session 对话会话
type Session struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UUID      string    `json:"uuid" gorm:"uniqueIndex;size:64;not null"`
	AgentID   uint      `json:"agent_id"`
	UserID    uint      `json:"user_id"`
	Title     string    `json:"title" gorm:"size:256"`
	Status    string    `json:"status" gorm:"size:16;default:active"` // active, closed
	TokenUsed int       `json:"token_used" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message 对话消息
type Message struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	SessionID  uint      `json:"session_id" gorm:"index"`
	Role       string    `json:"role" gorm:"size:16;not null"` // user, assistant, system, tool
	Content    string    `json:"content" gorm:"type:text"`
	ToolCalls  string    `json:"tool_calls" gorm:"type:text"` // JSON
	ToolCallID string    `json:"tool_call_id" gorm:"size:128"`
	Tokens     int       `json:"tokens" gorm:"default:0"`
	CreatedAt  time.Time `json:"created_at"`
}

// Memory 长期记忆
type Memory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	SessionID uint      `json:"session_id"`
	Category  string    `json:"category" gorm:"size:32"` // profile, preference, summary, fact
	Key       string    `json:"key" gorm:"size:256;index"`
	Content   string    `json:"content" gorm:"type:text"`
	Source    string    `json:"source" gorm:"size:32"` // auto, manual
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditLog 操作审计
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	Action    string    `json:"action" gorm:"size:64;not null"`
	Resource  string    `json:"resource" gorm:"size:64"`
	Detail    string    `json:"detail" gorm:"type:text"`
	IP        string    `json:"ip" gorm:"size:64"`
	CreatedAt time.Time `json:"created_at"`
}

// StatsTokenUsage Token用量统计
type StatsTokenUsage struct {
	Date    string `json:"date"`
	Total   int    `json:"total"`
	Input   int    `json:"input"`
	Output  int    `json:"output"`
	AgentID uint   `json:"agent_id"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	AgentID uint   `json:"agent_id" binding:"required"`
	UserID  uint   `json:"user_id"`
	Content string `json:"content" binding:"required"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	SessionID uint   `json:"session_id"`
	Content   string `json:"content"`
	Done      bool   `json:"done"`
}

// StreamEvent SSE事件
type StreamEvent struct {
	Type           string `json:"type"` // text, tool_call, tool_result, tool_confirmation, error, done
	Content        string `json:"content"`
	Tool           string `json:"tool,omitempty"`
	Args           string `json:"args,omitempty"`
	ConfirmationID string `json:"confirmation_id,omitempty"`
}

// AdminStats 管理统计数据
type AdminStats struct {
	TotalSessions  int     `json:"total_sessions"`
	TotalMessages  int     `json:"total_messages"`
	TotalTokens    int     `json:"total_tokens"`
	TotalUsers     int     `json:"total_users"`
	SuccessRate    float64 `json:"success_rate"`
	AvgLatency     float64 `json:"avg_latency_ms"`
	ActiveSessions int     `json:"active_sessions"`
}
