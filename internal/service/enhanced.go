package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"eino-agent/internal/config"
	"eino-agent/internal/domain"
	"eino-agent/internal/repository"

	"github.com/google/uuid"
)

// ========== 第三阶段：智能记忆系统 ==========

// MemoryService 智能记忆服务
type MemoryService struct {
	repo *repository.Repository
	cfg  *config.Config
	mu   sync.RWMutex
	// 工作记忆（当前会话）
	workingMemory map[uint]*WorkingMemory
}

// WorkingMemory 工作记忆
type WorkingMemory struct {
	SessionID  uint
	Messages   []string
	Context    map[string]interface{}
	LastAccess time.Time
}

// NewMemoryService 创建记忆服务
func NewMemoryService(repo *repository.Repository, cfg *config.Config) *MemoryService {
	return &MemoryService{
		repo:          repo,
		cfg:           cfg,
		workingMemory: make(map[uint]*WorkingMemory),
	}
}

// ========== 分层记忆存储 ==========

// StoreWorkingMemory 存储工作记忆
func (s *MemoryService) StoreWorkingMemory(sessionID uint, content string, context map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wm, ok := s.workingMemory[sessionID]
	if !ok {
		wm = &WorkingMemory{
			SessionID: sessionID,
			Messages:  make([]string, 0),
			Context:   make(map[string]interface{}),
		}
		s.workingMemory[sessionID] = wm
	}

	// 保持工作记忆最多20条
	wm.Messages = append(wm.Messages, content)
	if len(wm.Messages) > 20 {
		wm.Messages = wm.Messages[len(wm.Messages)-20:]
	}

	if context != nil {
		for k, v := range context {
			wm.Context[k] = v
		}
	}
	wm.LastAccess = time.Now()
}

// GetWorkingMemory 获取工作记忆
func (s *MemoryService) GetWorkingMemory(sessionID uint) *WorkingMemory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.workingMemory[sessionID]
}

// ConsolidateMemory 工作记忆向短期记忆合并
func (s *MemoryService) ConsolidateMemory(ctx context.Context, userID, sessionID uint) error {
	wm := s.GetWorkingMemory(sessionID)
	if wm == nil || len(wm.Messages) == 0 {
		return nil
	}

	// 合并成摘要
	summary := s.summarizeMessages(wm.Messages)

	memory := &domain.MemoryItem{
		UserID:    userID,
		SessionID: sessionID,
		Level:     "short",
		Type:      "summary",
		Content:   summary,
		Importance: 0.7,
		CreatedAt: time.Now(),
	}

	if err := s.repo.MemoryItem.Create(memory); err != nil {
		return fmt.Errorf("保存短期记忆失败: %w", err)
	}

	// 清空工作记忆
	s.mu.Lock()
	delete(s.workingMemory, sessionID)
	s.mu.Unlock()

	return nil
}

// summarizeMessages 简单摘要合并
func (s *MemoryService) summarizeMessages(messages []string) string {
	if len(messages) == 0 {
		return ""
	}
	if len(messages) == 1 {
		return messages[0]
	}

	// 取首尾和中间重要部分
	var parts []string
	parts = append(parts, messages[0])
	if len(messages) > 3 {
		parts = append(parts, "...")
		parts = append(parts, messages[len(messages)/2])
		parts = append(parts, "...")
	}
	parts = append(parts, messages[len(messages)-1])

	return strings.Join(parts, "\n")
}

// ========== 语义检索 ==========

// SemanticSearch 语义检索记忆
func (s *MemoryService) SemanticSearch(ctx context.Context, userID uint, query string, limit int) ([]domain.MemoryItem, error) {
	if limit <= 0 {
		limit = 5
	}

	// 优先关键词匹配
	memories, err := s.repo.MemoryItem.Search(userID, query, limit)
	if err != nil {
		return nil, err
	}

	return memories, nil
}

// StoreMemory 存储记忆
func (s *MemoryService) StoreMemory(ctx context.Context, userID uint, memType, content string, importance float64) error {
	// 抽取关键词
	keywords := s.extractKeywords(content)

	memory := &domain.MemoryItem{
		UserID:     userID,
		Level:      "long",
		Type:       memType,
		Content:    content,
		Keywords:   keywords,
		Importance: importance,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour), // 30天过期
		CreatedAt:  time.Now(),
	}

	return s.repo.MemoryItem.Create(memory)
}

