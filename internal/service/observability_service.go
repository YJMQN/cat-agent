package service

import (
	"encoding/json"
	"fmt"
	"time"

	"cat-agent/internal/domain"
	"cat-agent/internal/repository"

	"github.com/google/uuid"
)

// ObservabilityService 可观测性服务 - 支持OpenTelemetry风格的追踪
type ObservabilityService struct {
	repo *repository.Repository
}

// NewObservabilityService 创建可观测性服务
func NewObservabilityService(repo *repository.Repository) *ObservabilityService {
	return &ObservabilityService{
		repo: repo,
	}
}

// CreateExecutionTrace 创建执行轨迹
func (s *ObservabilityService) CreateExecutionTrace(userID, sessionID, agentID uint) (*domain.ExecutionTrace, error) {
	traceID := uuid.New().String()

	trace := &domain.ExecutionTrace{
		TraceID:   traceID,
		UserID:    userID,
		SessionID: sessionID,
		AgentID:   agentID,
		StartTime: time.Now(),
		Status:    "running",
	}

	if err := s.repo.ExecutionTrace.Create(trace); err != nil {
		return nil, fmt.Errorf("创建执行轨迹失败: %w", err)
	}

	return trace, nil
}

// FinishExecutionTrace 完成执行轨迹
func (s *ObservabilityService) FinishExecutionTrace(traceID string, status string, stepCount, toolCallCount int, inputTokens, outputTokens, modelLatencyMs int) error {
	trace, err := s.repo.ExecutionTrace.GetByTraceID(traceID)
	if err != nil {
		return fmt.Errorf("获取追踪失败: %w", err)
	}

	trace.EndTime = time.Now()
	trace.DurationMs = trace.EndTime.Sub(trace.StartTime).Milliseconds()
	trace.Status = status
	trace.StepCount = stepCount
	trace.ToolCallCount = toolCallCount
	trace.InputTokens = inputTokens
	trace.OutputTokens = outputTokens
	trace.TotalTokens = inputTokens + outputTokens
	trace.ModelLatencyMs = int64(modelLatencyMs)

	return s.repo.ExecutionTrace.Update(trace)
}

// AddTraceSpan 添加追踪跨度
func (s *ObservabilityService) AddTraceSpan(traceID, parentSpanID, operationType, operationName string, input, output interface{}) (*domain.TraceSpan, error) {
	spanID := uuid.New().String()

	inputJSON, _ := json.Marshal(input)
	outputJSON, _ := json.Marshal(output)

	span := &domain.TraceSpan{
		TraceID:       traceID,
		SpanID:        spanID,
		ParentSpanID:  parentSpanID,
		OperationType: operationType,
		OperationName: operationName,
		StartTime:     time.Now(),
		Input:         string(inputJSON),
		Output:        string(outputJSON),
		Status:        "success",
	}

	if err := s.repo.TraceSpan.Create(span); err != nil {
		return nil, fmt.Errorf("创建追踪跨度失败: %w", err)
	}

	return span, nil
}

// FinishTraceSpan 完成追踪跨度
func (s *ObservabilityService) FinishTraceSpan(spanID string, status string, errorMessage string) error {
	span, err := s.repo.TraceSpan.GetBySpanID(spanID)
	if err != nil {
		return fmt.Errorf("获取跨度失败: %w", err)
	}

	span.EndTime = time.Now()
	span.DurationMs = span.EndTime.Sub(span.StartTime).Milliseconds()
	span.Status = status
	span.ErrorMessage = errorMessage

	return s.repo.TraceSpan.Update(span)
}

