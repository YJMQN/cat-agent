package handler

import (
	"eino-agent/internal/service"
)

// Handlers 所有HTTP处理器集合
type Handlers struct {
	Auth        *AuthHandler
	Chat        *ChatHandler
	Admin        *AdminHandler
	Orchestrate *OrchestrateHandler // 多Agent协作编排处理器
}

// NewHandlers 创建所有处理器
func NewHandlers(svc *service.Services) *Handlers {
	return &Handlers{
		Auth:        NewAuthHandler(svc.Auth),
		Chat:        NewChatHandler(svc.Chat),
		Admin:       NewAdminHandler(svc.Admin, svc.Agent),
		Orchestrate: NewOrchestrateHandler(svc.Orchestrate), // 多Agent协作编排
	}
}