package llm

import (
	"context"

	"ai-agent-system/pkg/types"
)

// ModelConfig 模型配置
type ModelConfig struct {
	Name         string  `json:"name"`          // 模型名称标识
	Endpoint     string  `json:"endpoint"`      // API 端点
	APIKey       string  `json:"api_key"`       // API 密钥
	Model        string  `json:"model"`         // 具体模型 ID
	Temperature  float64 `json:"temperature"`   // 温度参数
	MaxTokens    int     `json:"max_tokens"`    // 最大输出 token 数
	TimeoutSecs  int     `json:"timeout_secs"`  // 超时时间（秒）
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Messages    []types.Message `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
}

// ToolDefinition 工具定义（用于告知 LLM 可用工具）
type ToolDefinition struct {
	Type        string       `json:"type"`
	Function    FunctionDef  `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Message      types.Message
	Usage        TokenUsage
	FinishReason string
}

// TokenUsage Token 使用情况
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Adapter LLM 适配器接口 - 支持多种模型后端
type Adapter interface {
	// Chat 发送聊天请求并获取完整响应
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream 发送流式聊天请求
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)

	// GetModelName 获取当前模型名称
	GetModelName() string

	// Close 关闭连接，释放资源
	Close() error
}

// StreamEvent 流式事件
type StreamEvent struct {
	Chunk *types.StreamChunk
	Error error
}

// Factory 模型工厂接口
type Factory interface {
	CreateAdapter(config *ModelConfig) (Adapter, error)
}

// Registry 模型注册表
type Registry struct {
	factories map[string]Factory
	adapters  map[string]Adapter
}

// NewRegistry 创建模型注册表
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
		adapters:  make(map[string]Adapter),
	}
}

// RegisterFactory 注册模型工厂
func (r *Registry) RegisterFactory(name string, factory Factory) {
	r.factories[name] = factory
}

// CreateAdapter 创建适配器实例
func (r *Registry) CreateAdapter(name string, config *ModelConfig) (Adapter, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, ErrFactoryNotFound
	}
	adapter, err := factory.CreateAdapter(config)
	if err != nil {
		return nil, err
	}
	r.adapters[config.Name] = adapter
	return adapter, nil
}

// GetAdapter 获取已创建的适配器
func (r *Registry) GetAdapter(name string) (Adapter, error) {
	adapter, ok := r.adapters[name]
	if !ok {
		return nil, ErrAdapterNotFound
	}
	return adapter, nil
}

// 错误定义
var (
	ErrFactoryNotFound  = &LLMError{Code: "FACTORY_NOT_FOUND", Message: "模型工厂未注册"}
	ErrAdapterNotFound  = &LLMError{Code: "ADAPTER_NOT_FOUND", Message: "适配器实例不存在"}
	ErrStreamNotSupported = &LLMError{Code: "STREAM_NOT_SUPPORTED", Message: "当前模型不支持流式输出"}
)

// LLMError LLM 错误类型
type LLMError struct {
	Code    string
	Message string
	Err     error
}

func (e *LLMError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + " - " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *LLMError) Unwrap() error {
	return e.Err
}

// StreamReader 流式读取辅助工具
type StreamReader struct {
	channel <-chan StreamEvent
}

// NewStreamReader 创建流式读取器
func NewStreamReader(ch <-chan StreamEvent) *StreamReader {
	return &StreamReader{channel: ch}
}

// ReadAll 读取所有流式内容
func (sr *StreamReader) ReadAll() (string, error) {
	var content string
	for event := range sr.channel {
		if event.Error != nil {
			return content, event.Error
		}
		if event.Chunk != nil {
			content += event.Chunk.Content
		}
	}
	return content, nil
}

// ReadToChannel 将流式内容转发到输出 channel（用于 HTTP SSE）
func (sr *StreamReader) ReadToChannel(out chan<- string) error {
	defer close(out)
	for event := range sr.channel {
		if event.Error != nil {
			return event.Error
		}
		if event.Chunk != nil && event.Chunk.Content != "" {
			out <- event.Chunk.Content
		}
	}
	return nil
}

// MergeMessages 合并消息历史，处理 token 超限
func MergeMessages(messages []types.Message, maxTokens int) []types.Message {
	// 简单实现：保留 system 消息，然后从后往前保留直到接近 maxTokens
	if len(messages) == 0 {
		return messages
	}

	var result []types.Message
	var systemMsg *types.Message

	// 分离 system 消息
	for i, msg := range messages {
		if msg.Role == types.RoleSystem {
			systemMsg = &messages[i]
			break
		}
	}

	// 添加 system 消息
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}

	// 从后往前添加消息
	tokenCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == types.RoleSystem {
			continue
		}
		// 简化的 token 估算：每 4 个字符约 1 个 token
		estimatedTokens := len(messages[i].Content) / 4
		if tokenCount+estimatedTokens > maxTokens {
			break
		}
		result = append(result, messages[i])
		tokenCount += estimatedTokens
	}

	// 反转非 system 消息部分
	if len(result) > 1 {
		reversed := result[1:]
		for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
			reversed[i], reversed[j] = reversed[j], reversed[i]
		}
		result = append(result[:1], reversed...)
	}

	return result
}
