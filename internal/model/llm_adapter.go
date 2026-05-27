package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LLMProvider 统一的大模型提供者
type LLMProvider struct {
	provider string
	BaseURL  string
	APIKey   string
	Client   *http.Client
}

// NewLLMProvider 创建通用LLM提供者
func NewLLMProvider(provider, baseURL, apiKey string) *LLMProvider {
	return &LLMProvider{
		provider: normalizeProvider(provider),
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Client:   DefaultHTTPClient(),
	}
}

// NewLLMProviderFromConfig 从数据库配置创建通用LLM提供者
func NewLLMProviderFromConfig(cfg *ProviderConfig) *LLMProvider {
	if cfg == nil {
		return NewLLMProvider("openai", "https://api.openai.com/v1", "")
	}
	return NewLLMProvider(cfg.Provider, cfg.BaseURL, cfg.APIKey)
}

// NewLLMProviderWithClient 使用自定义HTTP客户端创建
func NewLLMProviderWithClient(provider, baseURL, apiKey string, client *http.Client) *LLMProvider {
	return &LLMProvider{
		provider: normalizeProvider(provider),
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Client:   client,
	}
}

// NewOpenAIProvider 兼容旧API的OpenAI提供者创建函数
func NewOpenAIProvider(baseURL, apiKey string) *LLMProvider {
	return NewLLMProvider("openai", baseURL, apiKey)
}

// NewOpenAIProviderFromConfig 兼容旧API的OpenAI提供者配置创建函数
func NewOpenAIProviderFromConfig(cfg *ProviderConfig) *LLMProvider {
	if cfg == nil {
		return NewOpenAIProvider("https://api.openai.com/v1", "")
	}
	return NewLLMProvider("openai", cfg.BaseURL, cfg.APIKey)
}

// NewOpenAIProviderWithClient 兼容旧API的OpenAI提供者创建函数
func NewOpenAIProviderWithClient(baseURL, apiKey string, client *http.Client) *LLMProvider {
	return NewLLMProviderWithClient("openai", baseURL, apiKey, client)
}

// NewLocalModelProvider 兼容旧API的本地模型提供者创建函数
func NewLocalModelProvider(baseURL string) *LLMProvider {
	return NewLLMProvider("local", baseURL, "")
}

// NewLocalModelProviderFromConfig 兼容旧API的本地模型提供者配置创建函数
func NewLocalModelProviderFromConfig(cfg *ProviderConfig) *LLMProvider {
	if cfg == nil {
		return NewLocalModelProvider("http://localhost:11434")
	}
	return NewLLMProvider("local", cfg.BaseURL, cfg.APIKey)
}

// NewLocalModelProviderWithClient 兼容旧API的本地模型提供者创建函数
func NewLocalModelProviderWithClient(baseURL string, client *http.Client) *LLMProvider {
	return NewLLMProviderWithClient("local", baseURL, "", client)
}

// NormalizeProvider 将供应商映射到实际 SDK 类型
func NormalizeProvider(provider string) string {
	return normalizeProvider(provider)
}

// DefaultProviderConfig 返回提供者的默认配置
func DefaultProviderConfig(provider string) (baseURL, apiKey, modelName string) {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1", "", "gpt-4o-mini"
	case "deepseek":
		return "https://api.deepseek.com/v1", "", "deepseek-chat"
	case "openrouter":
		return "https://openrouter.ai/api/v1", "", "openai/gpt-4o-mini"
	case "modelscope":
		return "https://api-inference.modelscope.cn/v1", "", "Qwen/Qwen2.5-7B-Instruct"
	case "local", "ollama":
		return "http://localhost:11434", "", "qwen2.5"
	default:
		return "https://api.openai.com/v1", "", "gpt-4o-mini"
	}
}

// SDKProviderName 返回用于实际 SDK 调用的 provider 名称
func SDKProviderName(provider string) string {
	switch provider {
	case "openrouter", "modelscope", "custom":
		return "openai"
	case "ollama":
		return "local"
	default:
		return provider
	}
}

