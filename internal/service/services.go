package service

import (
	"eino-agent/internal/config"
	"eino-agent/internal/model"
	"eino-agent/internal/repository"
	"eino-agent/internal/tool"
)

// Services 服务层集合
type Services struct {
	Auth    *AuthService
	Agent   *AgentService
	Chat    *ChatService
	Admin   *AdminService
}

// NewServices 创建所有服务实例
func NewServices(repo *repository.Repository, cfg *config.Config) *Services {
	// 初始化模型提供者
	openaiProvider := model.NewOpenAIProvider(cfg.OpenAIBase, cfg.OpenAIKey)
	localProvider := model.NewLocalModelProvider(cfg.LocalModelURL)

	// 初始化工具注册表
	toolRegistry := tool.LoadBuiltinTools()

	return &Services{
		Auth:    NewAuthService(repo, cfg),
		Agent:   NewAgentService(repo, openaiProvider, localProvider, toolRegistry),
		Chat:    NewChatService(repo, openaiProvider, localProvider, toolRegistry, cfg),
		Admin:   NewAdminService(repo),
	}
}
