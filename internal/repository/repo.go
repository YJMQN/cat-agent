package repository

import (
	"eino-agent/internal/domain"
	"time"

	"gorm.io/gorm"
)

// Repository 仓储层接口集合
type Repository struct {
	User    UserRepository
	Agent   AgentRepository
	Tool    ToolRepository
	Session SessionRepository
	Message MessageRepository
	Memory  MemoryRepository
	Audit   AuditRepository
	Stats   StatsRepository

	// 第三/四阶段新增仓储
	MemoryItem    MemoryItemRepository
	CronJob       CronJobRepository
	CronLog       CronLogRepository
	Plugin        PluginRepository
	Document      DocumentRepository
	DocumentChunk DocumentChunkRepository
	ExportRecord  ExportRecordRepository
	TokenBudget   TokenBudgetRepository
	ModelConfig   GlobalModelConfigRepository
	
	// 多Agent协作编排仓储
	Workflow         WorkflowRepository
	WorkflowExecution WorkflowExecutionRepository
	StepExecution    StepExecutionRepository

	db *gorm.DB
}

// New 创建仓储层实例
func New(db *gorm.DB) *Repository {
	return &Repository{
		User:    &userRepo{db: db},
		Agent:   &agentRepo{db: db},
		Tool:    &toolRepo{db: db},
		Session: &sessionRepo{db: db},
		Message: &messageRepo{db: db},
		Memory:  &memoryRepo{db: db},
		Audit:   &auditRepo{db: db},
		Stats:   &statsRepo{db: db},

		MemoryItem:    &memoryItemRepo{db: db},
		CronJob:       &cronJobRepo{db: db},
		CronLog:       &cronLogRepo{db: db},
		Plugin:        &pluginRepo{db: db},
		Document:      &documentRepo{db: db},
		DocumentChunk: &documentChunkRepo{db: db},
		ExportRecord:  &exportRecordRepo{db: db},
		TokenBudget:   &tokenBudgetRepo{db: db},
		ModelConfig:    &modelConfigRepo{db: db},

		Workflow:          &workflowRepo{db: db},
		WorkflowExecution: &workflowExecutionRepo{db: db},
		StepExecution:     &stepExecutionRepo{db: db},

		db: db,
	}
}

// DB 获取底层数据库连接（用于审计日志等高级操作）
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// ========== 原有接口定义 ==========

// UserRepository 用户仓储
type UserRepository interface {
	Create(user *domain.User) error
	GetByID(id uint) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	List() ([]domain.User, error)
	UpdateRole(id uint, role string) error
}

// AgentRepository Agent配置仓储
type AgentRepository interface {
	Create(agent *domain.AgentConfig) error
	GetByID(id uint) (*domain.AgentConfig, error)
	List() ([]domain.AgentConfig, error)
	Update(agent *domain.AgentConfig) error
	Delete(id uint) error
}

// ToolRepository 工具仓储
type ToolRepository interface {
	Create(tool *domain.Tool) error
	GetByID(id uint) (*domain.Tool, error)
	GetByName(name string) (*domain.Tool, error)
	List() ([]domain.Tool, error)
	ListEnabled() ([]domain.Tool, error)
	Update(tool *domain.Tool) error
	Delete(id uint) error
}

// SessionRepository 会话仓储
type SessionRepository interface {
	Create(session *domain.Session) error
	GetByID(id uint) (*domain.Session, error)
	GetByUUID(uuid string) (*domain.Session, error)
	List(page, size int) ([]domain.Session, int64, error)
	ListByUser(userID uint) ([]domain.Session, error)
	Update(session *domain.Session) error
	Delete(id uint) error
	GetActiveCount() int
}

// MessageRepository 消息仓储
type MessageRepository interface {
	Create(msg *domain.Message) error
	GetBySession(sessionID uint, limit int) ([]domain.Message, error)
	CountBySession(sessionID uint) int64
}

