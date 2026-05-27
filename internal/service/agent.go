package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"eino-agent/internal/domain"
	"eino-agent/internal/model"
	"eino-agent/internal/repository"
	"eino-agent/internal/tool"
)

// AgentService Agent管理服务
type AgentService struct {
	repo           *repository.Repository
	openaiProvider model.ModelProvider
	localProvider  model.ModelProvider
	toolRegistry   *tool.Registry
}

// NewAgentService 创建Agent服务
func NewAgentService(
	repo *repository.Repository,
	openaiProvider model.ModelProvider,
	localProvider model.ModelProvider,
	toolRegistry *tool.Registry,
) *AgentService {
	return &AgentService{
		repo:           repo,
		openaiProvider: openaiProvider,
		localProvider:  localProvider,
		toolRegistry:   toolRegistry,
	}
}

// CreateAgentRequest 创建Agent请求
type CreateAgentRequest struct {
	Name                 string   `json:"name" binding:"required"`
	Description          string   `json:"description"`
	ModelProvider        string   `json:"model_provider" binding:"omitempty"`
	ModelName            string   `json:"model_name" binding:"omitempty"`
	UseGlobalModelConfig bool     `json:"use_global_model_config"`
	SystemPrompt         string   `json:"system_prompt"`
	MaxTokens            int      `json:"max_tokens"`
	Temperature          float64  `json:"temperature"`
	ToolNames            []string `json:"tool_names"`
}

// CreateAgent 创建Agent
func (s *AgentService) CreateAgent(req *CreateAgentRequest, createdBy uint) (*domain.AgentConfig, error) {
	if !req.UseGlobalModelConfig {
		if req.ModelProvider == "" || req.ModelName == "" {
			return nil, errors.New("模型提供者和模型名称不能为空")
		}
		if err := s.validateModel(req.ModelProvider, req.ModelName); err != nil {
			return nil, err
		}
	}

	// 验证工具是否存在
	for _, name := range req.ToolNames {
		if _, ok := s.toolRegistry.Get(name); !ok {
			return nil, fmt.Errorf("工具 %s 不存在", name)
		}
	}

	toolNamesJSON, _ := json.Marshal(req.ToolNames)

	agent := &domain.AgentConfig{
		Name:                 req.Name,
		Description:          req.Description,
		ModelProvider:        req.ModelProvider,
		ModelName:            req.ModelName,
		UseGlobalModelConfig: req.UseGlobalModelConfig,
		SystemPrompt:         req.SystemPrompt,
		MaxTokens:            req.MaxTokens,
		Temperature:          req.Temperature,
		ToolIDs:              string(toolNamesJSON),
		Status:               "stopped",
		CreatedBy:            createdBy,
	}

	if agent.MaxTokens == 0 {
		agent.MaxTokens = 4096
	}
	if agent.Temperature == 0 {
		agent.Temperature = 0.7
	}

	if err := s.repo.Agent.Create(agent); err != nil {
		return nil, errors.New("创建Agent失败")
	}

	return agent, nil
}

// UpdateAgent 更新Agent
func (s *AgentService) UpdateAgent(id uint, req *CreateAgentRequest) (*domain.AgentConfig, error) {
	agent, err := s.repo.Agent.GetByID(id)
	if err != nil {
		return nil, errors.New("Agent不存在")
	}

	if !req.UseGlobalModelConfig {
		if req.ModelProvider != "" && req.ModelName != "" {
			if err := s.validateModel(req.ModelProvider, req.ModelName); err != nil {
				return nil, err
			}
			agent.ModelProvider = req.ModelProvider
			agent.ModelName = req.ModelName
		} else if req.ModelProvider != "" || req.ModelName != "" {
			return nil, errors.New("模型提供者和模型名称不能为空")
		}
	}

	agent.UseGlobalModelConfig = req.UseGlobalModelConfig

	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.SystemPrompt != "" {
		agent.SystemPrompt = req.SystemPrompt
	}
	if req.MaxTokens > 0 {
		agent.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		agent.Temperature = req.Temperature
	}
	if len(req.ToolNames) > 0 {
		toolNamesJSON, _ := json.Marshal(req.ToolNames)
		agent.ToolIDs = string(toolNamesJSON)
	}

	if err := s.repo.Agent.Update(agent); err != nil {
		return nil, errors.New("更新Agent失败")
	}

	return agent, nil
}

// StartAgent 启动Agent
func (s *AgentService) StartAgent(id uint) error {
	agent, err := s.repo.Agent.GetByID(id)
	if err != nil {
		return errors.New("Agent不存在")
	}
	agent.Status = "running"
	return s.repo.Agent.Update(agent)
}

// StopAgent 停止Agent
func (s *AgentService) StopAgent(id uint) error {
	agent, err := s.repo.Agent.GetByID(id)
	if err != nil {
		return errors.New("Agent不存在")
	}
	agent.Status = "stopped"
	return s.repo.Agent.Update(agent)
}

// validateModel 验证模型是否可用
func (s *AgentService) validateModel(provider, modelName string) error {
	switch provider {
	case "openai":
		if s.openaiProvider == nil {
			return errors.New("OpenAI提供者未配置")
		}
	case "local":
		if s.localProvider == nil {
			return errors.New("本地模型提供者未配置")
		}
	case "openrouter", "deepseek":
		return nil
	default:
		return fmt.Errorf("不支持的模型提供者: %s", provider)
	}
	return nil
}
