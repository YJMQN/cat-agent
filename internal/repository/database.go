package repository

import (
	"eino-agent/internal/domain"
	"fmt"
	"os"
	"path/filepath"

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
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 创建默认管理员账户
	initDefaultAdmin(db)

	// 注册内置工具
	initBuiltinTools(db)

	return db, nil
}

func initDefaultAdmin(db *gorm.DB) {
	var count int64
	db.Model(&domain.User{}).Count(&count)
	if count == 0 {
		// 默认密码 admin123, 实际部署应修改
		db.Create(&domain.User{
			Username: "admin",
			Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // bcrypt of admin123
			Role:     "admin",
		})
	}
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
		}
		for _, t := range tools {
			db.Create(&t)
		}
	}
}
