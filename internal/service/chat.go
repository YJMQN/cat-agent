package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"eino-agent/internal/config"
	"eino-agent/internal/domain"
	"eino-agent/internal/model"
	"eino-agent/internal/repository"
	"eino-agent/internal/tool"

	"github.com/google/uuid"
)

// ChatService 对话服务 - Agent核心循环
type ChatService struct {
	repo           *repository.Repository
	openaiProvider *model.OpenAIProvider
	localProvider  *model.LocalModelProvider
	toolRegistry   *tool.Registry
	cfg            *config.Config
}

// NewChatService 创建对话服务
func NewChatService(
	repo *repository.Repository,
	openaiProvider *model.OpenAIProvider,
	localProvider *model.LocalModelProvider,
	toolRegistry *tool.Registry,
	cfg *config.Config,
) *ChatService {
	return &ChatService{
		repo:           repo,
		openaiProvider: openaiProvider,
		localProvider:  localProvider,
		toolRegistry:   toolRegistry,
		cfg:            cfg,
	}
}

// ChatInput 对话输入
type ChatInput struct {
	AgentID   uint   `json:"agent_id"`
	SessionID uint   `json:"session_id"`
	UserID    uint   `json:"user_id"`
	Content   string `json:"content"`
}

// ChatOutput 对话输出
type ChatOutput struct {
	SessionID uint           `json:"session_id"`
	Events    []domain.StreamEvent `json:"events,omitempty"`
}

// HandleChat 处理对话请求 - Agent核心循环
// 流程: 接收消息 → 安全检查 → 记忆提取 → 构建上下文 → 模型调用 → 工具执行 → 生成回复
func (s *ChatService) HandleChat(ctx context.Context, input *ChatInput, eventCh chan<- domain.StreamEvent) error {
	// ========== Step 1: 输入安全检查 ==========
	sanitizedContent := tool.SanitizeInput(input.Content)
	if sanitizedContent == "" {
		return fmt.Errorf("消息内容不能为空")
	}

	// ========== Step 2: 获取或创建会话 ==========
	session, err := s.getOrCreateSession(input)
	if err != nil {
		return fmt.Errorf("会话处理失败: %w", err)
	}

	// ========== Step 3: 获取Agent配置 ==========
	agentCfg, err := s.repo.Agent.GetByID(input.AgentID)
	if err != nil {
		return fmt.Errorf("Agent配置不存在: %w", err)
	}

	// ========== Step 4: 保存用户消息 ==========
	userMsg := &domain.Message{
		SessionID: session.ID,
		Role:      "user",
		Content:   sanitizedContent,
	}
	if err := s.repo.Message.Create(userMsg); err != nil {
		return fmt.Errorf("保存消息失败: %w", err)
	}

	// ========== Step 5: 提取记忆 ==========
	memoryContext, err := s.buildMemoryContext(input.UserID, session.ID)
	if err != nil {
		log.Printf("提取记忆失败: %v", err)
		memoryContext = ""
	}

	// ========== Step 6: 构建对话历史 ==========
	messages, err := s.buildMessages(session.ID, agentCfg, memoryContext)
	if err != nil {
		return fmt.Errorf("构建对话上下文失败: %w", err)
	}

	// ========== Step 7: 获取工具定义 ==========
	toolNames := s.parseToolIDs(agentCfg.ToolIDs)
	toolDefs := s.buildToolDefs(toolNames)

	// ========== Step 8: 选择模型 ==========
	mdl, err := s.getModel(agentCfg)
	if err != nil {
		return fmt.Errorf("获取模型失败: %w", err)
	}

	// ========== Step 9: Agent循环 - 模型调用 + 工具执行 ==========
	maxIterations := 10 // 防止无限循环
	for i := 0; i < maxIterations; i++ {
		req := &model.ChatRequest{
			Model:       agentCfg.ModelName,
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   agentCfg.MaxTokens,
			Temperature: agentCfg.Temperature,
			Stream:      eventCh != nil,
		}

		if eventCh != nil {
			// 流式模式
			err = s.handleStreamResponse(ctx, mdl, req, messages, session, agentCfg, eventCh)
		} else {
			// 同步模式
			err = s.handleSyncResponse(ctx, mdl, req, messages, session, agentCfg, eventCh)
		}

		if err != nil {
			return err
		}

		// 检查最后一条消息是否包含工具调用
		lastMsg := messages[len(messages)-1]
		if lastMsg.Role != "assistant" || len(lastMsg.ToolCalls) == 0 {
			// 没有工具调用，对话结束
			break
		}

		// ========== 执行工具调用 ==========
		for _, tc := range lastMsg.ToolCalls {
			if eventCh != nil {
				eventCh <- domain.StreamEvent{
					Type:    "tool_call",
					Content: tc.Function.Name,
					Tool:    tc.Function.Name,
					Args:    tc.Function.Arguments,
				}
			}

			result := s.executeTool(ctx, tc)

			if eventCh != nil {
				eventCh <- domain.StreamEvent{
					Type:    "tool_result",
					Content: result.Content,
					Tool:    tc.Function.Name,
				}
			}

			// 将工具结果添加到消息列表
			toolResultMsg := model.Message{
				Role:       "tool",
				Content:    result.Content,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// 保存工具结果消息
			s.repo.Message.Create(&domain.Message{
				SessionID: session.ID,
				Role:      "tool",
				Content:   result.Content,
				ToolCallID: tc.ID,
			})
		}
	}

	// ========== Step 10: 发送完成事件 ==========
	if eventCh != nil {
		eventCh <- domain.StreamEvent{Type: "done"}
	}

	// ========== Step 11: 异步更新记忆 ==========
	go s.updateMemory(input.UserID, session.ID, sanitizedContent)

	return nil
}

// handleStreamResponse 处理流式响应
func (s *ChatService) handleStreamResponse(
	ctx context.Context,
	mdl model.Model,
	req *model.ChatRequest,
	messages []model.Message,
	session *domain.Session,
	agentCfg *domain.AgentConfig,
	eventCh chan<- domain.StreamEvent,
) error {
	chunks, err := mdl.StreamChat(ctx, req)
	if err != nil {
		return fmt.Errorf("模型流式调用失败: %w", err)
	}

	var fullContent string
	var toolCalls []model.ToolCall
	totalTokens := 0

	for chunk := range chunks {
		if chunk.Done {
			totalTokens = chunk.Tokens
			break
		}

		fullContent += chunk.Delta

		// 收集工具调用
		if len(chunk.ToolCalls) > 0 {
			for _, tc := range chunk.ToolCalls {
				// 合并工具调用参数
				found := false
				for i, existing := range toolCalls {
					if existing.ID == tc.ID || (existing.Function.Name == "" && tc.ID == "") {
						toolCalls[i].Function.Name += tc.Function.Name
						toolCalls[i].Function.Arguments += tc.Function.Arguments
						if tc.ID != "" {
							toolCalls[i].ID = tc.ID
						}
						if tc.Type != "" {
							toolCalls[i].Type = tc.Type
						}
						found = true
						break
					}
				}
				if !found {
					toolCalls = append(toolCalls, tc)
				}
			}
		}

		// 发送文本增量
		if chunk.Delta != "" {
			eventCh <- domain.StreamEvent{
				Type:    "text",
				Content: chunk.Delta,
			}
		}
	}

	// 构建助手消息
	assistantMsg := model.Message{
		Role:      "assistant",
		Content:   fullContent,
		ToolCalls: toolCalls,
	}
	messages = append(messages, assistantMsg)

	// 保存助手消息
	msg := &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   fullContent,
		Tokens:    totalTokens,
	}
	if len(toolCalls) > 0 {
		tcJSON, _ := json.Marshal(toolCalls)
		msg.ToolCalls = string(tcJSON)
	}
	s.repo.Message.Create(msg)

	// 更新会话token用量
	session.TokenUsed += totalTokens
	s.repo.Session.Update(session)

	return nil
}