func normalizeProvider(provider string) string {
	switch provider {
	case "ollama", "local":
		return "local"
	case "openai", "openrouter", "modelscope", "custom", "deepseek":
		return "openai"
	default:
		return provider
	}
}

func (p *LLMProvider) ProviderName() string { return p.provider }

func (p *LLMProvider) Create(modelName string) (Model, error) {
	return &LLMModel{
		provider: p.provider,
		baseURL:  p.BaseURL,
		apiKey:   p.APIKey,
		model:    modelName,
		client:   p.Client,
	}, nil
}

// LLMModel 统一的大模型实现
type LLMModel struct {
	provider string
	baseURL  string
	apiKey   string
	model    string
	client   *http.Client
}

func (m *LLMModel) Name() string { return m.model }

func (m *LLMModel) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	switch m.provider {
	case "local":
		return m.chatLocal(ctx, req)
	default:
		return m.chatOpenAI(ctx, req)
	}
}

func (m *LLMModel) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	switch m.provider {
	case "local":
		return m.streamChatLocal(ctx, req)
	default:
		return m.streamChatOpenAI(ctx, req)
	}
}

func (m *LLMModel) chatLocal(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	ollamaReq := m.convertLocalRequest(req, false)
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求本地模型失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("本地模型返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	var lastResp ollamaResponse
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var r ollamaResponse
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		lastResp = r
		if r.Done {
			break
		}
	}

	var toolCalls []ToolCall
	if len(lastResp.Message.ToolCalls) > 0 {
		for _, tc := range lastResp.Message.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:   fmt.Sprintf("call_%s_%d", tc.Function.Name, time.Now().UnixNano()),
				Type: "function",
				Function: FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return &ChatResponse{
		Content:   lastResp.Message.Content,
		ToolCalls: toolCalls,
		Tokens:    lastResp.EvalCount + lastResp.PromptEvalCount,
	}, nil
}

func (m *LLMModel) streamChatLocal(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	ollamaReq := m.convertLocalRequest(req, true)
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求本地模型失败: %w", err)
	}

	ch := make(chan StreamChunk, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			var r ollamaResponse
			if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
				continue
			}

			var toolCalls []ToolCall
			if len(r.Message.ToolCalls) > 0 {
				for _, tc := range r.Message.ToolCalls {
					toolCalls = append(toolCalls, ToolCall{
						ID:   fmt.Sprintf("call_%s_%d", tc.Function.Name, time.Now().UnixNano()),
						Type: "function",
						Function: FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}
			}

			ch <- StreamChunk{
				Delta:     r.Message.Content,
				ToolCalls: toolCalls,
				Done:      r.Done,
				Tokens:    r.EvalCount + r.PromptEvalCount,
			}

			if r.Done {
				break
			}
		}
	}()

	return ch, nil
}

func (m *LLMModel) chatOpenAI(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	oaiReq := m.convertOpenAIRequest(req, false)
	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求模型API失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("模型API返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("模型返回空响应")
	}

	choice := oaiResp.Choices[0]
	result := &ChatResponse{
		Content: choice.Message.Content,
		Tokens:  oaiResp.Usage.TotalTokens,
	}

	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return result, nil
}

