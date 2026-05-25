package handler

import (
	"net/http"
	"strconv"

	"eino-agent/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理接口处理器
type AdminHandler struct {
	adminSvc *service.AdminService
	agentSvc *service.AgentService
}

// NewAdminHandler 创建管理处理器
func NewAdminHandler(adminSvc *service.AdminService, agentSvc *service.AgentService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, agentSvc: agentSvc}
}

// ========== Agent管理 ==========

// ListAgents 获取Agent列表
func (h *AdminHandler) ListAgents(c *gin.Context) {
	agents, err := h.adminSvc.ListAgents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": agents})
}

// CreateAgent 创建Agent
func (h *AdminHandler) CreateAgent(c *gin.Context) {
	var req service.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	createdBy, _ := userID.(uint)

	agent, err := h.agentSvc.CreateAgent(&req, createdBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": agent})
}

// GetAgent 获取Agent详情
func (h *AdminHandler) GetAgent(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	agent, err := h.adminSvc.GetAgent(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": agent})
}

// UpdateAgent 更新Agent
func (h *AdminHandler) UpdateAgent(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var req service.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	agent, err := h.agentSvc.UpdateAgent(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": agent})
}

// DeleteAgent 删除Agent
func (h *AdminHandler) DeleteAgent(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.adminSvc.DeleteAgent(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// StartAgent 启动Agent
func (h *AdminHandler) StartAgent(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.agentSvc.StartAgent(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "启动成功"})
}

// StopAgent 停止Agent
func (h *AdminHandler) StopAgent(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.agentSvc.StopAgent(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "停止成功"})
}

// ========== 工具管理 ==========

// ListTools 获取工具列表
func (h *AdminHandler) ListTools(c *gin.Context) {
	tools, err := h.adminSvc.ListTools()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tools})
}

// RegisterTool 注册工具
func (h *AdminHandler) RegisterTool(c *gin.Context) {
	var req service.RegisterToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	t, err := h.adminSvc.RegisterTool(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": t})
}

// GetTool 获取工具详情
func (h *AdminHandler) GetTool(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	t, err := h.adminSvc.GetTool(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工具不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": t})
}

// UpdateTool 更新工具
func (h *AdminHandler) UpdateTool(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var req service.RegisterToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	t, err := h.adminSvc.UpdateTool(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": t})
}

// DeleteTool 删除工具
func (h *AdminHandler) DeleteTool(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.adminSvc.DeleteTool(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// TestTool 测试工具
func (h *AdminHandler) TestTool(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var args map[string]interface{}
	if err := c.ShouldBindJSON(&args); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	result, err := h.adminSvc.TestTool(uint(id), args)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"result": result}})
}

// EnableTool 启用工具
func (h *AdminHandler) EnableTool(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.adminSvc.EnableTool(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "启用成功"})
}

// DisableTool 禁用工具
func (h *AdminHandler) DisableTool(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.adminSvc.DisableTool(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "禁用成功"})
}

// ========== 会话管理 ==========

// ListSessions 获取会话列表
func (h *AdminHandler) ListSessions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	sessions, total, err := h.adminSvc.ListSessions(page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  sessions,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetSession 获取会话详情
func (h *AdminHandler) GetSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	session, err := h.adminSvc.GetSession(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": session})
}

// GetSessionMessages 获取会话消息
func (h *AdminHandler) GetSessionMessages(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	msgs, err := h.adminSvc.GetSessionMessages(uint(id), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": msgs})
}

// InjectMessage 注入系统消息
func (h *AdminHandler) InjectMessage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.adminSvc.InjectMessage(uint(id), req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注入成功"})
}

// ResetSession 重置会话
func (h *AdminHandler) ResetSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.adminSvc.ResetSession(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "重置成功"})
}

// ========== 统计 ==========

// StatsOverview 获取统计概览
func (h *AdminHandler) StatsOverview(c *gin.Context) {
	stats, err := h.adminSvc.GetStatsOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// TokenUsage 获取Token用量
func (h *AdminHandler) TokenUsage(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	usage, err := h.adminSvc.GetTokenUsage(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": usage})
}

// ToolRanking 获取工具排行
func (h *AdminHandler) ToolRanking(c *gin.Context) {
	ranking, err := h.adminSvc.GetToolRanking()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ranking})
}

// ========== 记忆管理 ==========

// ListMemories 获取记忆列表
func (h *AdminHandler) ListMemories(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 32)

	memories, err := h.adminSvc.ListMemories(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": memories})
}

// GetMemory 获取记忆详情
func (h *AdminHandler) GetMemory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	mem, err := h.adminSvc.GetMemory(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记忆不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": mem})
}

// UpdateMemory 更新记忆
func (h *AdminHandler) UpdateMemory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.adminSvc.UpdateMemory(uint(id), req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteMemory 删除记忆
func (h *AdminHandler) DeleteMemory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.adminSvc.DeleteMemory(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ========== 用户管理 ==========

// ListUsers 获取用户列表
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.adminSvc.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// UpdateUserRole 更新用户角色
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.adminSvc.UpdateUserRole(uint(id), req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}