// handleSyncResponse 处理同步响应
func (s *ChatService) handleSyncResponse(
	ctx context.Context,
	mdl model.Model,
	req *model.ChatRequest,
	messages []model.Message,
	session *domain.Session,
	agentCfg *domain.AgentConfig,
	eventCh chan<- domain.StreamEvent,
) error {
	resp, err := mdl.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("模型调用失败: %w", err)
	}

	assistantMsg := model.Message{
		Role:      "assistant",
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
	}
	messages = append(messages, assistantMsg)

	// 保存助手消息
	msg := &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   resp.Content,
		Tokens:    resp.Tokens,
	}
	if len(resp.ToolCalls) > 0 {
		tcJSON, _ := json.Marshal(resp.ToolCalls)
		msg.ToolCalls = string(tcJSON)
	}
	s.repo.Message.Create(msg)

	session.TokenUsed += resp.Tokens
	s.repo.Session.Update(session)

	if eventCh != nil {
		eventCh <- domain.StreamEvent{
			Type:    "text",
			Content: resp.Content,
		}
	}

	return nil
}

// executeTool 执行工具调用
func (s *ChatService) executeTool(ctx context.Context, tc model.ToolCall) *domain.StreamEvent {
	t, ok := s.toolRegistry.Get(tc.Function.Name)
	if !ok {
		return &domain.StreamEvent{
			Type:    "tool_result",
			Content: fmt.Sprintf("错误：工具 %s 不存在", tc.Function.Name),
			Tool:    tc.Function.Name,
		}
	}

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return &domain.StreamEvent{
			Type:    "tool_result",
			Content: fmt.Sprintf("错误：解析工具参数失败: %v", err),
			Tool:    tc.Function.Name,
		}
	}

	// 校验参数
	if err := tool.ValidateArgs(t, args); err != nil {
		return &domain.StreamEvent{
			Type:    "tool_result",
			Content: fmt.Sprintf("错误：参数校验失败: %v", err),
			Tool:    tc.Function.Name,
		}
	}

	// 执行工具（带超时）
	toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := t.Execute(toolCtx, args)
	if err != nil {
		return &domain.StreamEvent{
			Type:    "tool_result",
			Content: fmt.Sprintf("工具执行异常: %v", err),
			Tool:    tc.Function.Name,
		}
	}

	return &domain.StreamEvent{
		Type:    "tool_result",
		Content: result.Content,
		Tool:    tc.Function.Name,
	}
}