// MemoryRepository 记忆仓储
type MemoryRepository interface {
	Create(mem *domain.Memory) error
	GetByID(id uint) (*domain.Memory, error)
	GetByUser(userID uint) ([]domain.Memory, error)
	GetByUserAndCategory(userID, category string) ([]domain.Memory, error)
	Update(mem *domain.Memory) error
	Delete(id uint) error
	Search(userID uint, keyword string) ([]domain.Memory, error)
}

// AuditRepository 审计日志仓储
type AuditRepository interface {
	Create(log *domain.AuditLog) error
	List(page, size int) ([]domain.AuditLog, int64, error)
}

// StatsRepository 统计仓储
type StatsRepository interface {
	Overview() (*domain.AdminStats, error)
	TokenUsageByDay(days int) ([]domain.StatsTokenUsage, error)
	ToolRanking() ([]map[string]interface{}, error)
}

// ========== 第三/四阶段新增接口定义 ==========

// MemoryItemRepository 增强记忆仓储
type MemoryItemRepository interface {
	Create(m *domain.MemoryItem) error
	Search(userID uint, query string, limit int) ([]domain.MemoryItem, error)
	Delete(id uint) error
}

// CronJobRepository 定时任务仓储
type CronJobRepository interface {
	Create(j *domain.CronJob) error
	Update(j *domain.CronJob) error
	GetByID(id uint) (*domain.CronJob, error)
	ListActive() ([]domain.CronJob, error)
	ListByUser(userID uint) ([]domain.CronJob, error)
	Delete(id uint) error
}

// CronLogRepository 定时任务日志仓储
type CronLogRepository interface {
	Create(l *domain.CronLog) error
	Update(l *domain.CronLog) error
	ListByJob(jobID uint, limit int) ([]domain.CronLog, error)
}

// PluginRepository 插件仓储
type PluginRepository interface {
	Create(p *domain.Plugin) error
	GetByID(id uint) (*domain.Plugin, error)
	ListEnabled() ([]domain.Plugin, error)
	ListByUser(userID uint) ([]domain.Plugin, error)
	Update(p *domain.Plugin) error
	Delete(id uint) error
}

// DocumentRepository 文档仓储
type DocumentRepository interface {
	Create(d *domain.Document) error
	GetByID(id uint) (*domain.Document, error)
	ListByUser(userID uint) ([]domain.Document, error)
	Update(d *domain.Document) error
	Delete(id uint) error
}

// DocumentChunkRepository 文档分块仓储
type DocumentChunkRepository interface {
	Create(dc *domain.DocumentChunk) error
	SearchByKeyword(userID uint, query string, limit int) []domain.DocumentChunk
	DeleteByDocument(docID uint) error
}

// ExportRecordRepository 导出记录仓储
type ExportRecordRepository interface {
	Create(e *domain.ExportRecord) error
	ListByUser(userID uint) ([]domain.ExportRecord, error)
}

// TokenBudgetRepository Token预算仓储
type TokenBudgetRepository interface {
	Create(b *domain.TokenBudget) error
	GetByUser(userID uint) (*domain.TokenBudget, error)
	Update(b *domain.TokenBudget) error
}

// GlobalModelConfigRepository 全局模型配置仓储
type GlobalModelConfigRepository interface {
	List() ([]domain.GlobalModelConfig, error)
	GetByProvider(provider string) (*domain.GlobalModelConfig, error)
	GetDefault() (*domain.GlobalModelConfig, error)
	Create(cfg *domain.GlobalModelConfig) error
	Update(cfg *domain.GlobalModelConfig) error
	Delete(id uint) error
}

// ========== 多Agent协作编排仓储接口 ==========

type WorkflowRepository interface {
	Create(w *domain.Workflow) error
	GetByID(id uint) (*domain.Workflow, error)
	List(userID uint) ([]domain.Workflow, error)
	Update(w *domain.Workflow) error
	Delete(id uint) error
}

type WorkflowExecutionRepository interface {
	Create(e *domain.WorkflowExecution) error
	GetByID(id uint) (*domain.WorkflowExecution, error)
	GetByWorkflowID(workflowID uint) ([]domain.WorkflowExecution, error)
	Update(e *domain.WorkflowExecution) error
}

