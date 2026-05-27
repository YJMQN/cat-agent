package service

import (
	"eino-agent/internal/config"
	"eino-agent/internal/model"
	"eino-agent/internal/repository"
	"eino-agent/internal/tool"
)

// Services 服务层集合
type Services struct {
	Auth       *AuthService
	Agent      *AgentService
	Chat       *ChatService
	Admin      *AdminService
	Orchestrate *OrchestrateService // 多Agent协作编排服务

	// 第三阶段：智能能力增强
	Memory      *MemoryService       // 智能记忆系统
	CronScheduler *CronScheduler    // Cron调度器
	PluginEngine *PluginEngine      // 动态插件引擎

	// 第四阶段：体验优化与扩展
	Export      *ExportService      // 对话导出
	RAG         *RAGService         // RAG文档检索
	TokenBudget *TokenBudgetService // Token预算监控
}

// NewServices 创建所有服务实例
func NewServices(repo *repository.Repository, cfg *config.Config) *Services {
	// 初始化模型提供者
	openaiProvider := model.NewOpenAIProvider(cfg.OpenAIBase, cfg.OpenAIKey)
	localProvider := model.NewLocalModelProvider(cfg.LocalModelURL)

	// 初始化工具注册表
	toolRegistry := tool.LoadBuiltinTools()

	// 初始化聊天服务
	chatService := NewChatService(repo, openaiProvider, localProvider, toolRegistry, cfg)

	return &Services{
		Auth:        NewAuthService(repo, cfg),
		Agent:       NewAgentService(repo, openaiProvider, localProvider, toolRegistry),
		Chat:        chatService,
		Admin:       NewAdminService(repo),
		Orchestrate: NewOrchestrateService(repo, chatService, cfg),

		// 第三阶段
		Memory:      NewMemoryService(repo, cfg),
		CronScheduler: NewCronScheduler(repo, chatService, cfg),
		PluginEngine: NewPluginEngine(repo, cfg),

		// 第四阶段
		Export:      NewExportService(repo, cfg),
		RAG:         NewRAGService(repo, cfg),
		TokenBudget: NewTokenBudgetService(repo),
	}
}