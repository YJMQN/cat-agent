package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"eino-agent/internal/domain"
	"eino-agent/internal/repository"
)

// AdminService 管理服务
type AdminService struct {
	repo *repository.Repository
}

// NewAdminService 创建管理服务
func NewAdminService(repo *repository.Repository) *AdminService {
	return &AdminService{repo: repo}
}

// ========== 审计日志 ==========

// Audit 记录审计日志
func (s *AdminService) Audit(userID uint, action, resource, detail, ip string) error {
	log := &domain.AuditLog{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Detail:    truncateString(detail, 500),
		IP:        ip,
		CreatedAt: time.Now(),
	}
	return s.repo.Audit.Create(log)
}

// ListAuditLogs 获取审计日志
func (s *AdminService) ListAuditLogs(limit int) ([]domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	logs, _, err := s.repo.Audit.List(1, limit)
	return logs, err
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ========== Agent管理 ==========

func (s *AdminService) ListAgents() ([]domain.AgentConfig, error) {
	return s.repo.Agent.List()
}

func (s *AdminService) GetAgent(id uint) (*domain.AgentConfig, error) {
	return s.repo.Agent.GetByID(id)
}

func (s *AdminService) DeleteAgent(id uint) error {
	return s.repo.Agent.Delete(id)
}

// ========== 工具管理 ==========

func (s *AdminService) ListTools() ([]domain.Tool, error) {
	return s.repo.Tool.List()
}

func (s *AdminService) GetTool(id uint) (*domain.Tool, error) {
	return s.repo.Tool.GetByID(id)
}

// RegisterToolRequest 注册工具请求
type RegisterToolRequest struct {
	Name         string `json:"name" binding:"required"`
	DisplayName  string `json:"display_name"`
	Description  string `json:"description"`
	ToolType     string `json:"tool_type" binding:"required"`
	ParamsSchema string `json:"params_schema"`
	HTTPEndpoint string `json:"http_endpoint"`
	HTTPMethod   string `json:"http_method"`
	HTTPHeaders  string `json:"http_headers"`
	ScriptLang   string `json:"script_lang"`
	ScriptCode   string `json:"script_code"`
}

func (s *AdminService) RegisterTool(req *RegisterToolRequest) (*domain.Tool, error) {
	// 检查名称是否已存在
	if _, err := s.repo.Tool.GetByName(req.Name); err == nil {
		return nil, errors.New("工具名称已存在")
	}

	// 验证参数Schema是否为合法JSON
	if req.ParamsSchema != "" {
		var js map[string]interface{}
		if err := json.Unmarshal([]byte(req.ParamsSchema), &js); err != nil {
			return nil, errors.New("参数Schema不是合法的JSON")
		}
	}

	tool := &domain.Tool{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		ToolType:     req.ToolType,
		ParamsSchema: req.ParamsSchema,
		HTTPEndpoint: req.HTTPEndpoint,
		HTTPMethod:   req.HTTPMethod,
		HTTPHeaders:  req.HTTPHeaders,
		ScriptLang:   req.ScriptLang,
		ScriptCode:   req.ScriptCode,
		Enabled:      true,
	}

	if tool.DisplayName == "" {
		tool.DisplayName = tool.Name
	}

	if err := s.repo.Tool.Create(tool); err != nil {
		return nil, errors.New("注册工具失败")
	}

	return tool, nil
}

func (s *AdminService) UpdateTool(id uint, req *RegisterToolRequest) (*domain.Tool, error) {
	t, err := s.repo.Tool.GetByID(id)
	if err != nil {
		return nil, errors.New("工具不存在")
	}

	if req.DisplayName != "" {
		t.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		t.Description = req.Description
	}
	if req.ParamsSchema != "" {
		var js map[string]interface{}
		if err := json.Unmarshal([]byte(req.ParamsSchema), &js); err != nil {
			return nil, errors.New("参数Schema不是合法的JSON")
		}
		t.ParamsSchema = req.ParamsSchema
	}
	if req.HTTPEndpoint != "" {
		t.HTTPEndpoint = req.HTTPEndpoint
	}
	if req.HTTPMethod != "" {
		t.HTTPMethod = req.HTTPMethod
	}
	if req.HTTPHeaders != "" {
		t.HTTPHeaders = req.HTTPHeaders
	}
	if req.ScriptCode != "" {
		t.ScriptCode = req.ScriptCode
	}

	if err := s.repo.Tool.Update(t); err != nil {
		return nil, errors.New("更新工具失败")
	}

	return t, nil
}

func (s *AdminService) DeleteTool(id uint) error {
	return s.repo.Tool.Delete(id)
}

func (s *AdminService) EnableTool(id uint) error {
	t, err := s.repo.Tool.GetByID(id)
	if err != nil {
		return errors.New("工具不存在")
	}
	t.Enabled = true
	return s.repo.Tool.Update(t)
}

func (s *AdminService) DisableTool(id uint) error {
	t, err := s.repo.Tool.GetByID(id)
	if err != nil {
		return errors.New("工具不存在")
	}
	t.Enabled = false
	return s.repo.Tool.Update(t)
}

func (s *AdminService) TestTool(id uint, args map[string]interface{}) (string, error) {
	// 简化的工具测试 - 返回参数校验结果
	t, err := s.repo.Tool.GetByID(id)
	if err != nil {
		return "", errors.New("工具不存在")
	}

	if !t.Enabled {
		return "", errors.New("工具已禁用")
	}

	// 解析Schema并校验参数
	if t.ParamsSchema != "" {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(t.ParamsSchema), &schema); err != nil {
			return "", errors.New("工具Schema格式错误")
		}

		required, _ := schema["required"].([]interface{})
		for _, r := range required {
			key, ok := r.(string)
			if !ok {
				continue
			}
			if _, exists := args[key]; !exists {
				return "", fmt.Errorf("缺少必填参数: %s", key)
			}
		}
	}

	resultJSON, _ := json.MarshalIndent(args, "", "  ")
	return fmt.Sprintf("参数校验通过，工具类型: %s\n输入参数:\n%s", t.ToolType, string(resultJSON)), nil
}