type StepExecutionRepository interface {
	Create(s *domain.StepExecution) error
	Update(s *domain.StepExecution) error
	ListByExecution(executionID uint) ([]domain.StepExecution, error)
}

// ========== 原有接口实现 ==========

type userRepo struct{ db *gorm.DB }
func (r *userRepo) Create(user *domain.User) error                       { return r.db.Create(user).Error }
func (r *userRepo) GetByID(id uint) (*domain.User, error)                { var u domain.User; err := r.db.First(&u, id).Error; return &u, err }
func (r *userRepo) GetByUsername(username string) (*domain.User, error)  { var u domain.User; err := r.db.Where("username = ?", username).First(&u).Error; return &u, err }
func (r *userRepo) List() ([]domain.User, error)                         { var users []domain.User; err := r.db.Find(&users).Error; return users, err }
func (r *userRepo) UpdateRole(id uint, role string) error                { return r.db.Model(&domain.User{}).Where("id = ?", id).Update("role", role).Error }

type agentRepo struct{ db *gorm.DB }
func (r *agentRepo) Create(agent *domain.AgentConfig) error { return r.db.Create(agent).Error }
func (r *agentRepo) GetByID(id uint) (*domain.AgentConfig, error) { var a domain.AgentConfig; err := r.db.First(&a, id).Error; return &a, err }
func (r *agentRepo) List() ([]domain.AgentConfig, error) { var agents []domain.AgentConfig; err := r.db.Find(&agents).Error; return agents, err }
func (r *agentRepo) Update(agent *domain.AgentConfig) error { return r.db.Save(agent).Error }
func (r *agentRepo) Delete(id uint) error { return r.db.Delete(&domain.AgentConfig{}, id).Error }

type toolRepo struct{ db *gorm.DB }
func (r *toolRepo) Create(tool *domain.Tool) error { return r.db.Create(tool).Error }
func (r *toolRepo) GetByID(id uint) (*domain.Tool, error) { var t domain.Tool; err := r.db.First(&t, id).Error; return &t, err }
func (r *toolRepo) GetByName(name string) (*domain.Tool, error) { var t domain.Tool; err := r.db.Where("name = ?", name).First(&t).Error; return &t, err }
func (r *toolRepo) List() ([]domain.Tool, error) { var tools []domain.Tool; err := r.db.Find(&tools).Error; return tools, err }
func (r *toolRepo) ListEnabled() ([]domain.Tool, error) { var tools []domain.Tool; err := r.db.Where("enabled = ?", true).Find(&tools).Error; return tools, err }
func (r *toolRepo) Update(tool *domain.Tool) error { return r.db.Save(tool).Error }
func (r *toolRepo) Delete(id uint) error { return r.db.Delete(&domain.Tool{}, id).Error }

type sessionRepo struct{ db *gorm.DB }
func (r *sessionRepo) Create(s *domain.Session) error { return r.db.Create(s).Error }
func (r *sessionRepo) GetByID(id uint) (*domain.Session, error) { var s domain.Session; err := r.db.First(&s, id).Error; return &s, err }
func (r *sessionRepo) GetByUUID(uuid string) (*domain.Session, error) { var s domain.Session; err := r.db.Where("uuid = ?", uuid).First(&s).Error; return &s, err }
func (r *sessionRepo) List(page, size int) ([]domain.Session, int64, error) { 
	var sessions []domain.Session
	var total int64
	r.db.Model(&domain.Session{}).Count(&total)
	offset := (page - 1) * size
	if offset < 0 { offset = 0 }
	err := r.db.Order("created_at DESC").Offset(offset).Limit(size).Find(&sessions).Error
	return sessions, total, err
}
func (r *sessionRepo) ListByUser(userID uint) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}
func (r *sessionRepo) Update(s *domain.Session) error { return r.db.Save(s).Error }
func (r *sessionRepo) Delete(id uint) error { return r.db.Delete(&domain.Session{}, id).Error }
func (r *sessionRepo) GetActiveCount() int { var count int64; r.db.Model(&domain.Session{}).Where("status = ?", "active").Count(&count); return int(count) }

