package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-agent-system/internal/components/llm"
	"ai-agent-system/internal/components/memory"
	"ai-agent-system/internal/components/tool"
	"ai-agent-system/internal/security"
	"ai-agent-system/pkg/types"
)

// Config Agent 配置
type Config struct {
	LLMAdapter      llm.Adapter
	ToolRegistry    *tool.Registry
	MemoryManager   *memory.MemoryManager
	SecurityGuard   security.Guard
	SystemPrompt    string
	MaxIterations   int // 最大 ReAct 迭代次数
	RetryCount      int // 工具调用失败重试次数
}

// Agent AI Agent 主类
type Agent struct {
	config        *Config
	sessionSystem types.Message
}

// NewAgent 创建 Agent 实例
func NewAgent(config *Config) (*Agent, error) {
	if config.LLMAdapter == nil {
		return nil, fmt.Errorf("LLM adapter is required")
	}
	if config.ToolRegistry == nil {
		return nil, fmt.Errorf("Tool registry is required")
	}
	if config.MemoryManager == nil {
		return nil, fmt.Errorf("Memory manager is required")
	}
	if config.SecurityGuard == nil {
		return nil, fmt.Errorf("Security guard is required")
	}

	maxIterations := config.MaxIterations
	if maxIterations == 0 {
		maxIterations = 10
	}

	retryCount := config.RetryCount
	if retryCount == 0 {
		retryCount = 3
	}

	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = getDefaultSystemPrompt()
	}

	return &Agent{
		config: config,
		sessionSystem: types.Message{
			Role:      types.RoleSystem,
			Content:   systemPrompt,
			Timestamp: time.Now(),
		},
	}, nil
}

// Run 运行 Agent 主循环（非流式）
func (a *Agent) Run(ctx context.Context, sessionID, userID, message string) (*types.AgentResponse, error) {
	// 速率限制检查
	if err := a.config.SecurityGuard.CheckRateLimit(ctx, userID); err != nil {
		return &types.AgentResponse{
			SessionID:    sessionID,
			FinishReason: "rate_limit",
			Error: &types.ErrorResponse{
				Code:    "RATE_LIMITED",
				Message: err.Error(),
			},
		}, nil
	}

	// 添加用户消息到会话
	userMsg := types.Message{
		Role:      types.RoleUser,
		Content:   message,
		Timestamp: time.Now(),
	}
	if err := a.config.MemoryManager.AddMessage(ctx, sessionID, userMsg); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 获取会话历史
	session, err := a.config.MemoryManager.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// 构建消息历史（包含 system prompt）
	messages := []types.Message{a.sessionSystem}
	messages = append(messages, session.Messages...)

	// ReAct 主循环
	var toolCalls []types.ToolCall
	iterations := 0
	maxIterations := a.config.MaxIterations
	if maxIterations == 0 {
		maxIterations = 10
	}

	for iterations < maxIterations {
		iterations++

		// 调用 LLM
		llmReq := &llm.ChatRequest{
			Messages: messages,
			Tools:    a.convertToolsToLLMFormat(),
			Stream:   false,
		}

		resp, err := a.config.LLMAdapter.Chat(ctx, llmReq)
		if err != nil {
			return &types.AgentResponse{
				SessionID:    sessionID,
				FinishReason: "error",
				Error: &types.ErrorResponse{
					Code:    "LLM_ERROR",
					Message: fmt.Sprintf("LLM call failed: %v", err),
				},
			}, nil
		}

		assistantMsg := resp.Message
		assistantMsg.Timestamp = time.Now()

		// 保存助手消息
		if err := a.config.MemoryManager.AddMessage(ctx, sessionID, assistantMsg); err != nil {
			return nil, fmt.Errorf("failed to save assistant message: %w", err)
		}
		messages = append(messages, assistantMsg)

		// 检查是否需要工具调用
		if len(assistantMsg.ToolCalls) == 0 {
			// 没有工具调用，返回最终响应
			return &types.AgentResponse{
				SessionID:    sessionID,
				Response:     assistantMsg.Content,
				ToolCalls:    toolCalls,
				Iterations:   iterations,
				FinishReason: resp.FinishReason,
			}, nil
		}

		// 执行工具调用
		toolCalls = append(toolCalls, assistantMsg.ToolCalls...)
		toolResults, shouldStop := a.executeToolCalls(ctx, sessionID, userID, assistantMsg.ToolCalls)
		
		// 添加工具结果到消息历史
		for _, result := range toolResults {
			toolMsg := types.Message{
				Role:       types.RoleTool,
				Content:    result.Content,
				ToolCallID: result.ToolCallID,
				Timestamp:  time.Now(),
			}
			messages = append(messages, toolMsg)
			if err := a.config.MemoryManager.AddMessage(ctx, sessionID, toolMsg); err != nil {
				return nil, fmt.Errorf("failed to save tool result: %w", err)
			}
		}

		if shouldStop {
			// 工具调用失败且无法恢复
			return &types.AgentResponse{
				SessionID:    sessionID,
				Response:     "抱歉，工具调用过程中出现错误，无法继续执行。",
				ToolCalls:    toolCalls,
				Iterations:   iterations,
				FinishReason: "tool_error",
			}, nil
		}
	}

	// 达到最大迭代次数
	return &types.AgentResponse{
		SessionID:    sessionID,
		Response:     "抱歉，已达到最大迭代次数，无法完成此任务。",
		ToolCalls:    toolCalls,
		Iterations:   iterations,
		FinishReason: "max_iterations",
	}, nil
}

