package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-agent-system/pkg/types"
)

// PrivateModelFactory 私有化模型工厂（示例：兼容 OpenAI 格式的私有部署）
type PrivateModelFactory struct{}

// CreateAdapter 创建私有模型适配器
func (f *PrivateModelFactory) CreateAdapter(config *ModelConfig) (Adapter, error) {
	return NewPrivateModelAdapter(config)
}

// PrivateModelAdapter 私有化模型适配器
// 支持任何兼容 OpenAI API 格式的私有部署模型（如 vLLM、TGI、LocalAI 等）
type PrivateModelAdapter struct {
	config     *ModelConfig
	httpClient *http.Client
	baseURL    string
	model      string
}

// NewPrivateModelAdapter 创建私有模型适配器
func NewPrivateModelAdapter(config *ModelConfig) (*PrivateModelAdapter, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for private model")
	}

	return &PrivateModelAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSecs) * time.Second,
		},
		baseURL: config.Endpoint,
		model:   config.Model,
	}, nil
}

// Chat 发送聊天请求
func (a *PrivateModelAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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
func (a *PrivateModelAdapter) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
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

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error [%d]: %s", resp.StatusCode, string(body))
	}

	eventChan := make(chan StreamEvent, 32)

	go func() {
		defer resp.Body.Close()
		defer close(eventChan)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				eventChan <- StreamEvent{Error: ctx.Err()}
				return
			default:
			}

			line := scanner.Text()
			if line == "" || line == "data: [DONE]" {
				eventChan <- StreamEvent{
					Chunk: &types.StreamChunk{IsFinished: true, FinishReason: "stop"},
				}
				return
			}

			if strings.HasPrefix(line, "data: ") {
				line = strings.TrimPrefix(line, "data: ")
			}

			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
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

			eventChan <- StreamEvent{Chunk: streamChunk}

			if chunk.Choices[0].FinishReason != "" {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			eventChan <- StreamEvent{Error: fmt.Errorf("scan error: %w", err)}
		}
	}()

	return eventChan, nil
}

// GetModelName 获取模型名称
func (a *PrivateModelAdapter) GetModelName() string {
	return a.model
}

// Close 关闭连接
func (a *PrivateModelAdapter) Close() error {
	return nil
}

// setHeaders 设置请求头
func (a *PrivateModelAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if a.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.config.APIKey)
	}
}

// convertRequest 转换请求格式
func (a *PrivateModelAdapter) convertRequest(req *ChatRequest) *openAIChatRequest {
	messages := make([]openAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = a.convertMessageToOpenAI(msg)
	}

	var tools []openAITool
	if len(req.Tools) > 0 {
		tools = make([]openAITool, len(req.Tools))
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
func (a *PrivateModelAdapter) convertMessageToOpenAI(msg types.Message) openAIMessage {
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
func (a *PrivateModelAdapter) convertMessage(msg openAIMessage) types.Message {
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
		Role:       types.Role(msg.Role),
		Content:    msg.Content,
		Name:       msg.Name,
		ToolCallID: msg.ToolCallID,
		ToolCalls:  toolCalls,
		Timestamp:  time.Now(),
	}
}
