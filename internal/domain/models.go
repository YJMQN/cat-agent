package domain

import (
	"time"
)

// ========== 第三阶段新增：智能记忆系统模型 ==========

// MemoryItem 增强的记忆条目
type MemoryItem struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"index;not null"`
	SessionID    uint      `json:"session_id" gorm:"index"`
	Level        string    `json:"level" gorm:"size:16;default:short"` // working, short, long
	Type         string    `json:"type" gorm:"size:32"`                // summary, fact, preference, concept, event
	Content      string    `json:"content" gorm:"type:text;not null"`
	Keywords     string    `json:"keywords" gorm:"type:text"`     // 抽取的关键词
	Embedding    []byte    `json:"-" gorm:"type:blob"`            // 向量嵌入
	Importance   float64   `json:"importance" gorm:"default:0.5"` // 重要性评分
	AccessCount  int       `json:"access_count" gorm:"default:0"` // 访问次数
	LastAccessAt time.Time `json:"last_access_at"`                // 最后访问时间
	ExpiresAt    time.Time `json:"expires_at"`                    // 过期时间
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ========== 第三阶段新增：Cron任务模型 ==========

// CronJob 定时任务
type CronJob struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Description string    `json:"description" gorm:"size:512"`
	UserID      uint      `json:"user_id" gorm:"index;not null"`
	AgentID     uint      `json:"agent_id" gorm:"not null"`
	CronExpr    string    `json:"cron_expr" gorm:"size:64;not null"`
	Prompt      string    `json:"prompt" gorm:"type:text"`
	Status      string    `json:"status" gorm:"size:16;default:active"` // active, paused, expired
	LastRunAt   time.Time `json:"last_run_at"`
	NextRunAt   time.Time `json:"next_run_at"`
	RunCount    int       `json:"run_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CronLog Cron执行日志
type CronLog struct {
	ID       uint      `json:"id" gorm:"primaryKey"`
	JobID    uint      `json:"job_id" gorm:"index;not null"`
	Status   string    `json:"status" gorm:"size:16"` // success, failed
	Result   string    `json:"result" gorm:"type:text"`
	Error    string    `json:"error" gorm:"type:text"`
	Duration int       `json:"duration"` // 执行耗时(毫秒)
	RunAt    time.Time `json:"run_at"`
}

// ========== 第三阶段新增：动态插件模型 ==========

// Plugin 动态插件
type Plugin struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"uniqueIndex;size:64;not null"`
	Description  string    `json:"description" gorm:"size:512"`
	PluginType   string    `json:"plugin_type" gorm:"size:16"` // http, script, webhook
	Endpoint     string    `json:"endpoint" gorm:"size:512"`   // HTTP/Webhook端点
	Method       string    `json:"method" gorm:"size:16;default:POST"`
	Headers      string    `json:"headers" gorm:"type:text"`
	ScriptLang   string    `json:"script_lang" gorm:"size:16"` // python, bash, node
	ScriptCode   string    `json:"script_code" gorm:"type:text"`
	ParamsSchema string    `json:"params_schema" gorm:"type:text"`
	Timeout      int       `json:"timeout" gorm:"default:30"`
	RetryCount   int       `json:"retry_count" gorm:"default:0"`
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ========== 第四阶段新增：文档与RAG模型 ==========

// Document 用户文档
type Document struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"index;not null"`
	Filename   string    `json:"filename" gorm:"size:256;not null"`
	FileType   string    `json:"file_type" gorm:"size:32"` // pdf, docx, xlsx, txt, md, image
	FileSize   int64     `json:"file_size"`
	FilePath   string    `json:"file_path" gorm:"size:512"`
	Content    string    `json:"content" gorm:"type:text"` // 提取的文本内容
	Embedding  []byte    `json:"-" gorm:"type:blob"`       // 向量嵌入
	ChunkCount int       `json:"chunk_count" gorm:"default:0"`
	Status     string    `json:"status" gorm:"size:16;default:processing"` // processing, ready, failed
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DocumentChunk 文档分块
type DocumentChunk struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	DocumentID uint      `json:"document_id" gorm:"index;not null"`
	UserID     uint      `json:"user_id" gorm:"index"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	Embedding  []byte    `json:"-" gorm:"type:blob"` // 向量嵌入
	CreatedAt  time.Time `json:"created_at"`
}

// ========== 第四阶段新增：对话导出 ==========

// ExportRecord 导出记录
type ExportRecord struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SessionID uint      `json:"session_id" gorm:"index;not null"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	Format    string    `json:"format" gorm:"size:16"` // markdown, json, pdf
	FilePath  string    `json:"file_path" gorm:"size:512"`
	FileSize  int64     `json:"file_size"`
	Status    string    `json:"status" gorm:"size:16;default:processing"`
	CreatedAt time.Time `json:"created_at"`
}