type messageRepo struct{ db *gorm.DB }
func (r *messageRepo) Create(msg *domain.Message) error { return r.db.Create(msg).Error }
func (r *messageRepo) GetBySession(sessionID uint, limit int) ([]domain.Message, error) {
	var msgs []domain.Message
	err := r.db.Where("session_id = ?", sessionID).Order("created_at ASC").Limit(limit).Find(&msgs).Error
	return msgs, err
}
func (r *messageRepo) CountBySession(sessionID uint) int64 {
	var count int64
	r.db.Model(&domain.Message{}).Where("session_id = ?", sessionID).Count(&count)
	return count
}

type memoryRepo struct{ db *gorm.DB }
func (r *memoryRepo) Create(m *domain.Memory) error { return r.db.Create(m).Error }
func (r *memoryRepo) GetByID(id uint) (*domain.Memory, error) { var m domain.Memory; err := r.db.First(&m, id).Error; return &m, err }
func (r *memoryRepo) GetByUser(userID uint) ([]domain.Memory, error) {
	var mems []domain.Memory
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&mems).Error
	return mems, err
}
func (r *memoryRepo) GetByUserAndCategory(userID, category string) ([]domain.Memory, error) {
	return nil, nil
}
func (r *memoryRepo) Update(m *domain.Memory) error { return r.db.Save(m).Error }
func (r *memoryRepo) Delete(id uint) error { return r.db.Delete(&domain.Memory{}, id).Error }
func (r *memoryRepo) Search(userID uint, keyword string) ([]domain.Memory, error) {
	var mems []domain.Memory
	err := r.db.Where("user_id = ? AND content LIKE ?", userID, "%"+keyword+"%").Find(&mems).Error
	return mems, err
}

type auditRepo struct{ db *gorm.DB }
func (r *auditRepo) Create(log *domain.AuditLog) error { return r.db.Create(log).Error }
func (r *auditRepo) List(page, size int) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64
	r.db.Model(&domain.AuditLog{}).Count(&total)
	offset := (page - 1) * size
	if offset < 0 { offset = 0 }
	err := r.db.Order("created_at DESC").Offset(offset).Limit(size).Find(&logs).Error
	return logs, total, err
}

type statsRepo struct{ db *gorm.DB }
func (r *statsRepo) Overview() (*domain.AdminStats, error) {
	var totalSessions, totalMessages, totalUsers int64
	r.db.Model(&domain.Session{}).Count(&totalSessions)
	r.db.Model(&domain.Message{}).Count(&totalMessages)
	r.db.Model(&domain.User{}).Count(&totalUsers)
	stats := &domain.AdminStats{
		TotalSessions: int(totalSessions),
		TotalMessages: int(totalMessages),
		TotalUsers:    int(totalUsers),
		SuccessRate:   0.95,
	}
	return stats, nil
}
func (r *statsRepo) TokenUsageByDay(days int) ([]domain.StatsTokenUsage, error) {
	return []domain.StatsTokenUsage{}, nil
}
func (r *statsRepo) ToolRanking() ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// ========== 第三/四阶段新增实现 ==========

type memoryItemRepo struct{ db *gorm.DB }
func (r *memoryItemRepo) Create(m *domain.MemoryItem) error { return r.db.Create(m).Error }
func (r *memoryItemRepo) Search(userID uint, query string, limit int) ([]domain.MemoryItem, error) {
	var items []domain.MemoryItem
	like := "%" + query + "%"
	err := r.db.Where("user_id = ? AND (content LIKE ? OR keywords LIKE ?) AND level = 'long'", userID, like, like).
		Order("importance DESC, access_count DESC").Limit(limit).Find(&items).Error
	return items, err
}
func (r *memoryItemRepo) Delete(id uint) error { return r.db.Delete(&domain.MemoryItem{}, id).Error }