// getOrCreateSession 获取或创建会话
func (s *ChatService) getOrCreateSession(input *ChatInput) (*domain.Session, error) {
	if input.SessionID > 0 {
		session, err := s.repo.Session.GetByID(input.SessionID)
		if err == nil {
			return session, nil
		}
	}

	// 创建新会话
	session := &domain.Session{
		UUID:    uuid.New().String(),
		AgentID: input.AgentID,
		UserID:  input.UserID,
		Title:   input.Content,
		Status:  "active",
	}
	if len(session.Title) > 100 {
		session.Title = session.Title[:100] + "..."
	}

	if err := s.repo.Session.Create(session); err != nil {
		return nil, err
	}
	return session, nil
}

// buildMemoryContext 构建记忆上下文
func (s *ChatService) buildMemoryContext(userID, sessionID uint) (string, error) {
	memories, err := s.repo.Memory.GetByUser(userID)
	if err != nil {
		return "", err
	}

	if len(memories) == 0 {
		return "", nil
	}

	var ctx string
	ctx += "## 用户长期记忆\n"
	for _, m := range memories {
		ctx += fmt.Sprintf("- [%s] %s: %s\n", m.Category, m.Key, m.Content)
	}

	return ctx, nil
}

// buildMessages 构建完整消息列表
func (s *ChatService) buildMessages(
	sessionID uint,
	agentCfg *domain.AgentConfig,
	memoryContext string,
) ([]model.Message, error) {
	var messages []model.Message

	// 系统提示词
	systemPrompt := "你是一个有用的AI助手。"
	if agentCfg.SystemPrompt != "" {
		systemPrompt = agentCfg.SystemPrompt
	}

	// 注入记忆
	if memoryContext != "" {
		systemPrompt += "\n\n" + memoryContext
	}

	messages = append(messages, model.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// 加载历史消息 (最近20条)
	history, err := s.repo.Message.GetBySession(sessionID, 20)
	if err != nil {
		return messages, nil
	}

	for _, msg := range history {
		m := model.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		if msg.ToolCalls != "" {
			var tc []model.ToolCall
			json.Unmarshal([]byte(msg.ToolCalls), &tc) // nolint
			m.ToolCalls = tc
		}
		messages = append(messages, m)
	}

	return messages, nil
}

// buildToolDefs 构建工具定义
func (s *ChatService) buildToolDefs(names []string) []model.ToolDef {
	var defs []model.ToolDef
	for _, name := range names {
		t, ok := s.toolRegistry.Get(name)
		if !ok {
			continue
		}
		defs = append(defs, model.ToolDef{
			Type: "function",
			Function: model.ToolDefFunction{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}

// parseToolIDs 解析工具ID列表 (JSON数组字符串)
func (s *ChatService) parseToolIDs(toolIDsJSON string) []string {
	if toolIDsJSON == "" {
		// 默认注册所有内置工具
		var names []string
		for _, t := range s.toolRegistry.List() {
			names = append(names, t.Name())
		}
		return names
	}

	var ids []string
	json.Unmarshal([]byte(toolIDsJSON), &ids) // nolint
	return ids
}

// getModel 根据配置获取模型实例
func (s *ChatService) getModel(agentCfg *domain.AgentConfig) (model.Model, error) {
	switch agentCfg.ModelProvider {
	case "openai":
		return s.openaiProvider.Create(agentCfg.ModelName)
	case "local":
		return s.localProvider.Create(agentCfg.ModelName)
	default:
		return nil, fmt.Errorf("不支持的模型提供者: %s", agentCfg.ModelProvider)
	}
}

// updateMemory 异步更新用户记忆
func (s *ChatService) updateMemory(userID, sessionID uint, content string) {
	// 简化实现：提取关键信息存入记忆
	// 实际应用中可以调用LLM进行摘要提取
	ctx := context.Background()
	_ = ctx

	// 示例：记录会话摘要
	_ = s.repo.Memory.Create(&domain.Memory{
		UserID:    userID,
		SessionID: sessionID,
		Category:  "summary",
		Key:       fmt.Sprintf("session_%d_last_msg", sessionID),
		Content:   content,
		Source:    "auto",
	})
}