// ========== 第四阶段新增：Token预算 ==========

// TokenBudget Token预算
type TokenBudget struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	DailyLimit     int64     `json:"daily_limit" gorm:"default:1000000"`    // 每日Token上限
	MonthlyLimit   int64     `json:"monthly_limit" gorm:"default:30000000"` // 每月Token上限
	AlertThreshold float64   `json:"alert_threshold" gorm:"default:0.8"`    // 告警阈值
	DailyUsed      int64     `json:"daily_used" gorm:"default:0"`
	MonthlyUsed    int64     `json:"monthly_used" gorm:"default:0"`
	LastResetDate  string    `json:"last_reset_date" gorm:"size:16"` // YYYY-MM-DD
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// User 用户模型
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:64;not null"`
	Password  string    `json:"-" gorm:"size:256;not null"`
	Role      string    `json:"role" gorm:"size:32;default:user"` // admin, operator, user
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GlobalModelConfig 全局模型配置（替代环境变量）
type GlobalModelConfig struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Provider     string    `json:"provider" gorm:"uniqueIndex;size:32;not null"` // openai, local
	BaseURL      string    `json:"base_url" gorm:"size:256"`
	APIKey       string    `json:"api_key" gorm:"size:512"`
	DefaultModel string    `json:"default_model" gorm:"size:128"`
	IsDefault    bool      `json:"is_default" gorm:"default:false"` // 是否默认提供商
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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

// ========== 功能1：用户画像系统 ==========

// UserProfile 用户画像主表
type UserProfile struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	UserID            uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	OnboardingStatus  string    `json:"onboarding_status" gorm:"size:16;default:pending"` // pending, in_progress, completed
	OnboardingStep    int       `json:"onboarding_step" gorm:"default:0"`
	Occupation        string    `json:"occupation" gorm:"size:128"`        // 职业
	Interests         string    `json:"interests" gorm:"type:text"`        // JSON array
	ExpertiseAreas    string    `json:"expertise_areas" gorm:"type:text"`  // 专业领域 JSON array
	InteractionStyle  string    `json:"interaction_style" gorm:"size:32"`  // concise, detailed, formal, casual
	CommunicationTone string    `json:"communication_tone" gorm:"size:32"` // professional, friendly, academic, casual
	PreferredLanguage string    `json:"preferred_language" gorm:"size:16;default:zh-CN"`
	ToolProficiency   string    `json:"tool_proficiency" gorm:"type:text"`  // 工具使用习惯 JSON map
	KnowledgeDomains  string    `json:"knowledge_domains" gorm:"type:text"` // 知识领域偏好 JSON array
	LearningStyle     string    `json:"learning_style" gorm:"size:32"`      // visual, auditory, reading, kinesthetic
	ResponseTime      string    `json:"response_time" gorm:"size:32"`       // brief, moderate, detailed
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ProfileDimension 用户画像多维度属性
type ProfileDimension struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"index;not null"`
	DimensionKey string    `json:"dimension_key" gorm:"size:64;not null"` // e.g., "communication_frequency", "tool_usage_frequency"
	Value        string    `json:"value" gorm:"type:text"`                // 多态值：可以是字符串、数字、JSON
	Score        float64   `json:"score" gorm:"default:0.5"`              // 评分 0-1
	Evidence     string    `json:"evidence" gorm:"type:text"`             // 支撑证据（来自哪些交互）
	LastUpdated  time.Time `json:"last_updated"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// OnboardingQuestion 引导式问卷题库
