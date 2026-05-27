package repository

import (
	"eino-agent/internal/domain"
	"time"

	"gorm.io/gorm"
)

// Repository 仓储层接口集合
type Repository struct {
	User              UserRepository
	Agent             AgentRepository
	Tool              ToolRepository
	Session           SessionRepository
	Message           MessageRepository
	Memory            MemoryRepository
	Audit             AuditRepository
	Stats             StatsRepository
	Workflow          WorkflowRepository          // 多Agent协作编排
	WorkflowExecution WorkflowExecutionRepository // 工作流执行记录
	StepExecution     StepExecutionRepository     // 步骤执行记录
}

// New 创建仓储层实例
func New(db *gorm.DB) *Repository {
	return &Repository{
		User:              &userRepo{db: db},
		Agent:             &agentRepo{db: db},
		Tool:              &toolRepo{db: db},
		Session:           &sessionRepo{db: db},
		Message:           &messageRepo{db: db},
		Memory:            &memoryRepo{db: db},
		Audit:             &auditRepo{db: db},
		Stats:             &statsRepo{db: db},
		Workflow:          &workflowRepo{db: db},
		WorkflowExecution: &workflowExecutionRepo{db: db},
		StepExecution:     &stepExecutionRepo{db: db},
	}
}

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

// ========== 实现 ==========

type userRepo struct{ db *gorm.DB }

func (r *userRepo) Create(user *domain.User) error { return r.db.Create(user).Error }
func (r *userRepo) GetByID(id uint) (*domain.User, error) {
	var u domain.User
	err := r.db.First(&u, id).Error
	return &u, err
}
func (r *userRepo) GetByUsername(username string) (*domain.User, error) {
	var u domain.User
	err := r.db.Where("username = ?", username).First(&u).Error
	return &u, err
}
func (r *userRepo) List() ([]domain.User, error) {
	var users []domain.User
	err := r.db.Find(&users).Error
	return users, err
}
func (r *userRepo) UpdateRole(id uint, role string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", id).Update("role", role).Error
}

type agentRepo struct{ db *gorm.DB }

func (r *agentRepo) Create(agent *domain.AgentConfig) error { return r.db.Create(agent).Error }
func (r *agentRepo) GetByID(id uint) (*domain.AgentConfig, error) {
	var a domain.AgentConfig
	err := r.db.First(&a, id).Error
	return &a, err
}
func (r *agentRepo) List() ([]domain.AgentConfig, error) {
	var agents []domain.AgentConfig
	err := r.db.Find(&agents).Error
	return agents, err
}
func (r *agentRepo) Update(agent *domain.AgentConfig) error { return r.db.Save(agent).Error }
func (r *agentRepo) Delete(id uint) error                    { return r.db.Delete(&domain.AgentConfig{}, id).Error }

type toolRepo struct{ db *gorm.DB }

func (r *toolRepo) Create(tool *domain.Tool) error { return r.db.Create(tool).Error }
func (r *toolRepo) GetByID(id uint) (*domain.Tool, error) {
	var t domain.Tool
	err := r.db.First(&t, id).Error
	return &t, err
}
func (r *toolRepo) GetByName(name string) (*domain.Tool, error) {
	var t domain.Tool
	err := r.db.Where("name = ?", name).First(&t).Error
	return &t, err
}
func (r *toolRepo) List() ([]domain.Tool, error) {
	var tools []domain.Tool
	err := r.db.Find(&tools).Error
	return tools, err
}
func (r *toolRepo) ListEnabled() ([]domain.Tool, error) {
	var tools []domain.Tool
	err := r.db.Where("enabled = ?", true).Find(&tools).Error
	return tools, err
}
func (r *toolRepo) Update(tool *domain.Tool) error { return r.db.Save(tool).Error }
func (r *toolRepo) Delete(id uint) error            { return r.db.Delete(&domain.Tool{}, id).Error }

type sessionRepo struct{ db *gorm.DB }

