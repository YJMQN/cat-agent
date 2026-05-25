package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"eino-agent/internal/api/handler"
	"eino-agent/internal/api/middleware"
	"eino-agent/internal/config"
	"eino-agent/internal/repository"
	"eino-agent/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.Load()

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

	// 设置路由
	r := setupRouter(h, cfg)

	// 启动服务
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Agent服务启动于 %s", addr)

	go func() {
		if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("服务正在关闭...")
}

func setupRouter(h *handler.Handlers, cfg *config.Config) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Agent 流式对话端点 (SSE)
	r.POST("/api/chat", h.Chat.HandleChat)
	r.GET("/api/chat/stream", h.Chat.HandleStream)

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
		admin.GET("/stats/tokens", h.Admin.TokenUsage)
		admin.GET("/stats/tools", h.Admin.ToolRanking)

		// 记忆管理
		admin.GET("/memories", h.Admin.ListMemories)
		admin.GET("/memories/:id", h.Admin.GetMemory)
		admin.PUT("/memories/:id", h.Admin.UpdateMemory)
		admin.DELETE("/memories/:id", h.Admin.DeleteMemory)

		// 用户管理
		admin.GET("/users", h.Admin.ListUsers)
		admin.PUT("/users/:id/role", h.Admin.UpdateUserRole)
	}

	// 管理前端静态文件
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	return r
}