type OnboardingQuestion struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Step         int       `json:"step" gorm:"index;not null"`       // 步骤序号 1-N
	Category     string    `json:"category" gorm:"size:32;not null"` // profile, style, expertise, preference
	Question     string    `json:"question" gorm:"type:text;not null"`
	QuestionType string    `json:"question_type" gorm:"size:32"` // single_choice, multiple_choice, free_text, scale
	Options      string    `json:"options" gorm:"type:text"`     // JSON array (如果是选择题)
	ScaleMin     int       `json:"scale_min"`                    // 量表题最小值
	ScaleMax     int       `json:"scale_max"`                    // 量表题最大值
	MapToField   string    `json:"map_to_field" gorm:"size:64"`  // 映射到UserProfile的字段
	IsRequired   bool      `json:"is_required" gorm:"default:true"`
	Description  string    `json:"description" gorm:"size:256"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProfileUpdate 用户画像更新日志
type ProfileUpdate struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	FieldName string    `json:"field_name" gorm:"size:64;not null"`
	OldValue  string    `json:"old_value" gorm:"type:text"`
	NewValue  string    `json:"new_value" gorm:"type:text"`
	Source    string    `json:"source" gorm:"size:32"`   // manual, feedback, inferred, onboarding
	Reason    string    `json:"reason" gorm:"type:text"` // 为什么更新
	SessionID uint      `json:"session_id"`              // 关联的会话
	CreatedAt time.Time `json:"created_at"`
}

// ========== 功能2：情景记忆增强 ==========

// EpisodicMemory 情景记忆
type EpisodicMemory struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	UserID             uint      `json:"user_id" gorm:"index;not null"`
	SessionID          uint      `json:"session_id" gorm:"index;not null"`
	EpisodeID          string    `json:"episode_id" gorm:"size:64;index"`       // 关联的交互情景ID
	StartMessageID     uint      `json:"start_message_id" gorm:"index"`         // 情景的开始消息
	EndMessageID       uint      `json:"end_message_id" gorm:"index"`           // 情景的结束消息
	Context            string    `json:"context" gorm:"type:text"`              // 完整的对话上下文
	Summary            string    `json:"summary" gorm:"type:text"`              // LLM自动总结
	CompressedInfo     string    `json:"compressed_info" gorm:"type:text"`      // 压缩后的关键信息
	KeyPoints          string    `json:"key_points" gorm:"type:text"`           // JSON array of key points
	UserPreferences    string    `json:"user_preferences" gorm:"type:text"`     // 用户偏好提取
	TaskCompletionMode string    `json:"task_completion_mode" gorm:"type:text"` // 任务完成方式记录
	CommonProblems     string    `json:"common_problems" gorm:"type:text"`      // JSON array of identified issues
	IsCompressed       bool      `json:"is_compressed" gorm:"default:false"`    // 是否已压缩
	Embedding          []byte    `json:"-" gorm:"type:blob"`                    // 向量嵌入
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ========== 功能3：反馈系统 ==========

// UserFeedback 显式用户反馈
type UserFeedback struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"user_id" gorm:"index;not null"`
	SessionID     uint      `json:"session_id" gorm:"index;not null"`
	MessageID     uint      `json:"message_id" gorm:"index"`
	FeedbackType  string    `json:"feedback_type" gorm:"size:16"` // like, dislike, rating, comment
	Rating        int       `json:"rating"`                       // 1-5 分值
	Comment       string    `json:"comment" gorm:"type:text"`
	Aspects       string    `json:"aspects" gorm:"type:text"` // JSON - 评分的多个维度
	TitleAccuracy int       `json:"title_accuracy"`           // 回复准确度 1-5
	StyleMatch    int       `json:"style_match"`              // 风格匹配度 1-5
	ToolChoice    int       `json:"tool_choice"`              // 工具选择是否恰当 1-5
	CreatedAt     time.Time `json:"created_at"`
}

