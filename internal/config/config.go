package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	Env           string
	ServerPort    int
	DatabasePath  string
	JWTSecret     string
	JWTExpire     int
	OpenAIBase    string
	OpenAIKey     string
	OpenAIModel   string
	LocalModelURL string
	LocalModel    string

	// 第一阶段: 速率限制
	RateLimitRequests int
	RateLimitBurst    int

	// 第二阶段: HTTP客户端
	HTTPTimeout       time.Duration
	HTTPMaxRetries    int
	HTTPRetryDelay    time.Duration
	MaxConcurrentAgents int
	AgentTimeout      time.Duration

	// 第三阶段: 智能能力增强
	MemoryEmbeddingModel string // 记忆嵌入模型
	CronEnabled          bool   // Cron调度器启用
	SandboxEnabled       bool   // 沙箱启用
	WSUrl                string // WebSocket监听地址
	WSPort               int    // WebSocket端口

	// 第四阶段: 体验优化
	ConfigFile         string // YAML/TOML配置文件路径
	DocumentDir        string // 文档存储目录
	ExportDir          string // 导出文件目录
	DBEngine           string // 数据库引擎: sqlite, postgres
	DBDSN              string // 数据库DSN

	// 费用相关
	TokenAlertWebhook string // Token告警Webhook
}

const defaultJWTSecret = "eino-agent-jwt-secret-2024"

// Load 从环境变量加载配置
func Load() (*Config, error) {
	cfg := &Config{
		Env:           getEnv("APP_ENV", "development"),
		ServerPort:    getEnvInt("SERVER_PORT", 8080),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/eino.db"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpire:     getEnvInt("JWT_EXPIRE_HOURS", 72),
		OpenAIBase:    getEnv("OPENAI_BASE", "https://api.openai.com/v1"),
		OpenAIKey:     getEnv("OPENAI_KEY", ""),
		OpenAIModel:   getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		LocalModelURL: getEnv("LOCAL_MODEL_URL", "http://localhost:11434"),
		LocalModel:    getEnv("LOCAL_MODEL", "qwen2.5"),

		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitBurst:    getEnvInt("RATE_LIMIT_BURST", 20),

		HTTPTimeout:       getEnvDuration("HTTP_TIMEOUT", 30*time.Second),
		HTTPMaxRetries:    getEnvInt("HTTP_MAX_RETRIES", 3),
		HTTPRetryDelay:    getEnvDuration("HTTP_RETRY_DELAY", 1*time.Second),
		MaxConcurrentAgents: getEnvInt("MAX_CONCURRENT_AGENTS", 10),
		AgentTimeout:      getEnvDuration("AGENT_TIMEOUT", 5*time.Minute),

		// 第三阶段
		MemoryEmbeddingModel: getEnv("MEMORY_EMBEDDING_MODEL", "local"),
		CronEnabled:          getEnvBool("CRON_ENABLED", true),
		SandboxEnabled:       getEnvBool("SANDBOX_ENABLED", false),
		WSUrl:                getEnv("WS_URL", "127.0.0.1"),
		WSPort:               getEnvInt("WS_PORT", 8081),

		// 第四阶段
		ConfigFile:         getEnv("CONFIG_FILE", ""),
		DocumentDir:        getEnv("DOCUMENT_DIR", "./data/documents"),
		ExportDir:          getEnv("EXPORT_DIR", "./data/exports"),
		DBEngine:           getEnv("DB_ENGINE", "sqlite"),
		DBDSN:              getEnv("DB_DSN", ""),
		TokenAlertWebhook: getEnv("TOKEN_ALERT_WEBHOOK", ""),
	}

	// JWT安全检查
	if cfg.JWTSecret == "" || cfg.JWTSecret == defaultJWTSecret {
		if cfg.Env == "production" {
			return nil, errors.New("生产环境必须设置JWT_SECRET环境变量")
		}
		fmt.Println("⚠️  警告: 使用默认JWT密钥，请设置JWT_SECRET环境变量")
		cfg.JWTSecret = defaultJWTSecret
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		if n, err := strconv.Atoi(val); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}