// RecordToolMetrics 记录工具指标
func (s *ObservabilityService) RecordToolMetrics(toolName string, latencyMs int64, success bool) error {
	date := time.Now().Format("2006-01-02")

	metrics, err := s.repo.ToolMetrics.GetByToolAndDate(toolName, date)
	if err == nil && metrics != nil {
		// 更新现有指标
		metrics.CallCount++
		if success {
			metrics.SuccessCount++
		} else {
			metrics.FailureCount++
		}

		metrics.TotalLatencyMs += latencyMs
		metrics.AverageLatencyMs = float64(metrics.TotalLatencyMs) / float64(metrics.CallCount)

		if latencyMs > metrics.MaxLatencyMs {
			metrics.MaxLatencyMs = latencyMs
		}
		if latencyMs < metrics.MinLatencyMs || metrics.MinLatencyMs == 0 {
			metrics.MinLatencyMs = latencyMs
		}

		metrics.SuccessRate = float64(metrics.SuccessCount) / float64(metrics.CallCount)
		metrics.ErrorRate = float64(metrics.FailureCount) / float64(metrics.CallCount)
		metrics.LastUsedAt = time.Now()

		return s.repo.ToolMetrics.Update(metrics)
	}

	// 创建新指标
	newMetrics := &domain.ToolMetrics{
		ToolName:       toolName,
		MetricsDate:    date,
		CallCount:      1,
		TotalLatencyMs: latencyMs,
		MaxLatencyMs:   latencyMs,
		MinLatencyMs:   latencyMs,
		LastUsedAt:     time.Now(),
	}

	if success {
		newMetrics.SuccessCount = 1
		newMetrics.SuccessRate = 1.0
		newMetrics.ErrorRate = 0.0
	} else {
		newMetrics.FailureCount = 1
		newMetrics.SuccessRate = 0.0
		newMetrics.ErrorRate = 1.0
	}

	newMetrics.AverageLatencyMs = float64(latencyMs)

	return s.repo.ToolMetrics.Create(newMetrics)
}

// RecordModelMetrics 记录模型指标
func (s *ObservabilityService) RecordModelMetrics(modelName, provider string, latencyMs int64, inputTokens, outputTokens int, qualityScore float64) error {
	date := time.Now().Format("2006-01-02")

	metrics, err := s.repo.ModelMetrics.GetByModelAndDate(modelName, date)
	if err == nil && metrics != nil {
		// 更新现有指标
		metrics.CallCount++
		metrics.SuccessCount++ // 假设成功
		metrics.TotalLatencyMs += latencyMs
		metrics.AverageLatencyMs = float64(metrics.TotalLatencyMs) / float64(metrics.CallCount)

		metrics.TotalInputTokens += int64(inputTokens)
		metrics.TotalOutputTokens += int64(outputTokens)
		metrics.AvgInputTokens = float64(metrics.TotalInputTokens) / float64(metrics.CallCount)
		metrics.AvgOutputTokens = float64(metrics.TotalOutputTokens) / float64(metrics.CallCount)

		metrics.SuccessRate = float64(metrics.SuccessCount) / float64(metrics.CallCount)
		metrics.QualityScore = (metrics.QualityScore + qualityScore) / 2.0
		metrics.LastUsedAt = time.Now()

		return s.repo.ModelMetrics.Update(metrics)
	}

	// 创建新指标
	newMetrics := &domain.ModelMetrics{
		ModelName:         modelName,
		Provider:          provider,
		MetricsDate:       date,
		CallCount:         1,
		SuccessCount:      1,
		TotalLatencyMs:    latencyMs,
		MaxLatencyMs:      latencyMs,
		MinLatencyMs:      latencyMs,
		TotalInputTokens:  int64(inputTokens),
		TotalOutputTokens: int64(outputTokens),
		AvgInputTokens:    float64(inputTokens),
		AvgOutputTokens:   float64(outputTokens),
		SuccessRate:       1.0,
		QualityScore:      qualityScore,
		LastUsedAt:        time.Now(),
	}

	return s.repo.ModelMetrics.Create(newMetrics)
}

