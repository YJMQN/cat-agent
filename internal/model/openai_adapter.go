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
)

// OpenAIProvider OpenAI兼容模型提供者
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
}

// NewOpenAIProvider 创建OpenAI提供者
func NewOpenAIProvider(baseURL, apiKey string) *OpenAIProvider {
	return &OpenAIProvider{BaseURL: baseURL, APIKey: apiKey}
}

func (p *OpenAIProvider) ProviderName() string { return "openai" }

func (p *OpenAIProvider) Create(modelName string) (Model, error) {
	return &OpenAIModel{
		baseURL:  p.BaseURL,
		apiKey:   p.APIKey,
		model:    modelName,
	}, nil
}

// OpenAIModel OpenAI兼容模型实现
type OpenAIModel struct {
	baseURL string
	apiKey  string
	model   string
}

func (m *OpenAIModel) Name() string { return m.model }

// openaiRequest OpenAI API请求格式
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Tools       []openaiTool    `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type openaiMessage struct {
	Role       string              `json:"role"`
	Content    string              `json:"content,omitempty"`
	ToolCallID string              `json:"tool_call_id,omitempty"`
	ToolCalls  []openaiToolCall    `json:"tool_calls,omitempty"`
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

// openaiResponse OpenAI API响应格式
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

// openaiStreamResponse 流式响应
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

func (m *OpenAIModel) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	oaiReq := m.convertRequest(req, false)
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

	resp, err := http.DefaultClient.Do(httpReq)
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

	// 转换工具调用
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

func (m *OpenAIModel) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	oaiReq := m.convertRequest(req, true)
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

	resp, err := http.DefaultClient.Do(httpReq)
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

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunks <- StreamChunk{Done: true}
				return
			}

			var chunk openaiStreamResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			c := chunk.Choices[0]
			sc := StreamChunk{
				Delta: c.Delta.Content,
				Done:  c.FinishReason != nil && *c.FinishReason == "stop",
			}

			for _, tc := range c.Delta.ToolCalls {
				sc.ToolCalls = append(sc.ToolCalls, ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}

			if chunk.Usage != nil {
				sc.Tokens = chunk.Usage.TotalTokens
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

func (m *OpenAIModel) convertRequest(req *ChatRequest, stream bool) *openaiRequest {
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