// RunStream 运行 Agent 主循环（流式输出）
func (a *Agent) RunStream(ctx context.Context, sessionID, userID, message string) (<-chan types.StreamChunk, error) {
	outChan := make(chan types.StreamChunk, 64)

	go func() {
		defer close(outChan)

		// 速率限制检查
		if err := a.config.SecurityGuard.CheckRateLimit(ctx, userID); err != nil {
			outChan <- types.StreamChunk{
				Content:    "",
				IsFinished: true,
			}
			return
		}

		// 添加用户消息
		userMsg := types.Message{
			Role:      types.RoleUser,
			Content:   message,
			Timestamp: time.Now(),
		}
		if err := a.config.MemoryManager.AddMessage(ctx, sessionID, userMsg); err != nil {
			outChan <- types.StreamChunk{
				Content:    fmt.Sprintf("Error: %v", err),
				IsFinished: true,
			}
			return
		}

		// 获取会话历史
		session, err := a.config.MemoryManager.GetSession(ctx, sessionID)
		if err != nil {
			outChan <- types.StreamChunk{
				Content:    fmt.Sprintf("Error: %v", err),
				IsFinished: true,
			}
			return
		}

		messages := []types.Message{a.sessionSystem}
		messages = append(messages, session.Messages...)

		// 流式调用 LLM
		llmReq := &llm.ChatRequest{
			Messages: messages,
			Tools:    a.convertToolsToLLMFormat(),
			Stream:   true,
		}

		streamChan, err := a.config.LLMAdapter.ChatStream(ctx, llmReq)
		if err != nil {
			outChan <- types.StreamChunk{
				Content:    fmt.Sprintf("Error: %v", err),
				IsFinished: true,
			}
			return
		}

		var content string
		var toolCalls []types.ToolCall

		// 读取流式响应
		for event := range streamChan {
			if event.Error != nil {
				outChan <- types.StreamChunk{
					Content:    fmt.Sprintf("Error: %v", event.Error),
					IsFinished: true,
				}
				return
			}

			if event.Chunk == nil {
				continue
			}

			// 转发内容块
			if event.Chunk.Content != "" {
				content += event.Chunk.Content
				outChan <- types.StreamChunk{
					Content: event.Chunk.Content,
				}
			}

			// 收集工具调用
			if event.Chunk.ToolCall != nil {
				toolCalls = append(toolCalls, *event.Chunk.ToolCall)
			}

			if event.Chunk.IsFinished {
				break
			}
		}

		// 如果有工具调用，执行它们并继续
		if len(toolCalls) > 0 {
			// 保存助手消息
			assistantMsg := types.Message{
				Role:      types.RoleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
				Timestamp: time.Now(),
			}
			if err := a.config.MemoryManager.AddMessage(ctx, sessionID, assistantMsg); err != nil {
				return
			}
			messages = append(messages, assistantMsg)

			// 执行工具
			toolResults, _ := a.executeToolCalls(ctx, sessionID, userID, toolCalls)

			// 添加工具结果
			for _, result := range toolResults {
				toolMsg := types.Message{
					Role:       types.RoleTool,
					Content:    result.Content,
					ToolCallID: result.ToolCallID,
					Timestamp:  time.Now(),
				}
				messages = append(messages, toolMsg)
				if err := a.config.MemoryManager.AddMessage(ctx, sessionID, toolMsg); err != nil {
					return
				}
			}

			// 递归调用获取最终响应（简化实现，实际应继续循环）
			finalReq := &llm.ChatRequest{
				Messages: messages,
				Stream:   true,
			}

			finalStream, err := a.config.LLMAdapter.ChatStream(ctx, finalReq)
			if err != nil {
				return
			}

			for event := range finalStream {
				if event.Error != nil {
					return
				}
				if event.Chunk != nil {
					outChan <- *event.Chunk
				}
			}
		}

		outChan <- types.StreamChunk{
			IsFinished: true,
		}
	}()

	return outChan, nil
}

