package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"cat-agent/internal/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 初始化数据库连接并执行自动迁移
func InitDB(dbPath string) (*gorm.DB, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 自动迁移所有表
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.AgentConfig{},
		&domain.Tool{},
		&domain.Session{},
		&domain.Message{},
		&domain.Memory{},
		&domain.AuditLog{},
		// 多Agent协作编排模型
		&domain.Workflow{},
		&domain.WorkflowExecution{},
		&domain.StepExecution{},
		// 第三阶段：智能能力增强
		&domain.MemoryItem{},
		&domain.CronJob{},
		&domain.CronLog{},
		&domain.Plugin{},
		// 第四阶段：体验优化
		&domain.Document{},
		&domain.DocumentChunk{},
		&domain.ExportRecord{},
		&domain.TokenBudget{},
		// 全局模型配置
		&domain.GlobalModelConfig{},
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 创建默认管理员账户
	if err := initDefaultAdmin(db); err != nil {
		return nil, fmt.Errorf("初始化默认管理员失败: %w", err)
	}

	// 注册内置工具
	initBuiltinTools(db)

	// 初始化默认模型配置
	initDefaultModelConfig(db)

	return db, nil
}

func initDefaultAdmin(db *gorm.DB) error {
	const (
		defaultAdminUsername = "admin"
		defaultAdminPassword = "admin123"
		legacyBadHash        = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	)

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(defaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var user domain.User
	if err := db.Where("username = ?", defaultAdminUsername).First(&user).Error; err == nil {
		if user.Password == legacyBadHash {
			return db.Model(&domain.User{}).Where("id = ?", user.ID).Update("password", string(hashedPwd)).Error
		}
		return nil
	}

	return db.Create(&domain.User{
		Username: defaultAdminUsername,
		Password: string(hashedPwd),
		Role:     "admin",
	}).Error
}

func initBuiltinTools(db *gorm.DB) {
	var count int64
	db.Model(&domain.Tool{}).Count(&count)
	if count == 0 {
		tools := []domain.Tool{
			{
				Name:         "weather",
				DisplayName:  "天气查询",
				Description:  "查询指定城市的当前天气信息",
				ToolType:     "builtin",
				ParamsSchema: `{"type":"object","properties":{"city":{"type":"string","description":"城市名称"}},"required":["city"]}`,
				Enabled:      true,
			},
			{
				Name:         "calculator",
				DisplayName:  "计算器",
				Description:  "执行数学表达式计算",
				ToolType:     "builtin",
				ParamsSchema: `{"type":"object","properties":{"expression":{"type":"string","description":"数学表达式"}},"required":["expression"]}`,
				Enabled:      true,
			},
			{
				Name:         "web_search",
				DisplayName:  "网络搜索",
				Description:  "搜索互联网获取信息",
				ToolType:     "builtin",
				ParamsSchema: `{"type":"object","properties":{"query":{"type":"string","description":"搜索关键词"},"limit":{"type":"integer","description":"结果数量","default":5}},"required":["query"]}`,
				Enabled:      true,
			},
			{
				Name:         "web_fetch",
				DisplayName:  "网页抓取",
				Description:  "抓取网页内容并提取文本",
				ToolType:     "builtin",
				ParamsSchema: `{"type":"object","properties":{"url":{"type":"string","description":"网页URL"},"extract_text":{"type":"boolean","description":"是否提取纯文本","default":true},"max_length":{"type":"integer","description":"最大返回长度","default":5000}},"required":["url"]}`,
				Enabled:      true,
			},
			{
				Name:         "file",
				DisplayName:  "文件操作",
				Description:  "文件系统操作：读取、写入、列表、删除",
				ToolType:     "builtin",
				ParamsSchema: `{"type":"object","properties":{"operation":{"type":"string","enum":["read","write","list","delete","create_dir"]},"path":{"type":"string","description":"文件路径"},"content":{"type":"string","description":"写入内容"},"recursive":{"type":"boolean"}},"required":["operation","path"]}`,
				Enabled:      true,
			},
		}
		for _, t := range tools {
			db.Create(&t)
		}
	}
}

// initDefaultModelConfig 初始化默认全局模型配置（所有模型提供者统一DB管理）
func initDefaultModelConfig(db *gorm.DB) {
	var count int64
	db.Model(&domain.GlobalModelConfig{}).Count(&count)
	if count == 0 {
		defaults := []domain.GlobalModelConfig{
			{
				Provider:     "openai",
				BaseURL:      "https://api.openai.com/v1",
				APIKey:       "",
				DefaultModel: "gpt-4o-mini",
				IsDefault:    true,
				Enabled:      true,
			},
			{
				Provider:     "deepseek",
				BaseURL:      "https://api.deepseek.com/v1",
				APIKey:       "",
				DefaultModel: "deepseek-chat",
				IsDefault:    false,
				Enabled:      true,
			},
			{
				Provider:     "openrouter",
				BaseURL:      "https://openrouter.ai/api/v1",
				APIKey:       "",
				DefaultModel: "openai/gpt-4o-mini",
				IsDefault:    false,
				Enabled:      true,
			},
			{
				Provider:     "modelscope",
				BaseURL:      "https://api-inference.modelscope.cn/v1",
				APIKey:       "",
				DefaultModel: "Qwen/Qwen2.5-7B-Instruct",
				IsDefault:    false,
				Enabled:      true,
			},
			{
				Provider:     "local",
				BaseURL:      "http://localhost:11434",
				APIKey:       "",
				DefaultModel: "qwen2.5",
				IsDefault:    false,
				Enabled:      true,
			},
		}
		for _, c := range defaults {
			db.Create(&c)
		}
	}
}