func (r *sessionRepo) Create(s *domain.Session) error { return r.db.Create(s).Error }
func (r *sessionRepo) GetByID(id uint) (*domain.Session, error) {
	var s domain.Session
	err := r.db.First(&s, id).Error
	return &s, err
}
func (r *sessionRepo) GetByUUID(uuid string) (*domain.Session, error) {
	var s domain.Session
	err := r.db.Where("uuid = ?", uuid).First(&s).Error
	return &s, err
}
func (r *sessionRepo) List(page, size int) ([]domain.Session, int64, error) {
	var sessions []domain.Session
	var total int64
	r.db.Model(&domain.Session{}).Count(&total)
	err := r.db.Order("updated_at DESC").Offset((page - 1) * size).Limit(size).Find(&sessions).Error
	return sessions, total, err
}
func (r *sessionRepo) ListByUser(userID uint) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&sessions).Error
	return sessions, err
}
func (r *sessionRepo) Update(s *domain.Session) error { return r.db.Save(s).Error }
func (r *sessionRepo) Delete(id uint) error            { return r.db.Delete(&domain.Session{}, id).Error }
func (r *sessionRepo) GetActiveCount() int {
	var count int64
	r.db.Model(&domain.Session{}).Where("status = ?", "active").Count(&count)
	return int(count)
}

type messageRepo struct{ db *gorm.DB }

func (r *messageRepo) Create(msg *domain.Message) error { return r.db.Create(msg).Error }
func (r *messageRepo) GetBySession(sessionID uint, limit int) ([]domain.Message, error) {
	var msgs []domain.Message
	q := r.db.Where("session_id = ?", sessionID).Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&msgs).Error
	return msgs, err
}
func (r *messageRepo) CountBySession(sessionID uint) int64 {
	var count int64
	r.db.Model(&domain.Message{}).Where("session_id = ?", sessionID).Count(&count)
	return count
}

type memoryRepo struct{ db *gorm.DB }

func (r *memoryRepo) Create(mem *domain.Memory) error { return r.db.Create(mem).Error }
func (r *memoryRepo) GetByID(id uint) (*domain.Memory, error) {
	var m domain.Memory
	err := r.db.First(&m, id).Error
	return &m, err
}
func (r *memoryRepo) GetByUser(userID uint) ([]domain.Memory, error) {
	var mems []domain.Memory
	err := r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&mems).Error
	return mems, err
}
func (r *memoryRepo) GetByUserAndCategory(userID, category string) ([]domain.Memory, error) {
	var mems []domain.Memory
	err := r.db.Where("user_id = ? AND category = ?", userID, category).Find(&mems).Error
	return mems, err
}
func (r *memoryRepo) Update(mem *domain.Memory) error { return r.db.Save(mem).Error }
func (r *memoryRepo) Delete(id uint) error             { return r.db.Delete(&domain.Memory{}, id).Error }
func (r *memoryRepo) Search(userID uint, keyword string) ([]domain.Memory, error) {
	var mems []domain.Memory
	err := r.db.Where("user_id = ? AND (key LIKE ? OR content LIKE ?)",
		userID, "%"+keyword+"%", "%"+keyword+"%").Find(&mems).Error
	return mems, err
}

type auditRepo struct{ db *gorm.DB }

func (r *auditRepo) Create(log *domain.AuditLog) error { return r.db.Create(log).Error }
func (r *auditRepo) List(page, size int) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64
	r.db.Model(&domain.AuditLog{}).Count(&total)
	err := r.db.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&logs).Error
	return logs, total, err
}

type statsRepo struct{ db *gorm.DB }

func (r *statsRepo) Overview() (*domain.AdminStats, error) {
	stats := &domain.AdminStats{}

	var totalSessions, totalMessages, totalUsers, activeSessions int64
	r.db.Model(&domain.Session{}).Count(&totalSessions) // nolint
	r.db.Model(&domain.Message{}).Count(&totalMessages) // nolint
	r.db.Model(&domain.User{}).Count(&totalUsers)       // nolint
	r.db.Model(&domain.Session{}).Where("status = ?", "active").Count(&activeSessions) // nolint
	stats.TotalSessions = int(totalSessions)
	stats.TotalMessages = int(totalMessages)
	stats.TotalUsers = int(totalUsers)
	stats.ActiveSessions = int(activeSessions)

	// 计算总token
	r.db.Model(&domain.Message{}).Select("COALESCE(SUM(tokens),0)").Scan(&stats.TotalTokens) // nolint

	// 成功率 (简化: 有回复的消息占比)
	var totalMsg, repliedMsg int64
	r.db.Model(&domain.Message{}).Where("role = ?", "user").Count(&totalMsg) // nolint
	r.db.Model(&domain.Message{}).Where("role = ?", "assistant").Count(&repliedMsg) // nolint
	if totalMsg > 0 {
		stats.SuccessRate = float64(repliedMsg) / float64(totalMsg) * 100
	}
	stats.AvgLatency = 250 // 简化处理

	return stats, nil
}