// extractKeywords 简单关键词抽取
func (s *MemoryService) extractKeywords(content string) string {
	// 去除常见词
	stopWords := map[string]bool{"的": true, "了": true, "是": true, "在": true, "有": true, "和": true, "就": true, "不": true, "人": true, "都": true, "一": true, "个": true, "上": true, "也": true, "很": true, "到": true, "说": true, "要": true, "去": true, "你": true, "会": true, "着": true, "没有": true, "看": true, "好": true, "自己": true, "这": true}
	
	words := strings.Fields(content)
	freq := make(map[string]int)
	for _, w := range words {
		if !stopWords[w] && len(w) > 1 {
			freq[w]++
		}
	}

	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].v > sorted[i].v {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	top := 10
	if len(sorted) < top {
		top = len(sorted)
	}

	var keywords []string
	for i := 0; i < top; i++ {
		keywords = append(keywords, sorted[i].k)
	}

	return strings.Join(keywords, ",")
}

// ========== 第三阶段：Cron调度器 ==========

// CronScheduler Cron调度服务
type CronScheduler struct {
	repo       *repository.Repository
	chat       *ChatService
	cfg        *config.Config
	jobs       map[uint]context.CancelFunc
	mu         sync.Mutex
	running    bool
}

// NewCronScheduler 创建Cron调度器
func NewCronScheduler(repo *repository.Repository, chat *ChatService, cfg *config.Config) *CronScheduler {
	return &CronScheduler{
		repo: repo,
		chat: chat,
		cfg:  cfg,
		jobs: make(map[uint]context.CancelFunc),
	}
}

// Start 启动调度器
func (s *CronScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.loop(ctx)
}

// Stop 停止调度器
func (s *CronScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, cancel := range s.jobs {
		cancel()
		delete(s.jobs, id)
	}
	s.running = false
}

// loop 主循环
func (s *CronScheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAndRun(ctx)
		}
	}
}

// checkAndRun 检查并执行到期的任务
func (s *CronScheduler) checkAndRun(ctx context.Context) {
	jobs, err := s.repo.CronJob.ListActive()
	if err != nil {
		return
	}

	now := time.Now()
	for _, job := range jobs {
		if job.NextRunAt.IsZero() || now.After(job.NextRunAt) || now.Equal(job.NextRunAt) {
			s.executeJob(ctx, &job)
		}
	}
}

// executeJob 执行任务
func (s *CronScheduler) executeJob(ctx context.Context, job *domain.CronJob) {
	log := &domain.CronLog{
		JobID:  job.ID,
		Status: "running",
		RunAt:  time.Now(),
	}
	s.repo.CronLog.Create(log)

	start := time.Now()

	// 构造对话输入
	chatInput := &ChatInput{
		AgentID: job.AgentID,
		Content: job.Prompt,
		UserID:  job.UserID,
	}

	eventCh := make(chan domain.StreamEvent, 100)
	err := s.chat.HandleChat(ctx, chatInput, eventCh)
	if err != nil {
		log.Status = "failed"
		log.Error = err.Error()
		log.Duration = int(time.Since(start).Milliseconds())
		s.repo.CronLog.Update(log)
		return
	}

	var result strings.Builder
	for event := range eventCh {
		if event.Type == "content" {
			result.WriteString(event.Content)
		}
	}

	// 更新任务状态
	job.LastRunAt = time.Now()
	job.RunCount++
	job.NextRunAt = s.calculateNextRun(job.CronExpr, job.LastRunAt)
	s.repo.CronJob.Update(job)

	log.Status = "success"
	log.Result = result.String()
	log.Duration = int(time.Since(start).Milliseconds())
	s.repo.CronLog.Update(log)
}

// calculateNextRun 计算下次运行时间（简化版，支持每N分钟/小时/天）
func (s *CronScheduler) calculateNextRun(cronExpr string, last time.Time) time.Time {
	parts := strings.Fields(cronExpr)
	if len(parts) < 5 {
		return last.Add(24 * time.Hour)
	}

	// 简单解析每N分钟/小时
	if parts[0] == "*/5" || parts[0] == "*/10" || parts[0] == "*/15" || parts[0] == "*/30" {
		minute := 0
		fmt.Sscanf(parts[0], "*/%d", &minute)
		if minute > 0 {
			return last.Add(time.Duration(minute) * time.Minute)
		}
	}
	if parts[0] == "*" && parts[1] == "*" {
		return last.Add(24 * time.Hour)
	}

	return last.Add(24 * time.Hour)
}

// ========== 第四阶段：Token预算监控 ==========

// TokenBudgetService Token预算服务
type TokenBudgetService struct {
	repo *repository.Repository
	mu   sync.Mutex
}

// NewTokenBudgetService 创建预算服务
func NewTokenBudgetService(repo *repository.Repository) *TokenBudgetService {
	return &TokenBudgetService{repo: repo}
}

