package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eino-agent/internal/api/handler"
	"eino-agent/internal/api/middleware"
	"eino-agent/internal/config"
	"eino-agent/internal/repository"
	"eino-agent/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置 - 第一阶段安全加固: JWT密钥检查
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 初始化数据库
	db, err := repository.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 初始化仓储层
	repo := repository.New(db)

	// 初始化服务层
	svc := service.NewServices(repo, cfg)

	// 初始化处理器
	h := handler.NewHandlers(svc)

	// 第三阶段：启动Cron调度器
	if cfg.CronEnabled {
		svc.CronScheduler.Start(context.Background())
		log.Println("Cron调度器已启动")
	}

	// 设置路由
	r := setupRouter(h, cfg)

	// 第一阶段安全加固: 使用http.Server实现优雅关闭
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second, // 流式响应可能需要较长写入时间
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务
	go func() {
		log.Printf("Agent服务启动于 %s (环境: %s)", addr, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 第一阶段安全加固: 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("收到关闭信号，正在优雅关闭服务...")

	// 给予30秒时间处理现有连接
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("服务强制关闭: %v", err)
	} else {
		log.Println("服务已优雅关闭")
	}
}

func setupRouter(h *handler.Handlers, cfg *config.Config) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CORS())

	// 第一阶段安全加固: 速率限制中间件
	r.Use(middleware.RateLimit(cfg.RateLimitRequests, cfg.RateLimitBurst))

	// 健康检查 (公开)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 第一阶段安全加固: 对话端点添加认证保护
	// 使用JWT认证中间件保护聊天端点
	chatGroup := r.Group("/api/chat")
	chatGroup.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		chatGroup.POST("", h.Chat.HandleChat)
		chatGroup.GET("/stream", h.Chat.HandleStream)
		chatGroup.POST("/tool/confirm", h.Chat.HandleToolConfirmation)
	}

	// 公开路由 (登录/注册)
	public := r.Group("/api/auth")
	{
		public.POST("/login", h.Auth.Login)
		public.POST("/register", h.Auth.Register)
	}

	// 管理API - 需要认证
	admin := r.Group("/api/admin")
	admin.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		// Agent管理
		admin.GET("/agents", h.Admin.ListAgents)
		admin.POST("/agents", h.Admin.CreateAgent)
		admin.GET("/agents/:id", h.Admin.GetAgent)
		admin.PUT("/agents/:id", h.Admin.UpdateAgent)
		admin.DELETE("/agents/:id", h.Admin.DeleteAgent)
		admin.POST("/agents/:id/start", h.Admin.StartAgent)
		admin.POST("/agents/:id/stop", h.Admin.StopAgent)

		// 工具管理
		admin.GET("/tools", h.Admin.ListTools)
		admin.POST("/tools", h.Admin.RegisterTool)
		admin.GET("/tools/:id", h.Admin.GetTool)
		admin.PUT("/tools/:id", h.Admin.UpdateTool)
		admin.DELETE("/tools/:id", h.Admin.DeleteTool)
		admin.POST("/tools/:id/test", h.Admin.TestTool)
		admin.POST("/tools/:id/enable", h.Admin.EnableTool)
		admin.POST("/tools/:id/disable", h.Admin.DisableTool)

		// 会话监控
		admin.GET("/sessions", h.Admin.ListSessions)
		admin.GET("/sessions/:id", h.Admin.GetSession)
		admin.GET("/sessions/:id/messages", h.Admin.GetSessionMessages)
		admin.POST("/sessions/:id/inject", h.Admin.InjectMessage)
		admin.POST("/sessions/:id/reset", h.Admin.ResetSession)

		// 数据统计
		admin.GET("/stats/overview", h.Admin.StatsOverview)
		admin.GET("/stats/tokens", h.Admin.TokenStats)
		admin.GET("/stats/tools", h.Admin.ToolStats)

		// 记忆库
		admin.GET("/memories", h.Admin.ListMemories)
		admin.GET("/memories/:id", h.Admin.GetMemory)
		admin.PUT("/memories/:id", h.Admin.UpdateMemory)
		admin.DELETE("/memories/:id", h.Admin.DeleteMemory)

		// 用户管理 (管理员专属)
		adminUsers := admin.Group("/users")
		adminUsers.Use(middleware.AdminOnly())
		{
			adminUsers.GET("", h.Admin.ListUsers)
			adminUsers.POST("", h.Admin.CreateUser)
			adminUsers.GET("/:id", h.Admin.GetUser)
			adminUsers.PUT("/:id", h.Admin.UpdateUser)
			adminUsers.DELETE("/:id", h.Admin.DeleteUser)
		}

		// 第三阶段：审计日志
		admin.GET("/audit-logs", h.Admin.ListAuditLogs)
	}

	// 多Agent协作编排API - 新增
	orchestrate := r.Group("/api/orchestrate")
	orchestrate.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		orchestrate.POST("/workflow", h.Orchestrate.CreateWorkflow)
		orchestrate.GET("/workflow/:id", h.Orchestrate.GetWorkflow)
		orchestrate.POST("/workflow/:id/execute", h.Orchestrate.ExecuteWorkflow)
		orchestrate.GET("/workflow/:id/status", h.Orchestrate.GetWorkflowStatus)
		orchestrate.POST("/workflow/:id/stop", h.Orchestrate.StopWorkflow)
		orchestrate.GET("/workflows", h.Orchestrate.ListWorkflows)
		orchestrate.DELETE("/workflow/:id", h.Orchestrate.DeleteWorkflow)
	}

	// 第三/四阶段：注册增强路由（智能记忆、Cron、插件、导出、RAG等）
	enhanced := r.Group("/api/v2")
	enhanced.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		handler.RegisterEnhancedRoutes(enhanced, h)
	}

	return r
}