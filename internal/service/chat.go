package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cat-agent/internal/config"
	"cat-agent/internal/domain"
	"cat-agent/internal/model"
	"cat-agent/internal/repository"
	"cat-agent/internal/tool"

	"github.com/google/uuid"
)

// ChatService 对话服务 - Agent核心循环
type ChatService struct {
	repo               *repository.Repository
	openaiProvider     model.ModelProvider
	localProvider      model.ModelProvider
	toolRegistry       *tool.Registry
	cfg                *config.Config
	pendingApprovals   map[string]*PendingToolApproval
	pendingApprovalsMu sync.Mutex
	observability      *ObservabilityService
}

type PendingToolApproval struct {
	ID         string
	UserID     uint
	SessionID  uint
	ToolName   string
	ToolCallID string
	Arguments  string
}

// NewChatService 创建对话服务
func NewChatService(
	repo *repository.Repository,
	openaiProvider model.ModelProvider,
	localProvider model.ModelProvider,
	toolRegistry *tool.Registry,
	cfg *config.Config,
) *ChatService {
	return &ChatService{
		repo:             repo,
		openaiProvider:   openaiProvider,
		localProvider:    localProvider,
		toolRegistry:     toolRegistry,
		cfg:              cfg,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}
}

// SetObservabilityService 设置可观测性服务（可选）
func (s *ChatService) SetObservabilityService(obs *ObservabilityService) {
	s.observability = obs
}