// CheckAndRecord 检查并记录Token消耗
func (s *TokenBudgetService) CheckAndRecord(userID uint, tokens int64) (bool, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	budget, err := s.repo.TokenBudget.GetByUser(userID)
	if err != nil {
		// 创建默认预算
		budget = &domain.TokenBudget{
			UserID:        userID,
			DailyLimit:    1000000,
			MonthlyLimit:  30000000,
			AlertThreshold: 0.8,
			LastResetDate: time.Now().Format("2006-01-02"),
		}
		s.repo.TokenBudget.Create(budget)
	}

	// 重置每日计数
	today := time.Now().Format("2006-01-02")
	if budget.LastResetDate != today {
		budget.DailyUsed = 0
		budget.LastResetDate = today
	}

	budget.DailyUsed += tokens
	budget.MonthlyUsed += tokens
	s.repo.TokenBudget.Update(budget)

	// 检查阈值
	threshold := float64(budget.DailyUsed) / float64(budget.DailyLimit)
	if threshold >= 1.0 {
		return false, fmt.Sprintf("Token用量已超每日限额 %d/%d", budget.DailyUsed, budget.DailyLimit), nil
	}
	if threshold >= budget.AlertThreshold {
		return true, fmt.Sprintf("Token用量已达 %.0f%% (%d/%d)", threshold*100, budget.DailyUsed, budget.DailyLimit), nil
	}

	return true, "", nil
}

// ========== 第四阶段：对话导出 ==========

// ExportService 对话导出服务
type ExportService struct {
	repo *repository.Repository
	cfg  *config.Config
}

// NewExportService 创建导出服务
func NewExportService(repo *repository.Repository, cfg *config.Config) *ExportService {
	return &ExportService{repo: repo, cfg: cfg}
}

// ExportSession 导出对话
func (s *ExportService) ExportSession(ctx context.Context, sessionID, userID uint, format string) (string, error) {
	session, err := s.repo.Session.GetByID(sessionID)
	if err != nil {
		return "", fmt.Errorf("会话不存在: %w", err)
	}

	messages, err := s.repo.Message.GetBySession(sessionID, 1000)
	if err != nil {
		return "", fmt.Errorf("获取消息失败: %w", err)
	}

	switch format {
	case "json":
		return s.exportJSON(session, messages)
	case "markdown":
		return s.exportMarkdown(session, messages)
	default:
		return "", fmt.Errorf("不支持的导出格式: %s", format)
	}
}

