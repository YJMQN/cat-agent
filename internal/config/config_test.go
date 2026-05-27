package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// 测试默认配置加载
	cfg, err := Load()
	if err != nil {
		// 在开发环境下应该允许默认密钥
		t.Logf("Load返回错误(可能是生产环境检测): %v", err)
	}
	if cfg != nil {
		if cfg.ServerPort == 0 {
			t.Error("ServerPort不应为0")
		}
		if cfg.RateLimitRequests == 0 {
			t.Error("RateLimitRequests不应为0")
		}
	}
}

func TestJWTSecretRequirement(t *testing.T) {
	// 测试生产环境JWT密钥要求
	os.Setenv("APP_ENV", "production")
	os.Setenv("JWT_SECRET", "") // 清除JWT密钥

	cfg, err := Load()
	if err == nil {
		t.Error("生产环境没有JWT_SECRET应该返回错误")
	}

	// 设置JWT密钥后应该成功
	os.Setenv("JWT_SECRET", "my-secret-key-12345")
	cfg, err = Load()
	if err != nil {
		t.Errorf("设置了JWT_SECRET后应该成功: %v", err)
	}
	if cfg != nil && cfg.JWTSecret != "my-secret-key-12345" {
		t.Errorf("JWT密钥不匹配: got %s", cfg.JWTSecret)
	}

	// 清理
	os.Unsetenv("APP_ENV")
	os.Unsetenv("JWT_SECRET")
}

func TestDefaultJWTSecretWarning(t *testing.T) {
	// 测试开发环境使用默认密钥
	os.Setenv("APP_ENV", "development")
	os.Setenv("JWT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Errorf("开发环境应该允许默认密钥: %v", err)
	}
	if cfg != nil && cfg.JWTSecret != defaultJWTSecret {
		t.Errorf("应该使用默认密钥: got %s", cfg.JWTSecret)
	}

	os.Unsetenv("APP_ENV")
	os.Unsetenv("JWT_SECRET")
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	result := getEnv("TEST_KEY", "default")
	if result != "test_value" {
		t.Errorf("getEnv失败: got %s, want test_value", result)
	}

	result = getEnv("NONEXISTENT_KEY", "default")
	if result != "default" {
		t.Errorf("getEnv默认值失败: got %s, want default", result)
	}

	os.Unsetenv("TEST_KEY")
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	result := getEnvInt("TEST_INT", 0)
	if result != 42 {
		t.Errorf("getEnvInt失败: got %d, want 42", result)
	}

	os.Setenv("TEST_INT_INVALID", "not_a_number")
	result = getEnvInt("TEST_INT_INVALID", 10)
	if result != 10 {
		t.Errorf("getEnvInt无效值应返回默认: got %d, want 10", result)
	}

	result = getEnvInt("NONEXISTENT_INT", 5)
	if result != 5 {
		t.Errorf("getEnvInt默认值失败: got %d, want 5", result)
	}

	os.Unsetenv("TEST_INT")
	os.Unsetenv("TEST_INT_INVALID")
}

func TestConfigDefaults(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	os.Setenv("JWT_SECRET", "test-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load失败: %v", err)
	}

	// 检查默认值
	expectedDefaults := map[string]interface{}{
		"ServerPort":         8080,
		"RateLimitRequests":  100,
		"RateLimitBurst":     20,
		"HTTPMaxRetries":     3,
		"MaxConcurrentAgents": 10,
	}

	for key, expected := range expectedDefaults {
		switch key {
		case "ServerPort":
			if cfg.ServerPort != expected.(int) {
				t.Errorf("%s 默认值不匹配: got %d, want %d", key, cfg.ServerPort, expected)
			}
		case "RateLimitRequests":
			if cfg.RateLimitRequests != expected.(int) {
				t.Errorf("%s 默认值不匹配: got %d, want %d", key, cfg.RateLimitRequests, expected)
			}
		case "RateLimitBurst":
			if cfg.RateLimitBurst != expected.(int) {
				t.Errorf("%s 默认值不匹配: got %d, want %d", key, cfg.RateLimitBurst, expected)
			}
		case "HTTPMaxRetries":
			if cfg.HTTPMaxRetries != expected.(int) {
				t.Errorf("%s 默认值不匹配: got %d, want %d", key, cfg.HTTPMaxRetries, expected)
			}
		case "MaxConcurrentAgents":
			if cfg.MaxConcurrentAgents != expected.(int) {
				t.Errorf("%s 默认值不匹配: got %d, want %d", key, cfg.MaxConcurrentAgents, expected)
			}
		}
	}

	os.Unsetenv("APP_ENV")
	os.Unsetenv("JWT_SECRET")
}