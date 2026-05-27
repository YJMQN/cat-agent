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
	JWTExpire     int // 小时
	OpenAIBase    string
	OpenAIKey     string
	OpenAIModel   string
	LocalModelURL string
	LocalModel    string
	// 第一阶段新增: 速率限制配置
	RateLimitRequests int           // 每秒允许的请求数
	RateLimitBurst    int           // 突发请求上限
	// 第二阶段新增: HTTP客户端配置
	HTTPTimeout       time.Duration // HTTP请求超时时间
	HTTPMaxRetries    int           // 最大重试次数
	HTTPRetryDelay    time.Duration // 重试间隔
	// 多agent协作编排配置
	MaxConcurrentAgents int         // 最大并发Agent数
	AgentTimeout        time.Duration // Agent执行超时
}

// 默认JWT密钥(仅用于检测)
const defaultJWTSecret = "eino-agent-jwt-secret-2024"

// Load 从环境变量加载配置
func Load() (*Config, error) {
	cfg := &Config{
		Env:          getEnv("APP_ENV", "development"),
		ServerPort:   getEnvInt("SERVER_PORT", 8080),
		DatabasePath: getEnv("DATABASE_PATH", "./data/eino.db"),
		JWTSecret:    getEnv("JWT_SECRET", ""),
		JWTExpire:    getEnvInt("JWT_EXPIRE_HOURS", 72),
		OpenAIBase:   getEnv("OPENAI_BASE", "https://api.openai.com/v1"),
		OpenAIKey:    getEnv("OPENAI_KEY", ""),
		OpenAIModel:  getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		LocalModelURL: getEnv("LOCAL_MODEL_URL", "http://localhost:11434"),
		LocalModel:   getEnv("LOCAL_MODEL", "qwen2.5"),
		// 第一阶段: 速率限制配置
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitBurst:    getEnvInt("RATE_LIMIT_BURST", 20),
		// 第二阶段: HTTP客户端配置
		HTTPTimeout:       getEnvDuration("HTTP_TIMEOUT", 30*time.Second),
		HTTPMaxRetries:    getEnvInt("HTTP_MAX_RETRIES", 3),
		HTTPRetryDelay:    getEnvDuration("HTTP_RETRY_DELAY", 1*time.Second),
		// 多agent协作编排
		MaxConcurrentAgents: getEnvInt("MAX_CONCURRENT_AGENTS", 10),
		AgentTimeout:        getEnvDuration("AGENT_TIMEOUT", 5*time.Minute),
	}

	// 第一阶段安全加固: JWT密钥强制从环境变量读取
	// 生产环境必须设置JWT_SECRET环境变量
	if cfg.JWTSecret == "" || cfg.JWTSecret == defaultJWTSecret {
		if cfg.Env == "production" {
			return nil, errors.New("生产环境必须设置JWT_SECRET环境变量，不允许使用默认密钥")
		}
		// 开发环境允许默认密钥但打印警告
		fmt.Println("⚠️  警告: 使用默认JWT密钥，请设置JWT_SECRET环境变量以确保安全")
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

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		// 尝试解析为秒数
		if n, err := strconv.Atoi(val); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}