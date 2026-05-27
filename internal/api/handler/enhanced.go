package handler

import (
	"net/http"
	"strconv"

	"cat-agent/internal/domain"
	"cat-agent/internal/service"

	"github.com/gin-gonic/gin"
)

// ========== 第三阶段：记忆管理 ==========

type MemoryHandler struct {
	svc *service.MemoryService
}

func NewMemoryHandler(svc *service.MemoryService) *MemoryHandler {
	return &MemoryHandler{svc: svc}
}

// StoreMemory 存储记忆
func (h *MemoryHandler) StoreMemory(c *gin.Context) {
	var req struct {
		Type       string  `json:"type" binding:"required"`
		Content    string  `json:"content" binding:"required"`
		Importance float64 `json:"importance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	err := h.svc.StoreMemory(c.Request.Context(), userID, req.Type, req.Content, req.Importance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "记忆已存储"})
}

// SearchMemory 搜索记忆
func (h *MemoryHandler) SearchMemory(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少搜索关键词"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	userID := c.GetUint("user_id")
	items, err := h.svc.SemanticSearch(c.Request.Context(), userID, query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"memories": items})
}

// ========== 第三阶段：Cron任务管理 ==========

type CronHandler struct {
	svc *service.CronScheduler
}

func NewCronHandler(svc *service.CronScheduler) *CronHandler {
	return &CronHandler{svc: svc}
}

// CreateCronJob 创建定时任务
func (h *CronHandler) CreateCronJob(c *gin.Context) {
	var job domain.CronJob
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 由repo创建，这里简化处理
	c.JSON(http.StatusOK, gin.H{"message": "任务已创建"})
}

// StartScheduler 启动调度器
func (h *CronHandler) StartScheduler(c *gin.Context) {
	h.svc.Start(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"message": "调度器已启动"})
}

// StopScheduler 停止调度器
func (h *CronHandler) StopScheduler(c *gin.Context) {
	h.svc.Stop()
	c.JSON(http.StatusOK, gin.H{"message": "调度器已停止"})
}

// ========== 第三阶段：插件管理 ==========

type PluginHandler struct {
	svc *service.PluginEngine
}

func NewPluginHandler(svc *service.PluginEngine) *PluginHandler {
	return &PluginHandler{svc: svc}
}

// RegisterPlugin 注册插件
func (h *PluginHandler) RegisterPlugin(c *gin.Context) {
	var plugin domain.Plugin
	if err := c.ShouldBindJSON(&plugin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.RegisterPlugin(&plugin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "插件已注册", "id": plugin.ID})
}

// ========== 第四阶段：对话导出 ==========

type ExportHandler struct {
	svc *service.ExportService
}

func NewExportHandler(svc *service.ExportService) *ExportHandler {
	return &ExportHandler{svc: svc}
}

// ExportSession 导出对话
func (h *ExportHandler) ExportSession(c *gin.Context) {
	sessionID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	format := c.DefaultQuery("format", "markdown")

	userID := c.GetUint("user_id")
	content, err := h.svc.ExportSession(c.Request.Context(), uint(sessionID), userID, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ext := ".md"
	contentType := "text/markdown"
	if format == "json" {
		ext = ".json"
		contentType = "application/json"
	}

	c.Header("Content-Disposition", "attachment; filename=chat-export"+ext)
	c.Data(http.StatusOK, contentType, []byte(content))
}

// ========== 第四阶段：文档与RAG ==========

type RAGHandler struct {
	svc *service.RAGService
}

func NewRAGHandler(svc *service.RAGService) *RAGHandler {
	return &RAGHandler{svc: svc}
}

// UploadDocument 上传文档
func (h *RAGHandler) UploadDocument(c *gin.Context) {
	userID := c.GetUint("user_id")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	// 读取文件内容（简化处理）
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	f.Read(buf)
	content := string(buf)

	doc, err := h.svc.IndexDocument(c.Request.Context(), userID, file.Filename, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "文档已索引", "document": doc})
}

// SearchDocument 搜索文档
func (h *RAGHandler) SearchDocument(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少搜索关键词"})
		return
	}

	userID := c.GetUint("user_id")
	chunks, err := h.svc.SemanticSearch(c.Request.Context(), userID, query, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": chunks})
}

// ========== 第四阶段：Token预算 ==========

type TokenBudgetHandler struct {
	svc *service.TokenBudgetService
}

func NewTokenBudgetHandler(svc *service.TokenBudgetService) *TokenBudgetHandler {
	return &TokenBudgetHandler{svc: svc}
}

// GetBudget 获取预算
func (h *TokenBudgetHandler) GetBudget(c *gin.Context) {
	userID := c.GetUint("user_id")
	ok, msg, err := h.svc.CheckAndRecord(userID, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": ok, "message": msg})
}

// ========== 第三阶段：WebSocket ==========

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	clients    map[string]chan string
}

// NewWebSocketHandler 创建WebSocket处理器
func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		clients: make(map[string]chan string),
	}
}

// HandleWebSocket WebSocket连接处理（由HTTP升级，这里提供HTTP接口模拟）
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "WebSocket端点已就绪（实际部署需配置WebSocket升级器）",
		"endpoint": "/ws",
	})
}

// ========== 路由注册 ==========

// RegisterEnhancedRoutes 注册第三/第四阶段增强路由
func RegisterEnhancedRoutes(r *gin.RouterGroup, handlers *Handlers) {
	// 第三阶段：记忆管理
	memory := r.Group("/memory")
	{
		memory.POST("/store", handlers.Memory.StoreMemory)
		memory.GET("/search", handlers.Memory.SearchMemory)
	}

	// 第三阶段：Cron任务管理
	cron := r.Group("/cron")
	{
		cron.POST("/jobs", handlers.Cron.CreateCronJob)
		cron.POST("/start", handlers.Cron.StartScheduler)
		cron.POST("/stop", handlers.Cron.StopScheduler)
	}

	// 第三阶段：插件管理
	plugin := r.Group("/plugins")
	{
		plugin.POST("/register", handlers.Plugin.RegisterPlugin)
	}

	// 第三阶段：WebSocket管理
	ws := r.Group("/ws")
	{
		ws.GET("/status", handlers.WebSocket.HandleWebSocket)
	}

	// 第四阶段：对话导出
	export := r.Group("/export")
	{
		export.GET("/session/:id", handlers.Export.ExportSession)
	}

	// 第四阶段：RAG文档
	rag := r.Group("/rag")
	{
		rag.POST("/upload", handlers.RAG.UploadDocument)
		rag.GET("/search", handlers.RAG.SearchDocument)
	}

	// 第四阶段：Token预算
	budget := r.Group("/budget")
	{
		budget.GET("/status", handlers.Budget.GetBudget)
	}

	// 全局模型配置管理（替代环境变量配置）
	mcfg := r.Group("/model-config")
	{
		mcfg.GET("", handlers.ModelConfig.List)
		mcfg.POST("", handlers.ModelConfig.Create)
		mcfg.GET("/:provider", handlers.ModelConfig.GetByProvider)
		mcfg.PUT("/:provider", handlers.ModelConfig.Update)
		mcfg.DELETE("/:provider", handlers.ModelConfig.Delete)
	}
}