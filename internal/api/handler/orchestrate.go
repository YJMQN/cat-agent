package handler

import (
	"net/http"

	"cat-agent/internal/service"

	"github.com/gin-gonic/gin"
)

// OrchestrateHandler 多Agent协作编排处理器
type OrchestrateHandler struct {
	svc *service.OrchestrateService
}

// NewOrchestrateHandler 创建编排处理器
func NewOrchestrateHandler(svc *service.OrchestrateService) *OrchestrateHandler {
	return &OrchestrateHandler{svc: svc}
}

// CreateWorkflow 创建工作流
func (h *OrchestrateHandler) CreateWorkflow(c *gin.Context) {
	var input service.WorkflowInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")
	input.UserID = userID.(uint)

	output, err := h.svc.CreateWorkflow(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}

// GetWorkflow 获取工作流详情
func (h *OrchestrateHandler) GetWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少工作流ID"})
		return
	}

	// 解析ID
	var id uint
	if err := parseUintParam(workflowID, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作流ID"})
		return
	}

	output, err := h.svc.GetWorkflow(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}

// ListWorkflows 列出工作流
func (h *OrchestrateHandler) ListWorkflows(c *gin.Context) {
	userID, _ := c.Get("user_id")

	outputs, err := h.svc.ListWorkflows(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, outputs)
}

// DeleteWorkflow 删除工作流
func (h *OrchestrateHandler) DeleteWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少工作流ID"})
		return
	}

	var id uint
	if err := parseUintParam(workflowID, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作流ID"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := h.svc.DeleteWorkflow(c.Request.Context(), id, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "工作流已删除"})
}

// ExecuteWorkflow 执行工作流
func (h *OrchestrateHandler) ExecuteWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少工作流ID"})
		return
	}

	var id uint
	if err := parseUintParam(workflowID, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作流ID"})
		return
	}

	var input struct {
		Input map[string]interface{} `json:"input"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	execInput := &service.ExecutionInput{
		WorkflowID: id,
		UserID:     userID.(uint),
		Input:      input.Input,
	}

	output, err := h.svc.ExecuteWorkflow(c.Request.Context(), execInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}

// GetWorkflowStatus 获取工作流执行状态
func (h *OrchestrateHandler) GetWorkflowStatus(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少工作流ID"})
		return
	}

	var id uint
	if err := parseUintParam(workflowID, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作流ID"})
		return
	}

	output, err := h.svc.GetWorkflowStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}

// StopWorkflow 停止工作流执行
func (h *OrchestrateHandler) StopWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少工作流ID"})
		return
	}

	var id uint
	if err := parseUintParam(workflowID, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作流ID"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := h.svc.StopWorkflow(c.Request.Context(), id, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "工作流已停止"})
}

// parseUintParam 解析uint参数
func parseUintParam(s string, out *uint) error {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return nil // 返回错误
		}
	}
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	*out = uint(n)
	return nil
}