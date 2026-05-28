package repository

import (
	"cat-agent/internal/domain"

	"gorm.io/gorm"
)

// ========== 功能4：Agent评估框架Repository定义 ==========

// AgentMetricsRepository Agent指标仓储接口
type AgentMetricsRepository interface {
	Create(metrics *domain.AgentMetrics) error
	GetByAgentAndDate(agentID uint, date string) (*domain.AgentMetrics, error)
	ListByAgent(agentID uint, limit int) ([]domain.AgentMetrics, error)
	Update(metrics *domain.AgentMetrics) error
	GetLatestByAgent(agentID uint) (*domain.AgentMetrics, error)
}

// EvaluationResultRepository 评估结果仓储接口
type EvaluationResultRepository interface {
	Create(result *domain.EvaluationResult) error
	GetByID(id uint) (*domain.EvaluationResult, error)
	GetBySession(sessionID uint) (*domain.EvaluationResult, error)
	ListByAgent(agentID uint, limit int) ([]domain.EvaluationResult, error)
	Update(result *domain.EvaluationResult) error
}

// DimensionalScoreRepository 多维度评分仓储接口
type DimensionalScoreRepository interface {
	Create(score *domain.DimensionalScore) error
	ListByEvaluation(evaluationID uint) ([]domain.DimensionalScore, error)
	Update(score *domain.DimensionalScore) error
}

// ========== Repository实现 ==========

type agentMetricsRepo struct {
	db *gorm.DB
}

func (r *agentMetricsRepo) Create(metrics *domain.AgentMetrics) error {
	return r.db.Create(metrics).Error
}

func (r *agentMetricsRepo) GetByAgentAndDate(agentID uint, date string) (*domain.AgentMetrics, error) {
	var metrics domain.AgentMetrics
	err := r.db.Where("agent_id = ? AND metrics_date = ?", agentID, date).First(&metrics).Error
	return &metrics, err
}

func (r *agentMetricsRepo) ListByAgent(agentID uint, limit int) ([]domain.AgentMetrics, error) {
	var metrics []domain.AgentMetrics
	err := r.db.Where("agent_id = ?", agentID).Order("metrics_date DESC").Limit(limit).Find(&metrics).Error
	return metrics, err
}

func (r *agentMetricsRepo) Update(metrics *domain.AgentMetrics) error {
	return r.db.Save(metrics).Error
}

func (r *agentMetricsRepo) GetLatestByAgent(agentID uint) (*domain.AgentMetrics, error) {
	var metrics domain.AgentMetrics
	err := r.db.Where("agent_id = ?", agentID).Order("metrics_date DESC").First(&metrics).Error
	return &metrics, err
}

type evaluationResultRepo struct {
	db *gorm.DB
}

func (r *evaluationResultRepo) Create(result *domain.EvaluationResult) error {
	return r.db.Create(result).Error
}

func (r *evaluationResultRepo) GetByID(id uint) (*domain.EvaluationResult, error) {
	var result domain.EvaluationResult
	err := r.db.First(&result, id).Error
	return &result, err
}

func (r *evaluationResultRepo) GetBySession(sessionID uint) (*domain.EvaluationResult, error) {
	var result domain.EvaluationResult
	err := r.db.Where("session_id = ?", sessionID).First(&result).Error
	return &result, err
}

func (r *evaluationResultRepo) ListByAgent(agentID uint, limit int) ([]domain.EvaluationResult, error) {
	var results []domain.EvaluationResult
	err := r.db.Where("agent_id = ?", agentID).Order("evaluation_date DESC").Limit(limit).Find(&results).Error
	return results, err
}

func (r *evaluationResultRepo) Update(result *domain.EvaluationResult) error {
	return r.db.Save(result).Error
}

type dimensionalScoreRepo struct {
	db *gorm.DB
}

func (r *dimensionalScoreRepo) Create(score *domain.DimensionalScore) error {
	return r.db.Create(score).Error
}

func (r *dimensionalScoreRepo) ListByEvaluation(evaluationID uint) ([]domain.DimensionalScore, error) {
	var scores []domain.DimensionalScore
	err := r.db.Where("evaluation_id = ?", evaluationID).Find(&scores).Error
	return scores, err
}

func (r *dimensionalScoreRepo) Update(score *domain.DimensionalScore) error {
	return r.db.Save(score).Error
}