func (m *LLMModel) streamChatOpenAI(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	oaiReq := m.convertOpenAIRequest(req, true)
	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if m.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求模型API失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("模型API返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	chunks := make(chan StreamChunk, 64)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		accumulatedToolCalls := make(map[string]*ToolCall)
		var toolCallsMu sync.Mutex

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				toolCallsMu.Lock()
				var finalToolCalls []ToolCall
				for _, tc := range accumulatedToolCalls {
					finalToolCalls = append(finalToolCalls, *tc)
				}
				toolCallsMu.Unlock()

				chunks <- StreamChunk{
					Delta:     "",
					ToolCalls: finalToolCalls,
					Done:      true,
				}
				break
			}

			var chunk openaiStreamResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			var toolCalls []ToolCall
			if len(chunk.Choices) > 0 {
				if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
					for _, tc := range chunk.Choices[0].Delta.ToolCalls {
						toolCalls = append(toolCalls, ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: FunctionCall{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						})
					}
				}
			}

			if len(toolCalls) > 0 {
				toolCallsMu.Lock()
				for i := range toolCalls {
					accumulatedToolCalls[toolCalls[i].ID] = &toolCalls[i]
				}
				toolCallsMu.Unlock()
			}

			sc := StreamChunk{
				Done: false,
			}
			if len(chunk.Choices) > 0 {
				sc.Delta = chunk.Choices[0].Delta.Content
			}
			if chunk.Usage != nil {
				sc.Tokens = chunk.Usage.TotalTokens
			}
			if len(toolCalls) > 0 {
				sc.ToolCalls = toolCalls
			}

			select {
			case chunks <- sc:
			case <-ctx.Done():
				return
			}
		}
	}()

	return chunks, nil
}

func (m *LLMModel) convertLocalRequest(req *ChatRequest, stream bool) *ollamaRequest {
	ollamaReq := &ollamaRequest{
		Model:  m.model,
		Stream: stream,
	}

	for _, msg := range req.Messages {
		ollamaMsg := ollamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.Role == "tool" && len(msg.ToolCalls) > 0 {
			ollamaMsg.ToolCalls = make([]ollamaToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				ollamaMsg.ToolCalls[i] = ollamaToolCall{
					Function: ollamaFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMsg)
	}

	if len(req.Tools) > 0 {
		ollamaReq.Tools = make([]ollamaTool, len(req.Tools))
		for i, tool := range req.Tools {
			ollamaReq.Tools[i] = ollamaTool{
				Type: "function",
				Function: ollamaToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters.(map[string]interface{}),
				},
			}
		}
	}

	if req.MaxTokens > 0 || req.Temperature > 0 {
		ollamaReq.Options = &ollamaOptions{
			NumPredict:  req.MaxTokens,
			Temperature: req.Temperature,
		}
	}

	return ollamaReq
}

func (m *LLMModel) convertOpenAIRequest(req *ChatRequest, stream bool) *openaiRequest {
	oaiReq := &openaiRequest{
		Model:       m.model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      stream,
	}

	for _, msg := range req.Messages {
		om := openaiMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		for _, tc := range msg.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, openaiToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: openaiFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		oaiReq.Messages = append(oaiReq.Messages, om)
	}

	for _, td := range req.Tools {
		oaiReq.Tools = append(oaiReq.Tools, openaiTool{
			Type: "function",
			Function: openaiToolFunction{
				Name:        td.Function.Name,
				Description: td.Function.Description,
				Parameters:  td.Function.Parameters,
			},
		})
	}

	return oaiReq
}

// ========== 共用 HTTP 客户端配置 ==========

func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          10,
			IdleConnTimeout:       30 * time.Second,
			DisableCompression:    false,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
}

// ========== Local / Ollama 数据结构 ==========

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaTool struct {
	Type     string             `json:"type"`
	Function ollamaToolFunction `json:"function"`
}

type ollamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ollamaToolCall struct {
	Function ollamaFunctionCall `json:"function"`
}

type ollamaFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ollamaOptions struct {
	NumPredict  int     `json:"num_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type ollamaResponse struct {
	Message struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done            bool `json:"done"`
	EvalCount       int  `json:"eval_count"`
	PromptEvalCount int  `json:"prompt_eval_count"`
}

// ========== OpenAI 兼容数据结构 ==========

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Tools       []openaiTool    `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiTool struct {
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiFunctionCall `json:"function"`
}

type openaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openaiToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type openaiStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content   string           `json:"content"`
			ToolCalls []openaiToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}
