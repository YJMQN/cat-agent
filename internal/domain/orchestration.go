package domain

import (
	"time"
)

// ========== 多Agent协作编排模型 ==========

// Workflow 工作流定义
type Workflow struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Description string    `json:"description" gorm:"size:512"`
	// 工作流步骤定义 (JSON)
	StepsJSON   string    `json:"steps_json" gorm:"type:text"`
	// 输入参数定义 (JSON Schema)
	InputSchema string    `json:"input_schema" gorm:"type:text"`
	Status      string    `json:"status" gorm:"size:16;default:draft"` // draft, active, archived
	CreatedBy   uint      `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	ID          string   `json:"id"`              // 步骤唯一ID
	Name        string   `json:"name"`            // 步骤名称
	AgentID     uint     `json:"agent_id"`        // 执行该步骤的Agent
	InputFrom   []string `json:"input_from"`      // 输入来源（来自哪些步骤的输出）
	InputMap    map[string]string `json:"input_map"` // 输入参数映射
	Timeout     int      `json:"timeout"`         // 超时时间（秒），默认60
	RetryCount  int      `json:"retry_count"`     // 重试次数，默认0
	OnError     string   `json:"on_error"`        // 错误处理策略：skip, retry, stop
	Condition   string   `json:"condition"`       // 执行条件表达式
}

// WorkflowExecution 工作流执行记录
type WorkflowExecution struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	WorkflowID   uint      `json:"workflow_id"`
	UUID         string    `json:"uuid" gorm:"uniqueIndex;size:64"`
	UserID       uint      `json:"user_id"`
	InputJSON    string    `json:"input_json" gorm:"type:text"`    // 执行输入参数
	OutputJSON   string    `json:"output_json" gorm:"type:text"`   // 最终输出结果
	Status       string    `json:"status" gorm:"size:16"`          // pending, running, completed, failed, cancelled
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	Duration     int       `json:"duration"`                       // 执行时长（秒）
	CreatedAt    time.Time `json:"created_at"`
}

// StepExecution 步骤执行记录
type StepExecution struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	ExecutionID    uint      `json:"execution_id"`
	StepID         string    `json:"step_id" gorm:"size:64"`
	AgentID        uint      `json:"agent_id"`
	InputJSON      string    `json:"input_json" gorm:"type:text"`
	OutputJSON     string    `json:"output_json" gorm:"type:text"`
	Status         string    `json:"status" gorm:"size:16"`    // pending, running, completed, failed, skipped
	Error          string    `json:"error" gorm:"type:text"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
	RetryCount     int       `json:"retry_count"`
	CreatedAt      time.Time `json:"created_at"`
}