// ChatInput 对话输入
type ChatInput struct {
	AgentID   uint   `json:"agent_id"`
	SessionID uint   `json:"session_id"`
	UserID    uint   `json:"user_id"`
	Content   string `json:"content"`
	VendorKey string `json:"vendor_key"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	ModelName string `json:"model_name"`
}

// ChatOutput 对话输出
type ChatOutput struct {
	SessionID uint                 `json:"session_id"`
	Events    []domain.StreamEvent `json:"events,omitempty"`
}

// ModelCallStats 模型调用统计
type ModelCallStats struct {
	LatencyMs    int64 // 单次模型调用延迟（毫秒）
	InputTokens  int   // 本次调用的输入tokens
	OutputTokens int   // 本次调用的输出tokens
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
	if eventCh != nil {
		eventCh <- domain.StreamEvent{Type: "session", Content: fmt.Sprintf("%d", session.ID)}
	}

	// ========== Step 3: 获取Agent配置 ==========
	agentCfg, err := s.repo.Agent.GetByID(input.AgentID)
	if err != nil {
		return fmt.Errorf("Agent配置不存在: %w", err)
	}

	// 创建执行轨迹（可观测性）
	var traceID string
	if s.observability != nil {
		if t, err := s.observability.CreateExecutionTrace(input.UserID, session.ID, input.AgentID); err == nil && t != nil {
			traceID = t.TraceID
			// 将 trace_id 注入 context 以便后续span可关联
			ctx = context.WithValue(ctx, "trace_id", traceID)
		}
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
	_, _, _, resolvedModelName, err := s.resolveModelConfig(agentCfg, input)
	if err != nil {
		return fmt.Errorf("获取模型失败: %w", err)
	}

	mdl, err := s.getModel(agentCfg, input)
	if err != nil {
		return fmt.Errorf("获取模型失败: %w", err)
	}

	// ========== Step 9: Agent循环 - 模型调用 + 工具执行 ==========
	maxIterations := 10 // 防止无限循环
	totalModelLatency := int64(0)
	totalInputTokens := 0

	for i := 0; i < maxIterations; i++ {
		req := &model.ChatRequest{
			Model:       resolvedModelName,
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   agentCfg.MaxTokens,
			Temperature: agentCfg.Temperature,
			Stream:      eventCh != nil,
		}

		var stats *ModelCallStats
		var err error
		if eventCh != nil {
			// 流式模式
			stats, err = s.handleStreamResponse(ctx, mdl, req, messages, session, agentCfg, eventCh)
		} else {
			// 同步模式
			stats, err = s.handleSyncResponse(ctx, mdl, req, messages, session, agentCfg, eventCh)
		}

		if err != nil {
			return err
		}

		// 累积模型调用统计
		if stats != nil {
			totalModelLatency += stats.LatencyMs
			totalInputTokens += stats.InputTokens
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

			// 记录工具调用span
			var span *domain.TraceSpan
			var toolStartTime time.Time
			if s.observability != nil && traceID != "" {
				toolStartTime = time.Now()
				sp, _ := s.observability.AddTraceSpan(traceID, "", "tool_call", tc.Function.Name, map[string]interface{}{"args": tc.Function.Arguments}, nil)
				span = sp
			}

			event, approvalID, err := s.executeTool(ctx, tc, session.ID)

			if span != nil && s.observability != nil {
				// 完成span并记录工具指标
				latencyMs := time.Since(toolStartTime).Milliseconds()
				tstatus := "success"
				if err != nil {
					tstatus = "failure"
				}
				_ = s.observability.FinishTraceSpan(span.SpanID, tstatus, "")
				_ = s.observability.RecordToolMetrics(tc.Function.Name, latencyMs, err == nil)
			}
			if err != nil {
				return err
			}

			if event.Type == "tool_confirmation" {
				if eventCh != nil {
					eventCh <- *event
				}
				if approvalID != "" {
					go s.updateMemory(input.UserID, session.ID, sanitizedContent)
					return nil
				}
				return nil
			}

			if eventCh != nil {
				eventCh <- *event
			}

			// 将工具结果添加到消息列表
			toolResultMsg := model.Message{
				Role:       "tool",
				Content:    event.Content,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// 保存工具结果消息
			s.repo.Message.Create(&domain.Message{
				SessionID:  session.ID,
				Role:       "tool",
				Content:    event.Content,
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

	// 完成执行轨迹（记录真实的延迟和token统计）
	if s.observability != nil && traceID != "" {
		stepCount := len(messages)
		toolCallCount := 0
		for _, m := range messages {
			if m.Role == "tool" {
				toolCallCount++
			}
		}
		_ = s.observability.FinishExecutionTrace(traceID, "success", stepCount, toolCallCount, totalInputTokens, session.TokenUsed, int(totalModelLatency))
	}

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
) (*ModelCallStats, error) {
	_ = agentCfg
	// 在可观测性中添加模型调用span
	var modelSpan *domain.TraceSpan
	var modelStartTime time.Time
	if s.observability != nil {
		modelStartTime = time.Now()
		if v := ctx.Value("trace_id"); v != nil {
			if tid, ok := v.(string); ok && tid != "" {
				sp, _ := s.observability.AddTraceSpan(tid, "", "model_call", req.Model, map[string]interface{}{"messages_len": len(req.Messages)}, nil)
				modelSpan = sp
			}
		}
	}

	chunks, err := mdl.StreamChat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("模型流式调用失败: %w", err)
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

	// 完成模型调用span并记录模型指标
	if modelSpan != nil && s.observability != nil {
		_ = s.observability.FinishTraceSpan(modelSpan.SpanID, "success", "")
		// 估算输入tokens（粗略估算：消息长度 / 4）
		inputTokens := 0
		for _, msg := range req.Messages {
			inputTokens += len(msg.Content) / 4
		}
		latencyMs := time.Since(modelStartTime).Milliseconds()
		_ = s.observability.RecordModelMetrics(req.Model, "unknown", latencyMs, inputTokens, totalTokens, 0.95)
	}

	return &ModelCallStats{
		LatencyMs:    time.Since(modelStartTime).Milliseconds(),
		InputTokens:  estimateInputTokens(req.Messages),
		OutputTokens: totalTokens,
	}, nil
}

// estimateInputTokens 估算输入tokens
func estimateInputTokens(messages []model.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content) / 4
	}
	return total
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
) (*ModelCallStats, error) {
	_ = agentCfg
	// 在可观测性中添加模型调用span
	var modelSpan *domain.TraceSpan
	var modelStartTime time.Time
	if s.observability != nil {
		modelStartTime = time.Now()
		if v := ctx.Value("trace_id"); v != nil {
			if tid, ok := v.(string); ok && tid != "" {
				sp, _ := s.observability.AddTraceSpan(tid, "", "model_call", req.Model, map[string]interface{}{"messages_len": len(req.Messages)}, nil)
				modelSpan = sp
			}
		}
	}

	resp, err := mdl.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("模型调用失败: %w", err)
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

	if modelSpan != nil && s.observability != nil {
		_ = s.observability.FinishTraceSpan(modelSpan.SpanID, "success", "")
		// 估算输入tokens（粗略估算：消息长度 / 4）
		inputTokens := estimateInputTokens(req.Messages)
		latencyMs := time.Since(modelStartTime).Milliseconds()
		_ = s.observability.RecordModelMetrics(req.Model, "unknown", latencyMs, inputTokens, resp.Tokens, 0.95)
	}

	return &ModelCallStats{
		LatencyMs:    time.Since(modelStartTime).Milliseconds(),
		InputTokens:  estimateInputTokens(req.Messages),
		OutputTokens: resp.Tokens,
	}, nil
}

// executeTool 执行工具调用
func (s *ChatService) executeTool(ctx context.Context, tc model.ToolCall, sessionID uint) (*domain.StreamEvent, string, error) {
	t, ok := s.toolRegistry.Get(tc.Function.Name)
	if !ok {
		return &domain.StreamEvent{
			Type:    "tool_result",
			Content: fmt.Sprintf("错误：工具 %s 不存在", tc.Function.Name),
			Tool:    tc.Function.Name,
		}, "", nil
	}

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return &domain.StreamEvent{
			Type:    "tool_result",
			Content: fmt.Sprintf("错误：解析工具参数失败: %v", err),
			Tool:    tc.Function.Name,
		}, "", nil
	}

	// 校验参数
	if err := tool.ValidateArgs(t, args); err != nil {
		return &domain.StreamEvent{
			Type:    "tool_result",
			Content: fmt.Sprintf("错误：参数校验失败: %v", err),
			Tool:    tc.Function.Name,
		}, "", nil
	}

	if confirmable, ok := t.(tool.ConfirmableTool); ok && confirmable.RequiresConfirmation(args) {
		approvalID := uuid.New().String()
		s.pendingApprovalsMu.Lock()
		s.pendingApprovals[approvalID] = &PendingToolApproval{
			ID:         approvalID,
			SessionID:  sessionID,
			ToolName:   tc.Function.Name,
			ToolCallID: tc.ID,
			Arguments:  tc.Function.Arguments,
		}
		s.pendingApprovalsMu.Unlock()

		return &domain.StreamEvent{
			Type:           "tool_confirmation",
			Content:        fmt.Sprintf("需要确认才会执行本地命令：%s", tc.Function.Name),
			Tool:           tc.Function.Name,
			Args:           tc.Function.Arguments,
			ConfirmationID: approvalID,
		}, approvalID, nil
	}

	return s.executeToolNow(ctx, tc.Function.Name, tc.Function.Arguments)
}

func (s *ChatService) executeToolNow(ctx context.Context, toolName string, argsJSON string) (*domain.StreamEvent, string, error) {
	t, ok := s.toolRegistry.Get(toolName)
	if !ok {
		return &domain.StreamEvent{Type: "tool_result", Content: fmt.Sprintf("错误：工具 %s 不存在", toolName), Tool: toolName}, "", nil
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return &domain.StreamEvent{Type: "tool_result", Content: fmt.Sprintf("错误：解析工具参数失败: %v", err), Tool: toolName}, "", nil
	}

	if err := tool.ValidateArgs(t, args); err != nil {
		return &domain.StreamEvent{Type: "tool_result", Content: fmt.Sprintf("错误：参数校验失败: %v", err), Tool: toolName}, "", nil
	}

	toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := t.Execute(toolCtx, args)
	if err != nil {
		return &domain.StreamEvent{Type: "tool_result", Content: fmt.Sprintf("工具执行异常: %v", err), Tool: toolName}, "", nil
	}

	return &domain.StreamEvent{Type: "tool_result", Content: result.Content, Tool: toolName}, "", nil
}

func (s *ChatService) ExecutePendingTool(approvalID string, approved bool, sessionID uint, userID uint) (*domain.StreamEvent, error) {
	s.pendingApprovalsMu.Lock()
	pending, ok := s.pendingApprovals[approvalID]
	if !ok {
		s.pendingApprovalsMu.Unlock()
		return nil, fmt.Errorf("确认请求已失效")
	}
	if pending.SessionID == 0 && sessionID > 0 {
		pending.SessionID = sessionID
	}
	delete(s.pendingApprovals, approvalID)
	s.pendingApprovalsMu.Unlock()

	if !approved {
		result := "已取消执行"
		_ = s.repo.Message.Create(&domain.Message{SessionID: pending.SessionID, Role: "tool", Content: result, ToolCallID: pending.ToolCallID})
		return &domain.StreamEvent{Type: "tool_result", Content: result, Tool: pending.ToolName}, nil
	}

	event, _, err := s.executeToolNow(context.Background(), pending.ToolName, pending.Arguments)
	if err != nil {
		return nil, err
	}
	if event != nil {
		_ = s.repo.Message.Create(&domain.Message{SessionID: pending.SessionID, Role: "tool", Content: event.Content, ToolCallID: pending.ToolCallID})
	}
	return event, nil
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
	_ = sessionID
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

// getModelDefaults 从数据库获取模型提供者的默认配置
// 所有提供者统一走此方法，无特殊对待
func (s *ChatService) getModelDefaults(provider string) (baseURL, apiKey, modelName string) {
	// 优先从数据库查询
	if s.repo != nil && s.repo.ModelConfig != nil {
		cfg, err := s.repo.ModelConfig.GetByProvider(provider)
		if err == nil && cfg != nil {
			return cfg.BaseURL, cfg.APIKey, cfg.DefaultModel
		}
	}

	// 数据库无配置时，使用硬编码降级默认值
	return model.DefaultProviderConfig(provider)
}

// resolveModelConfig 解析请求覆盖后的模型配置
func (s *ChatService) resolveModelConfig(agentCfg *domain.AgentConfig, input *ChatInput) (provider string, baseURL string, apiKey string, modelName string, err error) {
	if agentCfg == nil {
		return "", "", "", "", fmt.Errorf("Agent配置不能为空")
	}

	if agentCfg.UseGlobalModelConfig {
		provider = input.VendorKey
		baseURL = input.BaseURL
		apiKey = input.APIKey
		modelName = input.ModelName
		if provider == "" {
			provider = "openrouter"
		}
	} else {
		provider = agentCfg.ModelProvider
		modelName = agentCfg.ModelName
		if provider == "" || modelName == "" {
			return "", "", "", "", fmt.Errorf("Agent模型配置不完整，请补全模型提供者和模型名称")
		}
	}

	if provider == "" {
		return "", "", "", "", fmt.Errorf("模型提供者不能为空")
	}

	// 所有模型提供者统一从数据库获取默认配置（除custom外）
	if provider == "custom" {
		if baseURL == "" {
			return "", "", "", "", fmt.Errorf("自定义链接不能为空")
		}
		if modelName == "" {
			return "", "", "", "", fmt.Errorf("模型名称不能为空")
		}
		return provider, baseURL, apiKey, modelName, nil
	}

	// 统一从数据库获取默认配置
	if baseURL == "" || apiKey == "" || modelName == "" {
		defaultBase, defaultKey, defaultModel := s.getModelDefaults(provider)
		if baseURL == "" {
			baseURL = defaultBase
		}
		if apiKey == "" {
			apiKey = defaultKey
		}
		if modelName == "" {
			modelName = defaultModel
		}
	}

	return model.SDKProviderName(provider), baseURL, apiKey, modelName, nil
}

// getModel 根据配置获取模型实例
func (s *ChatService) getModel(agentCfg *domain.AgentConfig, input *ChatInput) (model.Model, error) {
	provider, baseURL, apiKey, modelName, err := s.resolveModelConfig(agentCfg, input)
	if err != nil {
		return nil, err
	}

	return model.NewLLMProvider(provider, baseURL, apiKey).Create(modelName)
}

// updateMemory 异步更新用户记忆
func (s *ChatService) updateMemory(userID, sessionID uint, content string) {
	// 简化实现：提取关键信息存入记忆
	// 实际应用中可以调用LLM进行摘要提取

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
