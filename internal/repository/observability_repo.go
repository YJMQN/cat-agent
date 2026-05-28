package repository

import (
	"cat-agent/internal/domain"
	"time"

	"gorm.io/gorm"
)

// ========== 功能5：可观测性Repository定义 ==========

// ExecutionTraceRepository 执行轨迹仓储接口
type ExecutionTraceRepository interface {
	Create(trace *domain.ExecutionTrace) error
	GetByID(id uint) (*domain.ExecutionTrace, error)
	GetByTraceID(traceID string) (*domain.ExecutionTrace, error)
	ListBySession(sessionID uint) ([]domain.ExecutionTrace, error)
	ListByUser(userID uint, limit int) ([]domain.ExecutionTrace, error)
	Update(trace *domain.ExecutionTrace) error
	GetTracesInDateRange(userID uint, startTime, endTime time.Time) ([]domain.ExecutionTrace, error)
}

// TraceSpanRepository 追踪跨度仓储接口
type TraceSpanRepository interface {
	Create(span *domain.TraceSpan) error
	GetByID(id uint) (*domain.TraceSpan, error)
	GetBySpanID(spanID string) (*domain.TraceSpan, error)
	ListByTrace(traceID string) ([]domain.TraceSpan, error)
	Update(span *domain.TraceSpan) error
	DeleteByTrace(traceID string) error
}

// ToolMetricsRepository 工具指标仓储接口
type ToolMetricsRepository interface {
	Create(metrics *domain.ToolMetrics) error
	GetByToolAndDate(toolName, date string) (*domain.ToolMetrics, error)
	ListByTool(toolName string, limit int) ([]domain.ToolMetrics, error)
	Update(metrics *domain.ToolMetrics) error
	GetLatestByTool(toolName string) (*domain.ToolMetrics, error)
	ListAllLatest() ([]domain.ToolMetrics, error)
}

// ModelMetricsRepository 模型指标仓储接口
type ModelMetricsRepository interface {
	Create(metrics *domain.ModelMetrics) error
	GetByModelAndDate(modelName, date string) (*domain.ModelMetrics, error)
	ListByModel(modelName string, limit int) ([]domain.ModelMetrics, error)
	Update(metrics *domain.ModelMetrics) error
	GetLatestByModel(modelName string) (*domain.ModelMetrics, error)
	ListAllLatest() ([]domain.ModelMetrics, error)
}

// ========== Repository实现 ==========

type executionTraceRepo struct {
	db *gorm.DB
}

func (r *executionTraceRepo) Create(trace *domain.ExecutionTrace) error {
	return r.db.Create(trace).Error
}

func (r *executionTraceRepo) GetByID(id uint) (*domain.ExecutionTrace, error) {
	var trace domain.ExecutionTrace
	err := r.db.First(&trace, id).Error
	return &trace, err
}

func (r *executionTraceRepo) GetByTraceID(traceID string) (*domain.ExecutionTrace, error) {
	var trace domain.ExecutionTrace
	err := r.db.Where("trace_id = ?", traceID).First(&trace).Error
	return &trace, err
}

func (r *executionTraceRepo) ListBySession(sessionID uint) ([]domain.ExecutionTrace, error) {
	var traces []domain.ExecutionTrace
	err := r.db.Where("session_id = ?", sessionID).Order("start_time DESC").Find(&traces).Error
	return traces, err
}

func (r *executionTraceRepo) ListByUser(userID uint, limit int) ([]domain.ExecutionTrace, error) {
	var traces []domain.ExecutionTrace
	err := r.db.Where("user_id = ?", userID).Order("start_time DESC").Limit(limit).Find(&traces).Error
	return traces, err
}

func (r *executionTraceRepo) Update(trace *domain.ExecutionTrace) error {
	return r.db.Save(trace).Error
}

func (r *executionTraceRepo) GetTracesInDateRange(userID uint, startTime, endTime time.Time) ([]domain.ExecutionTrace, error) {
	var traces []domain.ExecutionTrace
	err := r.db.Where("user_id = ? AND start_time >= ? AND start_time <= ?", userID, startTime, endTime).
		Order("start_time DESC").
		Find(&traces).Error
	return traces, err
}

type traceSpanRepo struct {
	db *gorm.DB
}

