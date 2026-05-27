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
	Env          string
	ServerPort   int
	DatabasePath string
	DBEngine     string
	DBDSN        string
	JWTSecret    string
	JWTExpire    int

	// 第一阶段: 速率限制
	RateLimitRequests int
	RateLimitBurst    int

	// 通用超时配置
	AgentTimeout time.Duration

	// 第三阶段: 智能能力增强
	ConfigFile         string
	DocumentDir        string
	ExportDir          string
	CronEnabled        bool
	SandboxEnabled     bool
	WSPort             int
	TokenAlertWebhook  string
}

const defaultJWTSecret = "cat-agent-jwt-secret-2024"

// Load 从环境变量加载配置（AI模型配置已移至数据库）
func Load() (*Config, error) {
	cfg := &Config{
		Env:          getEnv("APP_ENV", "development"),
		ServerPort:   getEnvInt("SERVER_PORT", 8080),
		DatabasePath: getEnv("DATABASE_PATH", "./data/cat-agent.db"),
		DBEngine:     getEnv("DB_ENGINE", "sqlite"),
		DBDSN:        getEnv("DB_DSN", ""),
		JWTSecret:    getEnv("JWT_SECRET", ""),
		JWTExpire:    getEnvInt("JWT_EXPIRE_HOURS", 72),

		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitBurst:    getEnvInt("RATE_LIMIT_BURST", 20),

		AgentTimeout:      getEnvDuration("AGENT_TIMEOUT", 5*time.Minute),

		ConfigFile:        getEnv("CONFIG_FILE", ""),
		DocumentDir:       getEnv("DOCUMENT_DIR", "./data/documents"),
		ExportDir:         getEnv("EXPORT_DIR", "./data/exports"),
		CronEnabled:       getEnvBool("CRON_ENABLED", true),
		SandboxEnabled:    getEnvBool("SANDBOX_ENABLED", false),
		WSPort:            getEnvInt("WS_PORT", 8081),
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