// ImplicitFeedback 隐式反馈信号
type ImplicitFeedback struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"index;not null"`
	SessionID       uint      `json:"session_id" gorm:"index"`
	SignalType      string    `json:"signal_type" gorm:"size:32"` // task_completion, repeat_request, interaction_duration, copy_action
	Value           float64   `json:"value"`                      // 信号值
	Context         string    `json:"context" gorm:"type:text"`   // 背景信息
	SessionDuration int       `json:"session_duration"`           // 会话时长（秒）
	IsPositive      bool      `json:"is_positive"`                // 是否是正向信号
	CreatedAt       time.Time `json:"created_at"`
}

// FeedbackAnalysis 反馈分析结果
type FeedbackAnalysis struct {
	ID                     uint      `json:"id" gorm:"primaryKey"`
	UserID                 uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	AnalysisDate           string    `json:"analysis_date" gorm:"size:16"`       // YYYY-MM-DD
	OverallSatisfaction    float64   `json:"overall_satisfaction"`               // 0-1
	StylePreferences       string    `json:"style_preferences" gorm:"type:text"` // JSON
	ToolPreferences        string    `json:"tool_preferences" gorm:"type:text"`  // JSON
	FrequentIssues         string    `json:"frequent_issues" gorm:"type:text"`   // JSON array
	ResponseStyleRating    float64   `json:"response_style_rating"`              // 回复风格评分 0-1
	ToolSelectionAccuracy  float64   `json:"tool_selection_accuracy"`            // 工具选择准确率 0-1
	ProcessedFeedbackCount int       `json:"processed_feedback_count"`           // 本周期处理的反馈数
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// ========== 功能4：Agent评估框架 ==========

// AgentMetrics Agent性能指标
type AgentMetrics struct {
	ID                        uint      `json:"id" gorm:"primaryKey"`
	AgentID                   uint      `json:"agent_id" gorm:"index;not null"`
	MetricsDate               string    `json:"metrics_date" gorm:"size:16;index"` // YYYY-MM-DD
	TaskCompletionRate        float64   `json:"task_completion_rate"`              // 任务完成率 0-1
	TaskQualityScore          float64   `json:"task_quality_score"`                // 任务质量评分 0-1
	AverageStepsPerTask       float64   `json:"average_steps_per_task"`            // 平均步骤数
	UnnecessaryToolCallRate   float64   `json:"unnecessary_tool_call_rate"`        // 不必要的工具调用率 0-1
	ToolSelectionAccuracy     float64   `json:"tool_selection_accuracy"`           // 工具选择正确率 0-1
	MemoryRetrievalHitRate    float64   `json:"memory_retrieval_hit_rate"`         // 记忆检索命中率 0-1
	MemoryRelevanceScore      float64   `json:"memory_relevance_score"`            // 记忆相关度 0-1
	PersonalizationAlignScore float64   `json:"personalization_align_score"`       // 个性化对齐度 0-1
	AverageLatencyMs          float64   `json:"average_latency_ms"`                // 平均延迟（毫秒）
	TotalTokenUsed            int64     `json:"total_token_used"`                  // 该周期总Token数
	ErrorRate                 float64   `json:"error_rate"`                        // 错误率 0-1
	CostEfficiency            float64   `json:"cost_efficiency"`                   // 成本效率 0-1 (任务数/Token数的归一化)
	SessionCount              int       `json:"session_count"`                     // 该周期会话数
	MessageCount              int       `json:"message_count"`                     // 该周期消息数
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// EvaluationResult 评估结果
type EvaluationResult struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	AgentID          uint      `json:"agent_id" gorm:"index;not null"`
	SessionID        uint      `json:"session_id" gorm:"index;not null"`
	EvaluationDate   time.Time `json:"evaluation_date"`
	GoalAchievement  float64   `json:"goal_achievement"`                 // 目标达成度 0-1
	PlanQuality      float64   `json:"plan_quality"`                     // 计划质量 0-1
	ExecutionQuality float64   `json:"execution_quality"`                // 执行质量 0-1
	FailurePoint     string    `json:"failure_point" gorm:"size:32"`     // goal, plan, action, none
	FailureReason    string    `json:"failure_reason" gorm:"type:text"`  // 失败原因
	OverallScore     float64   `json:"overall_score"`                    // 总分 0-1
	Recommendations  string    `json:"recommendations" gorm:"type:text"` // JSON array of improvement suggestions
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DimensionalScore 多维度评分
type DimensionalScore struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	EvaluationID uint      `json:"evaluation_id" gorm:"index;not null"`
	Dimension    string    `json:"dimension" gorm:"size:64;not null"` // task_quality, step_efficiency, tool_selection, etc.
	Score        float64   `json:"score"`                             // 0-1
	Weight       float64   `json:"weight"`                            // 权重
	Evidence     string    `json:"evidence" gorm:"type:text"`         // 支撑证据
	CreatedAt    time.Time `json:"created_at"`
}

// ========== 功能5：可观测性 ==========

// ExecutionTrace 执行轨迹（完整对话的追踪）
type ExecutionTrace struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	TraceID        string    `json:"trace_id" gorm:"uniqueIndex;size:64;not null"` // OpenTelemetry TraceID
	UserID         uint      `json:"user_id" gorm:"index;not null"`
	SessionID      uint      `json:"session_id" gorm:"index;not null"`
	AgentID        uint      `json:"agent_id" gorm:"index"`
	StartTime      time.Time `json:"start_time" gorm:"index"`
	EndTime        time.Time `json:"end_time"`
	DurationMs     int64     `json:"duration_ms"`           // 总耗时
	StepCount      int       `json:"step_count"`            // Agent执行步骤数
	ToolCallCount  int       `json:"tool_call_count"`       // 工具调用数
	Status         string    `json:"status" gorm:"size:16"` // success, error, timeout
	ErrorMessage   string    `json:"error_message" gorm:"type:text"`
	InputTokens    int       `json:"input_tokens"`                   // 输入Token数
	OutputTokens   int       `json:"output_tokens"`                  // 输出Token数
	TotalTokens    int       `json:"total_tokens"`                   // 总Token数
	ModelLatencyMs int64     `json:"model_latency_ms"`               // 模型响应延迟
	ParentTraceID  string    `json:"parent_trace_id" gorm:"size:64"` // 父轨迹ID (用于多Agent协作)
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TraceSpan 追踪跨度（ExecutionTrace中的单个操作）
type TraceSpan struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	TraceID       string    `json:"trace_id" gorm:"index;size:64;not null"`
	SpanID        string    `json:"span_id" gorm:"uniqueIndex;size:64;not null"` // OpenTelemetry SpanID
	ParentSpanID  string    `json:"parent_span_id" gorm:"size:64"`
	OperationType string    `json:"operation_type" gorm:"size:32"` // model_call, tool_call, memory_retrieval, decision
	OperationName string    `json:"operation_name" gorm:"size:128"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	DurationMs    int64     `json:"duration_ms"`
	Status        string    `json:"status" gorm:"size:16"` // success, error
	ErrorMessage  string    `json:"error_message" gorm:"type:text"`
	Input         string    `json:"input" gorm:"type:text"`    // 输入数据（JSON序列化）
	Output        string    `json:"output" gorm:"type:text"`   // 输出数据（JSON序列化）
	Metadata      string    `json:"metadata" gorm:"type:text"` // 附加元数据（JSON）
	CreatedAt     time.Time `json:"created_at"`
}

