package config

import (
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	Env          string
	ServerPort   int
	DatabasePath string
	JWTSecret    string
	JWTExpire    int // 小时
	OpenAIBase   string
	OpenAIKey    string
	OpenAIModel  string
	LocalModelURL string
	LocalModel   string
}

// Load 从环境变量加载配置
func Load() *Config {
	cfg := &Config{
		Env:          getEnv("APP_ENV", "development"),
		ServerPort:   getEnvInt("SERVER_PORT", 8080),
		DatabasePath: getEnv("DATABASE_PATH", "./data/eino.db"),
		JWTSecret:    getEnv("JWT_SECRET", "eino-agent-jwt-secret-2024"),
		JWTExpire:    getEnvInt("JWT_EXPIRE_HOURS", 72),
		OpenAIBase:   getEnv("OPENAI_BASE", "https://api.openai.com/v1"),
		OpenAIKey:    getEnv("OPENAI_KEY", ""),
		OpenAIModel:  getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		LocalModelURL: getEnv("LOCAL_MODEL_URL", "http://localhost:11434"),
		LocalModel:   getEnv("LOCAL_MODEL", "qwen2.5"),
	}
	return cfg
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
