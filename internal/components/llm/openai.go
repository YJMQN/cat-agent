package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-agent-system/pkg/types"
)

// OpenAIFactory OpenAI 模型工厂
type OpenAIFactory struct{}

// CreateAdapter 创建 OpenAI 适配器
func (f *OpenAIFactory) CreateAdapter(config *ModelConfig) (Adapter, error) {
	return NewOpenAIAdapter(config)
}

// OpenAIAdapter OpenAI API 适配器实现
type OpenAIAdapter struct {
	config      *ModelConfig
	httpClient  *http.Client
	baseURL     string
	model       string
}

// openAIChatRequest OpenAI API 请求格式
type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// openAIMessage OpenAI 消息格式
type openAIMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	Name       string          `json:"name,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

// openAITool OpenAI 工具定义
type openAITool struct {
	Type     string          `json:"type"`
	Function openAIFunctionDef `json:"function"`
}

// openAIFunctionDef OpenAI 函数定义
type openAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// openAIToolCall OpenAI 工具调用
type openAIToolCall struct {
	ID       string                `json:"id"`
	Type     string                `json:"type"`
	Function openAIFunctionCall    `json:"function"`
}

// openAIFunctionCall OpenAI 函数调用
type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// openAIChatResponse OpenAI API 响应格式
type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int             `json:"index"`
		Message      openAIMessage   `json:"message"`
		FinishReason string          `json:"finish_reason"`
		Delta        openAIMessage   `json:"delta,omitempty"` // 流式响应
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// openAIStreamChunk OpenAI 流式响应块
type openAIStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Delta        openAIMessage `json:"delta"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
}

// NewOpenAIAdapter 创建 OpenAI 适配器
func NewOpenAIAdapter(config *ModelConfig) (*OpenAIAdapter, error) {
	baseURL := config.Endpoint
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSecs) * time.Second,
		},
		baseURL: baseURL,
		model:   config.Model,
	}, nil
}

// Chat 发送聊天请求
func (a *OpenAIAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	openAIReq := a.convertRequest(req)
	
	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	a.setHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error [%d]: %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	choice := openAIResp.Choices[0]
	message := a.convertMessage(choice.Message)

	chatResp := &ChatResponse{
		Message:      message,
		FinishReason: choice.FinishReason,
	}

	if openAIResp.Usage != nil {
		chatResp.Usage = TokenUsage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		}
	}

	return chatResp, nil
}

// ChatStream 发送流式聊天请求
func (a *OpenAIAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	req.Stream = true
	openAIReq := a.convertRequest(req)

	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	a.setHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error [%d]: %s", resp.StatusCode, string(body))
	}

	eventChan := make(chan StreamEvent, 32)

	go func() {
		defer resp.Body.Close()
		defer close(eventChan)

		decoder := json.NewDecoder(resp.Body)
		
		for {
			select {
			case <-ctx.Done():
				eventChan <- StreamEvent{Error: ctx.Err()}
				return
			default:
			}

			var chunk openAIStreamChunk
			if err := decoder.Decode(&chunk); err != nil {
				if err == io.EOF {
					eventChan <- StreamEvent{
						Chunk: &types.StreamChunk{IsFinished: true, FinishReason: "stop"},
					}
					return
				}
				eventChan <- StreamEvent{Error: fmt.Errorf("decode chunk: %w", err)}
				return
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta
			
			streamChunk := &types.StreamChunk{
				Content:      delta.Content,
				IsFinished:   chunk.Choices[0].FinishReason != "",
				FinishReason: chunk.Choices[0].FinishReason,
			}

			// 处理工具调用
			if len(delta.ToolCalls) > 0 {
				tc := delta.ToolCalls[0]
				streamChunk.ToolCall = &types.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: types.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}

			eventChan <- StreamEvent{Chunk: streamChunk}

			if chunk.Choices[0].FinishReason != "" {
				break
			}
		}
	}()

	return eventChan, nil
}

// GetModelName 获取模型名称
func (a *OpenAIAdapter) GetModelName() string {
	return a.model
}

// Close 关闭连接
func (a *OpenAIAdapter) Close() error {
	// HTTP client 不需要特别清理
	return nil
}

// setHeaders 设置请求头
func (a *OpenAIAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)
}

// convertRequest 转换请求格式
func (a *OpenAIAdapter) convertRequest(req *ChatRequest) *openAIChatRequest {
	messages := make([]openAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = a.convertMessageToOpenAI(msg)
	}

	tools := make([]openAITool, len(req.Tools))
	for i, tool := range req.Tools {
		tools[i] = openAITool{
			Type: tool.Type,
			Function: openAIFunctionDef{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}

	temperature := a.config.Temperature
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	maxTokens := a.config.MaxTokens
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	return &openAIChatRequest{
		Model:       a.model,
		Messages:    messages,
		Tools:       tools,
		Stream:      req.Stream,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
}

// convertMessageToOpenAI 转换消息为 OpenAI 格式
func (a *OpenAIAdapter) convertMessageToOpenAI(msg types.Message) openAIMessage {
	toolCalls := make([]openAIToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		toolCalls[i] = openAIToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: openAIFunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}

	return openAIMessage{
		Role:       string(msg.Role),
		Content:    msg.Content,
		Name:       msg.Name,
		ToolCallID: msg.ToolCallID,
		ToolCalls:  toolCalls,
	}
}

// convertMessage 从 OpenAI 格式转换消息
func (a *OpenAIAdapter) convertMessage(msg openAIMessage) types.Message {
	toolCalls := make([]types.ToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		toolCalls[i] = types.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: types.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}

	return types.Message{
		Role:      types.Role(msg.Role),
		Content:   msg.Content,
		Name:      msg.Name,
		ToolCallID: msg.ToolCallID,
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
	}
}
