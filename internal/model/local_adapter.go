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
	"time"
)

// LocalModelProvider 本地模型提供者 (兼容Ollama API)
type LocalModelProvider struct {
	BaseURL string
	Client  *http.Client // 第二阶段新增：共享HTTP客户端
}

// NewLocalModelProvider 创建本地模型提供者
func NewLocalModelProvider(baseURL string) *LocalModelProvider {
	return &LocalModelProvider{
		BaseURL: baseURL,
		Client:  DefaultHTTPClient(),
	}
}

// NewLocalModelProviderWithClient 使用自定义HTTP客户端创建
func NewLocalModelProviderWithClient(baseURL string, client *http.Client) *LocalModelProvider {
	return &LocalModelProvider{
		BaseURL: baseURL,
		Client:  client,
	}
}

func (p *LocalModelProvider) ProviderName() string { return "local" }

func (p *LocalModelProvider) Create(modelName string) (Model, error) {
	return &LocalModel{
		baseURL: p.BaseURL,
		model:   modelName,
		client:  p.Client,
	}, nil
}

// LocalModel 本地模型实现 (Ollama兼容)
type LocalModel struct {
	baseURL string
	model   string
	client  *http.Client
}

func (m *LocalModel) Name() string { return m.model }

// ollamaRequest Ollama API请求 - 第二阶段修复：添加工具支持
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []ollamaTool    `json:"tools,omitempty"` // 第二阶段新增：工具定义
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"` // 第二阶段新增：工具调用结果
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

// ollamaResponse Ollama API响应 - 第二阶段修复：添加工具调用支持
type ollamaResponse struct {
	Message struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"` // 第二阶段新增
	} `json:"message"`
	Done               bool `json:"done"`
	EvalCount          int  `json:"eval_count"`
	PromptEvalCount    int  `json:"prompt_eval_count"`
}

func (m *LocalModel) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	ollamaReq := m.convertRequest(req, false)
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

	// Ollama返回流式JSON，读取最后一行
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

	// 第二阶段修复：转换工具调用
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

func (m *LocalModel) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	ollamaReq := m.convertRequest(req, true)
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

			// 第二阶段修复：转换工具调用
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

// convertRequest 转换请求格式 - 第二阶段修复：添加工具支持
func (m *LocalModel) convertRequest(req *ChatRequest, stream bool) *ollamaRequest {
	ollamaReq := &ollamaRequest{
		Model:  m.model,
		Stream: stream,
	}

	// 转换消息
	for _, msg := range req.Messages {
		ollamaMsg := ollamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		// 第二阶段修复：处理工具调用结果消息
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

	// 第二阶段修复：转换工具定义
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

	// 设置选项
	if req.MaxTokens > 0 || req.Temperature > 0 {
		ollamaReq.Options = &ollamaOptions{
			NumPredict:  req.MaxTokens,
			Temperature: req.Temperature,
		}
	}

	return ollamaReq
}

// ========== 第二阶段新增：HTTP客户端配置 ==========

// DefaultHTTPClient 返回默认HTTP客户端
func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
}