// exportJSON JSON格式导出
func (s *ExportService) exportJSON(session *domain.Session, messages []domain.Message) (string, error) {
	type exportMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
		Time    string `json:"time"`
	}
	type export struct {
		Title    string       `json:"title"`
		Exported string       `json:"exported_at"`
		Messages []exportMsg  `json:"messages"`
	}

	var msgs []exportMsg
	for _, m := range messages {
		msgs = append(msgs, exportMsg{
			Role:    m.Role,
			Content: m.Content,
			Time:    m.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	data := export{
		Title:    session.Title,
		Exported: time.Now().Format("2006-01-02 15:04:05"),
		Messages: msgs,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// exportMarkdown Markdown格式导出
func (s *ExportService) exportMarkdown(session *domain.Session, messages []domain.Message) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", session.Title))
	sb.WriteString(fmt.Sprintf("> 导出时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	for _, m := range messages {
		switch m.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("## 👤 用户\n\n%s\n\n", m.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("## 🤖 Agent\n\n%s\n\n", m.Content))
		case "system":
			sb.WriteString(fmt.Sprintf("## ⚙️ 系统\n\n%s\n\n", m.Content))
		}
	}

	return sb.String(), nil
}

// ========== 第四阶段：RAG文档服务 ==========

// RAGService RAG文档检索增强生成服务
type RAGService struct {
	repo *repository.Repository
	cfg  *config.Config
}

// NewRAGService 创建RAG服务
func NewRAGService(repo *repository.Repository, cfg *config.Config) *RAGService {
	return &RAGService{repo: repo, cfg: cfg}
}

// IndexDocument 索引文档
func (s *RAGService) IndexDocument(ctx context.Context, userID uint, filename, content string) (*domain.Document, error) {
	// 分块
	chunks := s.chunkText(content, 500)

	doc := &domain.Document{
		UserID:     userID,
		Filename:   filename,
		Content:    content,
		ChunkCount: len(chunks),
		Status:     "ready",
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Document.Create(doc); err != nil {
		return nil, fmt.Errorf("保存文档失败: %w", err)
	}

	// 保存分块
	for i, chunk := range chunks {
		dc := &domain.DocumentChunk{
			DocumentID: doc.ID,
			UserID:     userID,
			ChunkIndex: i,
			Content:    chunk,
			CreatedAt:  time.Now(),
		}
		s.repo.DocumentChunk.Create(dc)
	}

	return doc, nil
}

// chunkText 文本分块
func (s *RAGService) chunkText(text string, chunkSize int) []string {
	runes := []rune(text)
	if len(runes) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}

	return chunks
}

// SemanticSearch 语义搜索文档
func (s *RAGService) SemanticSearch(ctx context.Context, userID uint, query string, limit int) ([]domain.DocumentChunk, error) {
	if limit <= 0 {
		limit = 3
	}

	// 关键词匹配搜索
	return s.repo.DocumentChunk.SearchByKeyword(userID, query, limit), nil
}

// BuildRAGContext 构建RAG上下文
func (s *RAGService) BuildRAGContext(ctx context.Context, userID uint, query string) string {
	chunks, err := s.SemanticSearch(ctx, userID, query, 3)
	if err != nil || len(chunks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n[相关文档参考]\n")
	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("\n--- 文档片段 %d ---\n%s\n", i+1, chunk.Content))
	}

	return sb.String()
}

// ========== 第三阶段：动态插件引擎 ==========

// PluginEngine 插件引擎
type PluginEngine struct {
	repo *repository.Repository
	cfg  *config.Config
}

// NewPluginEngine 创建插件引擎
func NewPluginEngine(repo *repository.Repository, cfg *config.Config) *PluginEngine {
	return &PluginEngine{repo: repo, cfg: cfg}
}

// RegisterPlugin 注册插件
func (e *PluginEngine) RegisterPlugin(plugin *domain.Plugin) error {
	return e.repo.Plugin.Create(plugin)
}

// RunPlugin 运行插件
func (e *PluginEngine) RunPlugin(ctx context.Context, pluginID uint, params map[string]interface{}) (string, error) {
	plugin, err := e.repo.Plugin.GetByID(pluginID)
	if err != nil {
		return "", fmt.Errorf("插件不存在: %w", err)
	}
	if !plugin.Enabled {
		return "", fmt.Errorf("插件已禁用")
	}

	switch plugin.PluginType {
	case "http":
		return e.runHTTPPlugin(ctx, plugin, params)
	case "script":
		return e.runScriptPlugin(ctx, plugin, params)
	default:
		return "", fmt.Errorf("不支持的插件类型: %s", plugin.PluginType)
	}
}

// runHTTPPlugin 运行HTTP插件
func (e *PluginEngine) runHTTPPlugin(ctx context.Context, plugin *domain.Plugin, params map[string]interface{}) (string, error) {
	return fmt.Sprintf("[HTTP插件] %s - 端点: %s, 参数: %v", plugin.Name, plugin.Endpoint, params), nil
}

// runScriptPlugin 运行脚本插件
func (e *PluginEngine) runScriptPlugin(ctx context.Context, plugin *domain.Plugin, params map[string]interface{}) (string, error) {
	return fmt.Sprintf("[脚本插件] %s - 语言: %s, 参数: %v", plugin.Name, plugin.ScriptLang, params), nil
}

// ========== 第三阶段：Email工具 ==========

// EmailToolConfig 邮件工具配置
type EmailToolConfig struct {
	IMAPHost string `json:"imap_host"`
	IMAPPort int    `json:"imap_port"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// EmailTool 邮件工具
type EmailTool struct {
	Config EmailToolConfig
}

// SendEmail 发送邮件（接口占位，需实际SMTP集成）
func (e *EmailTool) SendEmail(to, subject, body string) error {
	// 实际实现需要集成net/smtp或第三方库
	// 这里提供mock实现
	if e.Config.SMTPHost == "" || e.Config.Username == "" {
		return fmt.Errorf("邮件服务未配置")
	}
	return nil
}

// ReadEmails 读取邮件（接口占位，需实际IMAP集成）
func (e *EmailTool) ReadEmails(limit int) ([]string, error) {
	if e.Config.IMAPHost == "" || e.Config.Username == "" {
		return nil, fmt.Errorf("邮件服务未配置")
	}
	return nil, nil
}

// ========== 辅助：余弦相似度 ==========

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// textToVector 简单文本转向量（字符频率）
func textToVector(text string, dim int) []float64 {
	vec := make([]float64, dim)
	runes := []rune(text)
	for i, r := range runes {
		vec[i%dim] += float64(r)
	}
	norm := 0.0
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		sqrtNorm := math.Sqrt(norm)
		for i := range vec {
			vec[i] /= sqrtNorm
		}
	}
	return vec
}

// uuid 引用
var _ = uuid.New
var _ = "important_placeholder"