// ========== 会话管理 ==========

func (s *AdminService) ListSessions(page, size int) ([]domain.Session, int64, error) {
	return s.repo.Session.List(page, size)
}

func (s *AdminService) GetSession(id uint) (*domain.Session, error) {
	return s.repo.Session.GetByID(id)
}

func (s *AdminService) GetSessionMessages(sessionID uint, limit int) ([]domain.Message, error) {
	return s.repo.Message.GetBySession(sessionID, limit)
}

func (s *AdminService) InjectMessage(sessionID uint, content string) error {
	return s.repo.Message.Create(&domain.Message{
		SessionID: sessionID,
		Role:      "system",
		Content:   content,
	})
}

func (s *AdminService) ResetSession(sessionID uint) error {
	session, err := s.repo.Session.GetByID(sessionID)
	if err != nil {
		return errors.New("会话不存在")
	}
	session.Status = "closed"
	return s.repo.Session.Update(session)
}

// ========== 统计 ==========

func (s *AdminService) GetStatsOverview() (*domain.AdminStats, error) {
	return s.repo.Stats.Overview()
}

func (s *AdminService) GetTokenUsage(days int) ([]domain.StatsTokenUsage, error) {
	return s.repo.Stats.TokenUsageByDay(days)
}

func (s *AdminService) GetToolRanking() ([]map[string]interface{}, error) {
	return s.repo.Stats.ToolRanking()
}

// ========== 记忆管理 ==========

func (s *AdminService) ListMemories(userID uint) ([]domain.Memory, error) {
	if userID > 0 {
		return s.repo.Memory.GetByUser(userID)
	}
	// 返回所有记忆 (简化实现)
	return []domain.Memory{}, nil
}

func (s *AdminService) GetMemory(id uint) (*domain.Memory, error) {
	return s.repo.Memory.GetByID(id)
}

func (s *AdminService) UpdateMemory(id uint, content string) error {
	mem, err := s.repo.Memory.GetByID(id)
	if err != nil {
		return errors.New("记忆不存在")
	}
	mem.Content = content
	return s.repo.Memory.Update(mem)
}

func (s *AdminService) DeleteMemory(id uint) error {
	return s.repo.Memory.Delete(id)
}

// ========== 用户管理 ==========

func (s *AdminService) ListUsers() ([]domain.User, error) {
	return s.repo.User.List()
}

func (s *AdminService) UpdateUserRole(id uint, role string) error {
	if role != "admin" && role != "operator" && role != "user" {
		return errors.New("无效的角色")
	}
	return s.repo.User.UpdateRole(id, role)
}
