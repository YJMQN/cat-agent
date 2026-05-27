package handler

import (
	"eino-agent/internal/repository"
	"eino-agent/internal/service"
)

// Handlers 所有HTTP处理器集合
type Handlers struct {
	Auth        *AuthHandler
	Chat        *ChatHandler
	Admin        *AdminHandler
	Orchestrate *OrchestrateHandler // 多Agent协作编排处理器

	// 第三阶段：智能能力增强
	Memory    *MemoryHandler
	Cron      *CronHandler
	Plugin    *PluginHandler
	WebSocket *WebSocketHandler

	// 第四阶段：体验优化与扩展
	Export *ExportHandler
	RAG    *RAGHandler
	Budget *TokenBudgetHandler

	// 模型配置（替代环境变量）
	ModelConfig *ModelConfigHandler
}

// NewHandlers 创建所有处理器
func NewHandlers(svc *service.Services, repo *repository.Repository) *Handlers {
	return &Handlers{
		Auth:        NewAuthHandler(svc.Auth),
		Chat:        NewChatHandler(svc.Chat),
		Admin:       NewAdminHandler(svc.Admin, svc.Agent),
		Orchestrate: NewOrchestrateHandler(svc.Orchestrate),

		// 第三阶段
		Memory:    NewMemoryHandler(svc.Memory),
		Cron:      NewCronHandler(svc.CronScheduler),
		Plugin:    NewPluginHandler(svc.PluginEngine),
		WebSocket: NewWebSocketHandler(),

		// 第四阶段
		Export: NewExportHandler(svc.Export),
		RAG:    NewRAGHandler(svc.RAG),
		Budget: NewTokenBudgetHandler(svc.TokenBudget),

		// 模型配置
		ModelConfig: NewModelConfigHandler(repo),
	}
}