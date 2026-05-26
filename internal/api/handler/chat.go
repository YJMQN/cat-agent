package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"eino-agent/internal/domain"
	"eino-agent/internal/service"

	"github.com/gin-gonic/gin"
)

// ChatHandler 对话处理器
type ChatHandler struct {
	svc *service.ChatService
}

// NewChatHandler 创建对话处理器
func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// HandleChat 同步对话接口
func (h *ChatHandler) HandleChat(c *gin.Context) {
	var req struct {
		AgentID   uint   `json:"agent_id" binding:"required"`
		SessionID uint   `json:"session_id"`
		Content   string `json:"content" binding:"required"`
		VendorKey string `json:"vendor_key"`
		BaseURL   string `json:"base_url"`
		APIKey    string `json:"api_key"`
		ModelName string `json:"model_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		uid = 1 // 默认用户 (未登录时)
	}

	input := &service.ChatInput{
		AgentID:   req.AgentID,
		SessionID: req.SessionID,
		UserID:    uid,
		Content:   req.Content,
		VendorKey: req.VendorKey,
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		ModelName: req.ModelName,
	}

	// 收集所有事件
	var events []domain.StreamEvent
	err := h.svc.HandleChat(c.Request.Context(), input, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"events": events,
		},
	})
}

// HandleStream SSE流式对话接口
func (h *ChatHandler) HandleStream(c *gin.Context) {
	agentID := c.Query("agent_id")
	sessionID := c.Query("session_id")
	content := c.Query("content")
	vendorKey := c.Query("vendor_key")
	baseURL := c.Query("vendor_base_url")
	apiKey := c.Query("vendor_api_key")
	modelName := c.Query("vendor_model")

	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少content参数"})
		return
	}

	var agentIDUint, sessionIDUint uint
	fmt.Sscanf(agentID, "%d", &agentIDUint)
	fmt.Sscanf(sessionID, "%d", &sessionIDUint)

	if agentIDUint == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少agent_id参数"})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		uid = 1
	}

	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	input := &service.ChatInput{
		AgentID:   agentIDUint,
		SessionID: sessionIDUint,
		UserID:    uid,
		Content:   content,
		VendorKey: vendorKey,
		BaseURL:   baseURL,
		APIKey:    apiKey,
		ModelName: modelName,
	}

	eventCh := make(chan domain.StreamEvent, 64)

	go func() {
		defer close(eventCh)
		h.svc.HandleChat(c.Request.Context(), input, eventCh)
	}()

	// 流式写入SSE
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return false
			}
			data, _ := json.Marshal(event)
			c.SSEvent("message", string(data))
			return event.Type != "done"
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// HandleStreamPost POST方式的SSE流式对话
func (h *ChatHandler) HandleStreamPost(c *gin.Context) {
	var req struct {
		AgentID   uint   `json:"agent_id" binding:"required"`
		SessionID uint   `json:"session_id"`
		Content   string `json:"content" binding:"required"`
		VendorKey string `json:"vendor_key"`
		BaseURL   string `json:"base_url"`
		APIKey    string `json:"api_key"`
		ModelName string `json:"model_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		uid = 1
	}

	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	input := &service.ChatInput{
		AgentID:   req.AgentID,
		SessionID: req.SessionID,
		UserID:    uid,
		Content:   req.Content,
		VendorKey: req.VendorKey,
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		ModelName: req.ModelName,
	}

	eventCh := make(chan domain.StreamEvent, 64)

	go func() {
		defer close(eventCh)
		h.svc.HandleChat(c.Request.Context(), input, eventCh)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return false
			}
			data, _ := json.Marshal(event)
			c.SSEvent("message", string(data))
			return event.Type != "done"
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// HandleToolConfirmation 处理工具执行确认
func (h *ChatHandler) HandleToolConfirmation(c *gin.Context) {
	var req struct {
		ConfirmationID string `json:"confirmation_id" binding:"required"`
		Approved       bool   `json:"approved"`
		SessionID      uint   `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		uid = 1
	}

	event, err := h.svc.ExecutePendingTool(req.ConfirmationID, req.Approved, req.SessionID, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"tool":    event.Tool,
		"content": event.Content,
	}})
}
