package service

import (
	"encoding/json"
	"fmt"
	"time"

	"cat-agent/internal/domain"
	"cat-agent/internal/repository"
)

// AgentEvaluationService Agent评估服务
type AgentEvaluationService struct {
	repo *repository.Repository
}

// NewAgentEvaluationService 创建Agent评估服务
func NewAgentEvaluationService(repo *repository.Repository) *AgentEvaluationService {
	return &AgentEvaluationService{
		repo: repo,
	}
}

// EvaluateSession 评估单个会话
func (s *AgentEvaluationService) EvaluateSession(agentID, sessionID uint, goalAchievement, planQuality, executionQuality float64) (*domain.EvaluationResult, error) {
	result := &domain.EvaluationResult{
		AgentID:          agentID,
		SessionID:        sessionID,
		EvaluationDate:   time.Now(),
		GoalAchievement:  goalAchievement,
		PlanQuality:      planQuality,
		ExecutionQuality: executionQuality,
	}

	// 根据Agent GPA框架判断失败点
	failurePoint := s.identifyFailurePoint(goalAchievement, planQuality, executionQuality)
	result.FailurePoint = failurePoint

	// 计算总分
	result.OverallScore = (goalAchievement + planQuality + executionQuality) / 3.0

	// 生成改进建议
	recommendations := s.generateRecommendations(goalAchievement, planQuality, executionQuality, failurePoint)
	recsJSON, _ := json.Marshal(recommendations)
	result.Recommendations = string(recsJSON)

	if err := s.repo.EvaluationResult.Create(result); err != nil {
		return nil, fmt.Errorf("创建评估结果失败: %w", err)
	}

	return result, nil
}

// RecordAgentMetrics 记录Agent的性能指标
func (s *AgentEvaluationService) RecordAgentMetrics(agentID uint, metrics map[string]interface{}) (*domain.AgentMetrics, error) {
	date := time.Now().Format("2006-01-02")

	// 检查是否已存在今天的指标
	existing, _ := s.repo.AgentMetrics.GetByAgentAndDate(agentID, date)
	if existing != nil {
		// 更新现有指标
		for key, value := range metrics {
			switch key {
			case "task_completion_rate":
				existing.TaskCompletionRate = value.(float64)
			case "task_quality_score":
				existing.TaskQualityScore = value.(float64)
			case "average_steps_per_task":
				existing.AverageStepsPerTask = value.(float64)
			case "unnecessary_tool_call_rate":
				existing.UnnecessaryToolCallRate = value.(float64)
			case "tool_selection_accuracy":
				existing.ToolSelectionAccuracy = value.(float64)
			case "memory_retrieval_hit_rate":
				existing.MemoryRetrievalHitRate = value.(float64)
			case "average_latency_ms":
				existing.AverageLatencyMs = value.(float64)
			case "total_token_used":
				existing.TotalTokenUsed = int64(value.(float64))
			case "error_rate":
				existing.ErrorRate = value.(float64)
			}
		}
		return existing, s.repo.AgentMetrics.Update(existing)
	}

	// 创建新的指标记录
	agentMetrics := &domain.AgentMetrics{
		AgentID:      agentID,
		MetricsDate:  date,
		SessionCount: 1,
	}

	for key, value := range metrics {
		switch key {
		case "task_completion_rate":
			agentMetrics.TaskCompletionRate = value.(float64)
		case "task_quality_score":
			agentMetrics.TaskQualityScore = value.(float64)
		case "average_steps_per_task":
			agentMetrics.AverageStepsPerTask = value.(float64)
		case "unnecessary_tool_call_rate":
			agentMetrics.UnnecessaryToolCallRate = value.(float64)
		case "tool_selection_accuracy":
			agentMetrics.ToolSelectionAccuracy = value.(float64)
		case "memory_retrieval_hit_rate":
			agentMetrics.MemoryRetrievalHitRate = value.(float64)
		case "average_latency_ms":
			agentMetrics.AverageLatencyMs = value.(float64)
		case "total_token_used":
			agentMetrics.TotalTokenUsed = int64(value.(float64))
		case "error_rate":
			agentMetrics.ErrorRate = value.(float64)
		}
	}

	if err := s.repo.AgentMetrics.Create(agentMetrics); err != nil {
		return nil, fmt.Errorf("创建指标失败: %w", err)
	}

	return agentMetrics, nil
}