type cronJobRepo struct{ db *gorm.DB }
func (r *cronJobRepo) Create(j *domain.CronJob) error { return r.db.Create(j).Error }
func (r *cronJobRepo) Update(j *domain.CronJob) error { return r.db.Save(j).Error }
func (r *cronJobRepo) GetByID(id uint) (*domain.CronJob, error) { var j domain.CronJob; err := r.db.First(&j, id).Error; return &j, err }
func (r *cronJobRepo) ListActive() ([]domain.CronJob, error) { var jobs []domain.CronJob; err := r.db.Where("status = ?", "active").Find(&jobs).Error; return jobs, err }
func (r *cronJobRepo) ListByUser(userID uint) ([]domain.CronJob, error) { var jobs []domain.CronJob; err := r.db.Where("user_id = ?", userID).Find(&jobs).Error; return jobs, err }
func (r *cronJobRepo) Delete(id uint) error { return r.db.Delete(&domain.CronJob{}, id).Error }

type cronLogRepo struct{ db *gorm.DB }
func (r *cronLogRepo) Create(l *domain.CronLog) error { return r.db.Create(l).Error }
func (r *cronLogRepo) Update(l *domain.CronLog) error { return r.db.Save(l).Error }
func (r *cronLogRepo) ListByJob(jobID uint, limit int) ([]domain.CronLog, error) { var logs []domain.CronLog; err := r.db.Where("job_id = ?", jobID).Order("run_at DESC").Limit(limit).Find(&logs).Error; return logs, err }

type pluginRepo struct{ db *gorm.DB }
func (r *pluginRepo) Create(p *domain.Plugin) error { return r.db.Create(p).Error }
func (r *pluginRepo) GetByID(id uint) (*domain.Plugin, error) { var p domain.Plugin; err := r.db.First(&p, id).Error; return &p, err }
func (r *pluginRepo) ListEnabled() ([]domain.Plugin, error) { var plugins []domain.Plugin; err := r.db.Where("enabled = ?", true).Find(&plugins).Error; return plugins, err }
func (r *pluginRepo) ListByUser(userID uint) ([]domain.Plugin, error) { var plugins []domain.Plugin; err := r.db.Where("created_by = ?", userID).Find(&plugins).Error; return plugins, err }
func (r *pluginRepo) Update(p *domain.Plugin) error { return r.db.Save(p).Error }
func (r *pluginRepo) Delete(id uint) error { return r.db.Delete(&domain.Plugin{}, id).Error }

type documentRepo struct{ db *gorm.DB }
func (r *documentRepo) Create(d *domain.Document) error { return r.db.Create(d).Error }
func (r *documentRepo) GetByID(id uint) (*domain.Document, error) { var d domain.Document; err := r.db.First(&d, id).Error; return &d, err }
func (r *documentRepo) ListByUser(userID uint) ([]domain.Document, error) { var docs []domain.Document; err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&docs).Error; return docs, err }
func (r *documentRepo) Update(d *domain.Document) error { return r.db.Save(d).Error }
func (r *documentRepo) Delete(id uint) error { return r.db.Delete(&domain.Document{}, id).Error }

type documentChunkRepo struct{ db *gorm.DB }
func (r *documentChunkRepo) Create(dc *domain.DocumentChunk) error { return r.db.Create(dc).Error }
func (r *documentChunkRepo) SearchByKeyword(userID uint, query string, limit int) []domain.DocumentChunk {
	var chunks []domain.DocumentChunk
	r.db.Where("user_id = ? AND content LIKE ?", userID, "%"+query+"%").Limit(limit).Find(&chunks)
	return chunks
}
func (r *documentChunkRepo) DeleteByDocument(docID uint) error { return r.db.Where("document_id = ?", docID).Delete(&domain.DocumentChunk{}).Error }

type exportRecordRepo struct{ db *gorm.DB }
func (r *exportRecordRepo) Create(e *domain.ExportRecord) error { return r.db.Create(e).Error }
func (r *exportRecordRepo) ListByUser(userID uint) ([]domain.ExportRecord, error) { var records []domain.ExportRecord; err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&records).Error; return records, err }