// ToolMetrics 工具性能指标
type ToolMetrics struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	ToolName         string    `json:"tool_name" gorm:"index;size:128;not null"`
	MetricsDate      string    `json:"metrics_date" gorm:"size:16;index"` // YYYY-MM-DD
	CallCount        int       `json:"call_count"`                        // 调用次数
	SuccessCount     int       `json:"success_count"`                     // 成功次数
	FailureCount     int       `json:"failure_count"`                     // 失败次数
	SuccessRate      float64   `json:"success_rate"`                      // 成功率 0-1
	AverageLatencyMs float64   `json:"average_latency_ms"`                // 平均延迟
	MaxLatencyMs     int64     `json:"max_latency_ms"`                    // 最大延迟
	MinLatencyMs     int64     `json:"min_latency_ms"`                    // 最小延迟
	TotalLatencyMs   int64     `json:"total_latency_ms"`                  // 总耗时
	ErrorRate        float64   `json:"error_rate"`                        // 错误率 0-1
	LastUsedAt       time.Time `json:"last_used_at"`                      // 最后使用时间
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ModelMetrics 模型性能指标
type ModelMetrics struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	ModelName         string    `json:"model_name" gorm:"index;size:128;not null"`
	Provider          string    `json:"provider" gorm:"size:32"`           // openai, local, etc.
	MetricsDate       string    `json:"metrics_date" gorm:"size:16;index"` // YYYY-MM-DD
	CallCount         int       `json:"call_count"`                        // 调用次数
	SuccessCount      int       `json:"success_count"`
	FailureCount      int       `json:"failure_count"`
	SuccessRate       float64   `json:"success_rate"`       // 成功率 0-1
	AverageLatencyMs  float64   `json:"average_latency_ms"` // 平均响应时间
	MaxLatencyMs      int64     `json:"max_latency_ms"`
	MinLatencyMs      int64     `json:"min_latency_ms"`
	TotalLatencyMs    int64     `json:"total_latency_ms"`
	TotalInputTokens  int64     `json:"total_input_tokens"`
	TotalOutputTokens int64     `json:"total_output_tokens"`
	AvgInputTokens    float64   `json:"avg_input_tokens"`  // 平均输入Token数
	AvgOutputTokens   float64   `json:"avg_output_tokens"` // 平均输出Token数
	QualityScore      float64   `json:"quality_score"`     // 从用户反馈推导的质量评分 0-1
	LastUsedAt        time.Time `json:"last_used_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