// GetExecutionTraceDetails 获取执行轨迹详情
func (s *ObservabilityService) GetExecutionTraceDetails(traceID string) (map[string]interface{}, error) {
	trace, err := s.repo.ExecutionTrace.GetByTraceID(traceID)
	if err != nil {
		return nil, fmt.Errorf("获取执行轨迹失败: %w", err)
	}

	// 获取所有跨度
	spans, err := s.repo.TraceSpan.ListByTrace(traceID)
	if err != nil {
		return nil, fmt.Errorf("获取追踪跨度失败: %w", err)
	}

	// 构建跨度层级
	spanMap := make(map[string]interface{})
	for _, span := range spans {
		spanMap[span.SpanID] = map[string]interface{}{
			"operation_type": span.OperationType,
			"operation_name": span.OperationName,
			"duration_ms":    span.DurationMs,
			"status":         span.Status,
			"error_message":  span.ErrorMessage,
		}
	}

	details := map[string]interface{}{
		"trace_id":         trace.TraceID,
		"session_id":       trace.SessionID,
		"agent_id":         trace.AgentID,
		"start_time":       trace.StartTime,
		"end_time":         trace.EndTime,
		"duration_ms":      trace.DurationMs,
		"step_count":       trace.StepCount,
		"tool_call_count":  trace.ToolCallCount,
		"status":           trace.Status,
		"input_tokens":     trace.InputTokens,
		"output_tokens":    trace.OutputTokens,
		"total_tokens":     trace.TotalTokens,
		"model_latency_ms": trace.ModelLatencyMs,
		"spans":            spanMap,
	}

	return details, nil
}

// GetToolMetricsReport 获取工具指标报告
func (s *ObservabilityService) GetToolMetricsReport() (map[string]interface{}, error) {
	allMetrics, err := s.repo.ToolMetrics.ListAllLatest()
	if err != nil {
		return nil, err
	}

	report := map[string]interface{}{
		"total_tools": len(allMetrics),
		"tools":       []map[string]interface{}{},
	}

	for _, m := range allMetrics {
		toolReport := map[string]interface{}{
			"tool_name":      m.ToolName,
			"call_count":     m.CallCount,
			"success_count":  m.SuccessCount,
			"failure_count":  m.FailureCount,
			"success_rate":   m.SuccessRate,
			"avg_latency_ms": m.AverageLatencyMs,
			"max_latency_ms": m.MaxLatencyMs,
			"min_latency_ms": m.MinLatencyMs,
		}
		report["tools"] = append(report["tools"].([]map[string]interface{}), toolReport)
	}

	return report, nil
}

// GetModelMetricsReport 获取模型指标报告
func (s *ObservabilityService) GetModelMetricsReport() (map[string]interface{}, error) {
	allMetrics, err := s.repo.ModelMetrics.ListAllLatest()
	if err != nil {
		return nil, err
	}

	report := map[string]interface{}{
		"total_models": len(allMetrics),
		"models":       []map[string]interface{}{},
	}

	for _, m := range allMetrics {
		modelReport := map[string]interface{}{
			"model_name":        m.ModelName,
			"provider":          m.Provider,
			"call_count":        m.CallCount,
			"success_rate":      m.SuccessRate,
			"avg_latency_ms":    m.AverageLatencyMs,
			"avg_input_tokens":  m.AvgInputTokens,
			"avg_output_tokens": m.AvgOutputTokens,
			"quality_score":     m.QualityScore,
		}
		report["models"] = append(report["models"].([]map[string]interface{}), modelReport)
	}

	return report, nil
}

// GetSessionMetrics 获取会话指标
func (s *ObservabilityService) GetSessionMetrics(sessionID uint) (map[string]interface{}, error) {
	traces, err := s.repo.ExecutionTrace.ListBySession(sessionID)
	if err != nil {
		return nil, err
	}

	metrics := map[string]interface{}{
		"session_id":        sessionID,
		"trace_count":       len(traces),
		"total_duration_ms": int64(0),
		"total_tokens":      int64(0),
		"tool_calls":        0,
		"avg_latency_ms":    float64(0),
	}

	totalLatency := int64(0)
	for _, trace := range traces {
		metrics["total_duration_ms"] = metrics["total_duration_ms"].(int64) + trace.DurationMs
		metrics["total_tokens"] = metrics["total_tokens"].(int64) + int64(trace.TotalTokens)
		metrics["tool_calls"] = metrics["tool_calls"].(int) + trace.ToolCallCount
		totalLatency += trace.ModelLatencyMs
	}

	if len(traces) > 0 {
		metrics["avg_latency_ms"] = float64(totalLatency) / float64(len(traces))
	}

	return metrics, nil
}