// GetAgentScore 获取Agent的综合评分
func (s *AgentEvaluationService) GetAgentScore(agentID uint) (map[string]interface{}, error) {
	metrics, err := s.repo.AgentMetrics.GetLatestByAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("获取指标失败: %w", err)
	}

	score := map[string]interface{}{
		"overall_score":               (metrics.TaskCompletionRate + metrics.TaskQualityScore + metrics.MemoryRetrievalHitRate) / 3.0,
		"task_completion_rate":        metrics.TaskCompletionRate,
		"task_quality_score":          metrics.TaskQualityScore,
		"tool_selection_accuracy":     metrics.ToolSelectionAccuracy,
		"memory_retrieval_hit_rate":   metrics.MemoryRetrievalHitRate,
		"personalization_align_score": metrics.PersonalizationAlignScore,
		"efficiency_score":            1.0 - metrics.UnnecessaryToolCallRate,
	}

	return score, nil
}

// ComputeDimensionalScores 计算多维度评分
func (s *AgentEvaluationService) ComputeDimensionalScores(evaluationID uint, dimensions map[string]float64) error {
	weights := map[string]float64{
		"task_quality":     0.30,
		"step_efficiency":  0.20,
		"tool_selection":   0.20,
		"memory_retrieval": 0.15,
		"personalization":  0.10,
		"cost_efficiency":  0.05,
	}

	for dim, score := range dimensions {
		weight, exists := weights[dim]
		if !exists {
			weight = 1.0 / float64(len(dimensions))
		}

		dimScore := &domain.DimensionalScore{
			EvaluationID: evaluationID,
			Dimension:    dim,
			Score:        score,
			Weight:       weight,
		}

		if err := s.repo.DimensionalScore.Create(dimScore); err != nil {
			return fmt.Errorf("创建多维度评分失败: %w", err)
		}
	}

	return nil
}

// CompareAgentPerformance 比较多个Agent的性能
func (s *AgentEvaluationService) CompareAgentPerformance(agentIDs []uint) (map[uint]map[string]interface{}, error) {
	result := make(map[uint]map[string]interface{})

	for _, agentID := range agentIDs {
		metrics, err := s.repo.AgentMetrics.GetLatestByAgent(agentID)
		if err != nil {
			continue
		}

		result[agentID] = map[string]interface{}{
			"task_completion_rate":    metrics.TaskCompletionRate,
			"task_quality_score":      metrics.TaskQualityScore,
			"average_latency_ms":      metrics.AverageLatencyMs,
			"tool_selection_accuracy": metrics.ToolSelectionAccuracy,
			"error_rate":              metrics.ErrorRate,
		}
	}

	return result, nil
}

// GetPerformanceTrend 获取Agent的性能趋势
func (s *AgentEvaluationService) GetPerformanceTrend(agentID uint, days int) (map[string]interface{}, error) {
	metricsHistory, err := s.repo.AgentMetrics.ListByAgent(agentID, days)
	if err != nil {
		return nil, err
	}

	trend := map[string]interface{}{
		"dates":                   []string{},
		"completion_rates":        []float64{},
		"quality_scores":          []float64{},
		"average_latencies":       []float64{},
		"tool_selection_accuracy": []float64{},
	}

	for _, m := range metricsHistory {
		trend["dates"] = append(trend["dates"].([]string), m.MetricsDate)
		trend["completion_rates"] = append(trend["completion_rates"].([]float64), m.TaskCompletionRate)
		trend["quality_scores"] = append(trend["quality_scores"].([]float64), m.TaskQualityScore)
		trend["average_latencies"] = append(trend["average_latencies"].([]float64), m.AverageLatencyMs)
		trend["tool_selection_accuracy"] = append(trend["tool_selection_accuracy"].([]float64), m.ToolSelectionAccuracy)
	}

	return trend, nil
}

// 辅助方法

func (s *AgentEvaluationService) identifyFailurePoint(goal, plan, execution float64) string {
	// 根据Agent GPA框架判断失败点
	if goal < 0.3 {
		return "goal" // 目标不清楚
	}
	if plan < 0.3 {
		return "plan" // 计划不当
	}
	if execution < 0.3 {
		return "action" // 执行不力
	}
	return "none"
}

func (s *AgentEvaluationService) generateRecommendations(goal, plan, execution float64, failurePoint string) []string {
	var recommendations []string

	switch failurePoint {
	case "goal":
		recommendations = append(recommendations, "改进目标理解能力，更好地解析用户意图")
		recommendations = append(recommendations, "增强澄清问题的能力")
	case "plan":
		recommendations = append(recommendations, "优化计划生成策略，考虑更多路径选项")
		recommendations = append(recommendations, "提高工具选择的多样性和准确性")
	case "action":
		recommendations = append(recommendations, "改进执行能力，减少错误率")
		recommendations = append(recommendations, "优化工具调用顺序")
	}

	// 通用建议
	if execution < 0.5 {
		recommendations = append(recommendations, "考虑增加错误恢复机制")
	}
	if plan < 0.5 {
		recommendations = append(recommendations, "审查和优化Agent的系统提示词")
	}

	return recommendations
}