func (r *statsRepo) TokenUsageByDay(days int) ([]domain.StatsTokenUsage, error) {
	var results []domain.StatsTokenUsage
	since := time.Now().AddDate(0, 0, -days)
	r.db.Model(&domain.Message{}).
		Select("DATE(created_at) as date, SUM(tokens) as total").
		Where("created_at >= ? AND tokens > 0", since).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&results)
	return results, nil
}

func (r *statsRepo) ToolRanking() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	r.db.Model(&domain.Message{}).
		Select("JSON_EXTRACT(tool_calls, '$.name') as tool_name, COUNT(*) as call_count").
		Where("tool_calls IS NOT NULL AND tool_calls != ''").
		Group("tool_name").
		Order("call_count DESC").
		Limit(10).
		Scan(&results)
	return results, nil
}

// ========== 多Agent协作编排仓储 ==========

// WorkflowRepository 工作流仓储接口
type WorkflowRepository interface {
	Create(workflow *domain.Workflow) error
	GetByID(id uint) (*domain.Workflow, error)
	List(userID uint) ([]domain.Workflow, error)
	Update(workflow *domain.Workflow) error
	Delete(id uint) error
}

// WorkflowExecutionRepository 工作流执行仓储接口
type WorkflowExecutionRepository interface {
	Create(execution *domain.WorkflowExecution) error
	GetByID(id uint) (*domain.WorkflowExecution, error)
	GetByUUID(uuid string) (*domain.WorkflowExecution, error)
	List(workflowID uint) ([]domain.WorkflowExecution, error)
	Update(execution *domain.WorkflowExecution) error
}

// StepExecutionRepository 步骤执行仓储接口
type StepExecutionRepository interface {
	Create(stepExec *domain.StepExecution) error
	GetByID(id uint) (*domain.StepExecution, error)
	ListByExecution(executionID uint) ([]domain.StepExecution, error)
	Update(stepExec *domain.StepExecution) error
}

// ========== 工作流仓储实现 ==========

type workflowRepo struct{ db *gorm.DB }

func (r *workflowRepo) Create(workflow *domain.Workflow) error { return r.db.Create(workflow).Error }
func (r *workflowRepo) GetByID(id uint) (*domain.Workflow, error) {
	var w domain.Workflow
	err := r.db.First(&w, id).Error
	return &w, err
}
func (r *workflowRepo) List(userID uint) ([]domain.Workflow, error) {
	var workflows []domain.Workflow
	err := r.db.Where("created_by = ?", userID).Order("updated_at DESC").Find(&workflows).Error
	return workflows, err
}
func (r *workflowRepo) Update(workflow *domain.Workflow) error { return r.db.Save(workflow).Error }
func (r *workflowRepo) Delete(id uint) error                    { return r.db.Delete(&domain.Workflow{}, id).Error }

type workflowExecutionRepo struct{ db *gorm.DB }

func (r *workflowExecutionRepo) Create(execution *domain.WorkflowExecution) error { return r.db.Create(execution).Error }
func (r *workflowExecutionRepo) GetByID(id uint) (*domain.WorkflowExecution, error) {
	var e domain.WorkflowExecution
	err := r.db.First(&e, id).Error
	return &e, err
}
func (r *workflowExecutionRepo) GetByUUID(uuid string) (*domain.WorkflowExecution, error) {
	var e domain.WorkflowExecution
	err := r.db.Where("uuid = ?", uuid).First(&e).Error
	return &e, err
}
func (r *workflowExecutionRepo) List(workflowID uint) ([]domain.WorkflowExecution, error) {
	var executions []domain.WorkflowExecution
	err := r.db.Where("workflow_id = ?", workflowID).Order("created_at DESC").Find(&executions).Error
	return executions, err
}
func (r *workflowExecutionRepo) Update(execution *domain.WorkflowExecution) error { return r.db.Save(execution).Error }

type stepExecutionRepo struct{ db *gorm.DB }

func (r *stepExecutionRepo) Create(stepExec *domain.StepExecution) error { return r.db.Create(stepExec).Error }
func (r *stepExecutionRepo) GetByID(id uint) (*domain.StepExecution, error) {
	var s domain.StepExecution
	err := r.db.First(&s, id).Error
	return &s, err
}
func (r *stepExecutionRepo) ListByExecution(executionID uint) ([]domain.StepExecution, error) {
	var stepExecs []domain.StepExecution
	err := r.db.Where("execution_id = ?", executionID).Order("created_at ASC").Find(&stepExecs).Error
	return stepExecs, err
}
func (r *stepExecutionRepo) Update(stepExec *domain.StepExecution) error { return r.db.Save(stepExec).Error }
