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

// LocalModelProvider 本地模型提供者 (兼容Ollama API)
type LocalModelProvider struct {
	BaseURL string
}

// NewLocalModelProvider 创建本地模型提供者
func NewLocalModelProvider(baseURL string) *LocalModelProvider {
	return &LocalModelProvider{BaseURL: baseURL}
}

func (p *LocalModelProvider) ProviderName() string { return "local" }

func (p *LocalModelProvider) Create(modelName string) (Model, error) {
	return &LocalModel{
		baseURL: p.BaseURL,
		model:   modelName,
	}, nil
}

// LocalModel 本地模型实现 (Ollama兼容)
type LocalModel struct {
	baseURL string
	model   string
}

func (m *LocalModel) Name() string { return m.model }

// ollamaRequest Ollama API请求
type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
	Options  *ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	NumPredict int     `json:"num_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// ollamaResponse Ollama API响应
type ollamaResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
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

	resp, err := http.DefaultClient.Do(httpReq)
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

	return &ChatResponse{
		Content: lastResp.Message.Content,
		Tokens:  lastResp.EvalCount + lastResp.PromptEvalCount,
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

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求本地模型失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("本地模型返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	chunks := make(chan StreamChunk, 64)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var r ollamaResponse
			if err := json.Unmarshal([]byte(line), &r); err != nil {
				continue
			}

			sc := StreamChunk{
				Delta: r.Message.Content,
				Done:  r.Done,
			}
			if r.Done {
				sc.Tokens = r.EvalCount + r.PromptEvalCount
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

func (m *LocalModel) convertRequest(req *ChatRequest) *ollamaRequest {
	ollamaReq := &ollamaRequest{
		Model:  m.model,
		Stream: req.Stream,
	}

	for _, msg := range req.Messages {
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if req.MaxTokens > 0 || req.Temperature > 0 {
		opts := &ollamaOptions{}
		if req.MaxTokens > 0 {
			opts.NumPredict = req.MaxTokens
		}
		if req.Temperature > 0 {
			opts.Temperature = req.Temperature
		}
		ollamaReq.Options = opts
	}

	return ollamaReq
}