func (r *traceSpanRepo) Create(span *domain.TraceSpan) error {
	return r.db.Create(span).Error
}

func (r *traceSpanRepo) GetByID(id uint) (*domain.TraceSpan, error) {
	var span domain.TraceSpan
	err := r.db.First(&span, id).Error
	return &span, err
}

func (r *traceSpanRepo) GetBySpanID(spanID string) (*domain.TraceSpan, error) {
	var span domain.TraceSpan
	err := r.db.Where("span_id = ?", spanID).First(&span).Error
	return &span, err
}

func (r *traceSpanRepo) ListByTrace(traceID string) ([]domain.TraceSpan, error) {
	var spans []domain.TraceSpan
	err := r.db.Where("trace_id = ?", traceID).Order("start_time ASC").Find(&spans).Error
	return spans, err
}

func (r *traceSpanRepo) Update(span *domain.TraceSpan) error {
	return r.db.Save(span).Error
}

func (r *traceSpanRepo) DeleteByTrace(traceID string) error {
	return r.db.Where("trace_id = ?", traceID).Delete(&domain.TraceSpan{}).Error
}

type toolMetricsRepo struct {
	db *gorm.DB
}

func (r *toolMetricsRepo) Create(metrics *domain.ToolMetrics) error {
	return r.db.Create(metrics).Error
}

func (r *toolMetricsRepo) GetByToolAndDate(toolName, date string) (*domain.ToolMetrics, error) {
	var metrics domain.ToolMetrics
	err := r.db.Where("tool_name = ? AND metrics_date = ?", toolName, date).First(&metrics).Error
	return &metrics, err
}

func (r *toolMetricsRepo) ListByTool(toolName string, limit int) ([]domain.ToolMetrics, error) {
	var metrics []domain.ToolMetrics
	err := r.db.Where("tool_name = ?", toolName).Order("metrics_date DESC").Limit(limit).Find(&metrics).Error
	return metrics, err
}

func (r *toolMetricsRepo) Update(metrics *domain.ToolMetrics) error {
	return r.db.Save(metrics).Error
}

func (r *toolMetricsRepo) GetLatestByTool(toolName string) (*domain.ToolMetrics, error) {
	var metrics domain.ToolMetrics
	err := r.db.Where("tool_name = ?", toolName).Order("metrics_date DESC").First(&metrics).Error
	return &metrics, err
}

func (r *toolMetricsRepo) ListAllLatest() ([]domain.ToolMetrics, error) {
	var metrics []domain.ToolMetrics
	// 获取每个工具最新的指标
	err := r.db.Distinct("tool_name").Order("tool_name, metrics_date DESC").Find(&metrics).Error
	return metrics, err
}

type modelMetricsRepo struct {
	db *gorm.DB
}

func (r *modelMetricsRepo) Create(metrics *domain.ModelMetrics) error {
	return r.db.Create(metrics).Error
}

func (r *modelMetricsRepo) GetByModelAndDate(modelName, date string) (*domain.ModelMetrics, error) {
	var metrics domain.ModelMetrics
	err := r.db.Where("model_name = ? AND metrics_date = ?", modelName, date).First(&metrics).Error
	return &metrics, err
}

func (r *modelMetricsRepo) ListByModel(modelName string, limit int) ([]domain.ModelMetrics, error) {
	var metrics []domain.ModelMetrics
	err := r.db.Where("model_name = ?", modelName).Order("metrics_date DESC").Limit(limit).Find(&metrics).Error
	return metrics, err
}

func (r *modelMetricsRepo) Update(metrics *domain.ModelMetrics) error {
	return r.db.Save(metrics).Error
}

func (r *modelMetricsRepo) GetLatestByModel(modelName string) (*domain.ModelMetrics, error) {
	var metrics domain.ModelMetrics
	err := r.db.Where("model_name = ?", modelName).Order("metrics_date DESC").First(&metrics).Error
	return &metrics, err
}

func (r *modelMetricsRepo) ListAllLatest() ([]domain.ModelMetrics, error) {
	var metrics []domain.ModelMetrics
	// 获取每个模型最新的指标
	err := r.db.Distinct("model_name").Order("model_name, metrics_date DESC").Find(&metrics).Error
	return metrics, err
}