type tokenBudgetRepo struct{ db *gorm.DB }
func (r *tokenBudgetRepo) Create(b *domain.TokenBudget) error { return r.db.Create(b).Error }
func (r *tokenBudgetRepo) GetByUser(userID uint) (*domain.TokenBudget, error) { var b domain.TokenBudget; err := r.db.Where("user_id = ?", userID).First(&b).Error; return &b, err }
func (r *tokenBudgetRepo) Update(b *domain.TokenBudget) error { return r.db.Save(b).Error }

// ========== GlobalModelConfig Repo ==========

type modelConfigRepo struct{ db *gorm.DB }
func (r *modelConfigRepo) List() ([]domain.GlobalModelConfig, error) {
	var cfgs []domain.GlobalModelConfig
	err := r.db.Find(&cfgs).Error
	return cfgs, err
}
func (r *modelConfigRepo) GetByProvider(provider string) (*domain.GlobalModelConfig, error) {
	var cfg domain.GlobalModelConfig
	err := r.db.Where("provider = ?", provider).First(&cfg).Error
	return &cfg, err
}
func (r *modelConfigRepo) GetDefault() (*domain.GlobalModelConfig, error) {
	var cfg domain.GlobalModelConfig
	err := r.db.Where("is_default = ?", true).First(&cfg).Error
	return &cfg, err
}
func (r *modelConfigRepo) Create(cfg *domain.GlobalModelConfig) error { return r.db.Create(cfg).Error }
func (r *modelConfigRepo) Update(cfg *domain.GlobalModelConfig) error { return r.db.Save(cfg).Error }
func (r *modelConfigRepo) Delete(id uint) error { return r.db.Delete(&domain.GlobalModelConfig{}, id).Error }

// ========== 多Agent协作编排实现 ==========

type workflowRepo struct{ db *gorm.DB }
func (r *workflowRepo) Create(w *domain.Workflow) error { return r.db.Create(w).Error }
func (r *workflowRepo) GetByID(id uint) (*domain.Workflow, error) { var w domain.Workflow; err := r.db.First(&w, id).Error; return &w, err }
func (r *workflowRepo) List(userID uint) ([]domain.Workflow, error) { var ws []domain.Workflow; err := r.db.Where("user_id = ?", userID).Find(&ws).Error; return ws, err }
func (r *workflowRepo) Update(w *domain.Workflow) error { return r.db.Save(w).Error }
func (r *workflowRepo) Delete(id uint) error { return r.db.Delete(&domain.Workflow{}, id).Error }

type workflowExecutionRepo struct{ db *gorm.DB }
func (r *workflowExecutionRepo) Create(e *domain.WorkflowExecution) error { return r.db.Create(e).Error }
func (r *workflowExecutionRepo) GetByID(id uint) (*domain.WorkflowExecution, error) { var e domain.WorkflowExecution; err := r.db.First(&e, id).Error; return &e, err }
func (r *workflowExecutionRepo) GetByWorkflowID(workflowID uint) ([]domain.WorkflowExecution, error) { var execs []domain.WorkflowExecution; err := r.db.Where("workflow_id = ?", workflowID).Find(&execs).Error; return execs, err }
func (r *workflowExecutionRepo) Update(e *domain.WorkflowExecution) error { return r.db.Save(e).Error }

type stepExecutionRepo struct{ db *gorm.DB }
func (r *stepExecutionRepo) Create(s *domain.StepExecution) error { return r.db.Create(s).Error }
func (r *stepExecutionRepo) Update(s *domain.StepExecution) error { return r.db.Save(s).Error }
func (r *stepExecutionRepo) ListByExecution(executionID uint) ([]domain.StepExecution, error) { var steps []domain.StepExecution; err := r.db.Where("execution_id = ?", executionID).Find(&steps).Error; return steps, err }

// 抑制未使用导入警告
var _ = time.Now