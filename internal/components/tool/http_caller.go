package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// HTTPCallerTool HTTP 调用工具 - 用于调用外部 API
type HTTPCallerTool struct {
	*BaseTool
	client *http.Client
}

// HTTPCallerArgs HTTP 调用参数
type HTTPCallerArgs struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

// NewHTTPCallerTool 创建 HTTP 调用工具
func NewHTTPCallerTool() *HTTPCallerTool {
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"method": map[string]interface{}{
				"type":        "string",
				"description": "HTTP 方法 (GET, POST, PUT, DELETE 等)",
				"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "请求 URL",
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "请求头",
			},
			"body": map[string]interface{}{
				"type":        "object",
				"description": "请求体 (JSON 格式)",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "超时时间（秒）",
			},
		},
		"required": []string{"method", "url"},
	}

	tool := &HTTPCallerTool{
		BaseTool: NewBaseTool("http_caller", "调用外部 HTTP API", params),
		client:   &http.Client{},
	}
	tool.SetRequired("method", "url")

	return tool
}

// Execute 执行 HTTP 调用
func (t *HTTPCallerTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var reqArgs HTTPCallerArgs
	if err := json.Unmarshal(args, &reqArgs); err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Invalid arguments: %v", err),
			IsError: true,
		}, nil
	}

	parsedURL, err := url.Parse(reqArgs.URL)
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Invalid URL: %v", err),
			IsError: true,
		}, nil
	}

	if isInternalURL(parsedURL) {
		return &ToolResult{
			Content: "Access to internal network addresses is forbidden",
			IsError: true,
		}, nil
	}

	timeout := time.Duration(30) * time.Second
	if reqArgs.Timeout > 0 {
		timeout = time.Duration(reqArgs.Timeout) * time.Second
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctxWithTimeout, reqArgs.Method, reqArgs.URL, nil)
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Failed to create request: %v", err),
			IsError: true,
		}, nil
	}

	for k, v := range reqArgs.Headers {
		httpReq.Header.Set(k, v)
	}

	if len(reqArgs.Body) > 0 {
		httpReq.Body = io.NopCloser(bytes.NewReader(reqArgs.Body))
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Request failed: %v", err),
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Failed to read response: %v", err),
			IsError: true,
		}, nil
	}

	result := fmt.Sprintf("Status: %d\nBody: %s", resp.StatusCode, string(respBody))

	return &ToolResult{
		Content: result,
		Data: map[string]interface{}{
			"status": resp.StatusCode,
			"body":   string(respBody),
		},
	}, nil
}

// ValidatePermission 权限校验
func (t *HTTPCallerTool) ValidatePermission(userID string, args json.RawMessage) error {
	return nil
}

func isInternalURL(u *url.URL) bool {
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	return false
}
