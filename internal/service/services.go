package service

import (
	"eino-agent/internal/config"
	"eino-agent/internal/model"
	"eino-agent/internal/repository"
	"eino-agent/internal/tool"
)

// Services 服务层集合
type Services struct {
	Auth        *AuthService
	Agent       *AgentService
	Chat        *ChatService
	Admin       *AdminService
	Orchestrate *OrchestrateService // 多Agent协作编排服务

	// 第三阶段：智能能力增强
	Memory       *MemoryService    // 智能记忆系统
	CronScheduler *CronScheduler  // Cron调度器
	PluginEngine *PluginEngine    // 动态插件引擎

	// 第四阶段：体验优化与扩展
	Export      *ExportService      // 对话导出
	RAG         *RAGService         // RAG文档检索
	TokenBudget *TokenBudgetService // Token预算监控
}

// NewServices 创建所有服务实例
func NewServices(repo *repository.Repository, cfg *config.Config) *Services {
	// 从数据库加载模型配置
	modelConfigs := loadModelConfigs(repo)

	// 初始化模型提供者（基于数据库配置）
	openaiProvider := newProviderFromDB(modelConfigs, "openai")
	localProvider := newProviderFromDB(modelConfigs, "local")

	// 初始化工具注册表
	toolRegistry := tool.LoadBuiltinTools()

	// 初始化聊天服务
	chatService := NewChatService(repo, openaiProvider, localProvider, toolRegistry, cfg)

	return &Services{
		Auth:         NewAuthService(repo, cfg),
		Agent:        NewAgentService(repo, openaiProvider, localProvider, toolRegistry),
		Chat:         chatService,
		Admin:        NewAdminService(repo),
		Orchestrate:  NewOrchestrateService(repo, chatService, cfg),

		// 第三阶段
		Memory:       NewMemoryService(repo, cfg),
		CronScheduler: NewCronScheduler(repo, chatService, cfg),
		PluginEngine:  NewPluginEngine(repo, cfg),

		// 第四阶段
		Export:       NewExportService(repo, cfg),
		RAG:          NewRAGService(repo, cfg),
		TokenBudget:  NewTokenBudgetService(repo),
	}
}

// loadModelConfigs 从数据库加载模型配置列表
func loadModelConfigs(repo *repository.Repository) []model.ProviderConfig {
	configs, err := repo.ModelConfig.List()
	if err != nil || len(configs) == 0 {
		// 降级：返回默认配置
		return []model.ProviderConfig{
			{Provider: "openai", BaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-4o-mini"},
			{Provider: "local", BaseURL: "http://localhost:11434", DefaultModel: "qwen2.5"},
		}
	}

	var result []model.ProviderConfig
	for _, c := range configs {
		result = append(result, model.ProviderConfig{
			Provider:     c.Provider,
			BaseURL:      c.BaseURL,
			APIKey:       c.APIKey,
			DefaultModel: c.DefaultModel,
		})
	}
	return result
}

// newProviderFromDB 根据数据库配置创建模型提供者
func newProviderFromDB(configs []model.ProviderConfig, providerName string) model.ModelProvider {
	for _, c := range configs {
		if c.Provider == providerName {
			switch providerName {
			case "openai":
				return model.NewOpenAIProviderFromConfig(&c)
			case "local":
				return model.NewLocalModelProviderFromConfig(&c)
			default:
				return model.NewOpenAIProviderFromConfig(&c)
			}
		}
	}
	// 未找到，返回默认
	switch providerName {
	case "local":
		return model.NewLocalModelProvider("http://localhost:11434")
	default:
		return model.NewOpenAIProvider("https://api.openai.com/v1", "")
	}
}