// executeToolCalls 执行工具调用
func (a *Agent) executeToolCalls(ctx context.Context, sessionID, userID string, toolCalls []types.ToolCall) ([]*types.ToolResult, bool) {
	results := make([]*types.ToolResult, 0, len(toolCalls))
	hasError := false

	for _, tc := range toolCalls {
		args, _ := json.Marshal(tc.Function.Arguments)

		// 安全校验
		if err := a.config.SecurityGuard.ValidateToolCall(ctx, userID, tc.Function.Name, args); err != nil {
			results = append(results, &types.ToolResult{
				ToolCallID: tc.ID,
				Content:    fmt.Sprintf("Security check failed: %v", err),
				IsError:    true,
			})
			hasError = true
			continue
		}

		// 执行工具（带重试）
		var result *types.ToolResult
		var execErr error

		for attempt := 0; attempt < a.config.RetryCount; attempt++ {
			result, execErr = a.config.ToolRegistry.Execute(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments), userID)
			if execErr == nil && !result.IsError {
				break
			}
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}

		if execErr != nil {
			result = &types.ToolResult{
				ToolCallID: tc.ID,
				Content:    fmt.Sprintf("Execution failed: %v", execErr),
				IsError:    true,
			}
			hasError = true
		}

		result.ToolCallID = tc.ID
		results = append(results, result)
	}

	return results, hasError
}

// convertToolsToLLMFormat 转换工具为 LLM 格式
func (a *Agent) convertToolsToLLMFormat() []llm.ToolDefinition {
	tools := a.config.ToolRegistry.List()
	llmTools := make([]llm.ToolDefinition, len(tools))

	for i, t := range tools {
		llmTools[i] = llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}

	return llmTools
}

func getDefaultSystemPrompt() string {
	return `你是一个智能助手，能够使用各种工具帮助用户完成任务。

你可以使用的工具包括：
- http_caller: 调用外部 HTTP API
- calculator: 执行数学计算

请遵循以下规则：
1. 仔细分析用户需求，确定是否需要使用工具
2. 如果需要多个步骤，按顺序执行
3. 如果工具调用失败，尝试重新执行或告知用户
4. 始终保持友好和专业的态度`
}
