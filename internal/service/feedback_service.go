package service

import (
	"encoding/json"
	"fmt"
	"time"

	"cat-agent/internal/domain"
	"cat-agent/internal/repository"
)

// FeedbackService 反馈服务
type FeedbackService struct {
	repo *repository.Repository
}

// NewFeedbackService 创建反馈服务
func NewFeedbackService(repo *repository.Repository) *FeedbackService {
	return &FeedbackService{
		repo: repo,
	}
}

// CreateExplicitFeedback 创建显式反馈
func (s *FeedbackService) CreateExplicitFeedback(userID, sessionID, messageID uint, feedbackType string, rating int, comment string) (*domain.UserFeedback, error) {
	feedback := &domain.UserFeedback{
		UserID:       userID,
		SessionID:    sessionID,
		MessageID:    messageID,
		FeedbackType: feedbackType,
		Rating:       rating,
		Comment:      comment,
	}

	if err := s.repo.UserFeedback.Create(feedback); err != nil {
		return nil, fmt.Errorf("创建反馈失败: %w", err)
	}

	// 触发反馈分析更新
	go s.updateFeedbackAnalysis(userID)

	return feedback, nil
}

// CreateImplicitFeedback 创建隐式反馈信号
func (s *FeedbackService) CreateImplicitFeedback(userID, sessionID uint, signalType string, value float64, isPositive bool, sessionDuration int) (*domain.ImplicitFeedback, error) {
	feedback := &domain.ImplicitFeedback{
		UserID:          userID,
		SessionID:       sessionID,
		SignalType:      signalType,
		Value:           value,
		IsPositive:      isPositive,
		SessionDuration: sessionDuration,
	}

	if err := s.repo.ImplicitFeedback.Create(feedback); err != nil {
		return nil, fmt.Errorf("创建隐式反馈失败: %w", err)
	}

	return feedback, nil
}

// TrackTaskCompletion 追踪任务完成（隐式反馈）
func (s *FeedbackService) TrackTaskCompletion(userID, sessionID uint, isCompleted bool, efficiency float64) error {
	signalType := "task_completion"
	if !isCompleted {
		signalType = "task_incomplete"
	}

	_, err := s.CreateImplicitFeedback(userID, sessionID, signalType, efficiency, isCompleted, 0)
	return err
}

// TrackInteractionDuration 追踪交互时长
func (s *FeedbackService) TrackInteractionDuration(userID, sessionID uint, durationSeconds int) error {
	// 根据时长长度判断是否正向
	isPositive := durationSeconds > 60 // 超过1分钟认为是充分交互

	_, err := s.CreateImplicitFeedback(userID, sessionID, "interaction_duration", float64(durationSeconds), isPositive, durationSeconds)
	return err
}

// AnalyzeFeedback 分析用户反馈
func (s *FeedbackService) AnalyzeFeedback(userID uint, days int) (*domain.FeedbackAnalysis, error) {
	// 获取最近的显式反馈
	explicitFeedbacks, err := s.repo.UserFeedback.ListByUser(userID, 100)
	if err != nil {
		return nil, err
	}

	// 获取最近的隐式反馈
	implicitFeedbacks, err := s.repo.ImplicitFeedback.GetRecentByUser(userID, days)
	if err != nil {
		return nil, err
	}

	analysis := &domain.FeedbackAnalysis{
		UserID:                 userID,
		AnalysisDate:           time.Now().Format("2006-01-02"),
		ProcessedFeedbackCount: len(explicitFeedbacks) + len(implicitFeedbacks),
	}

	// 计算总体满意度
	if len(explicitFeedbacks) > 0 {
		totalRating := 0
		ratingCount := 0
		for _, f := range explicitFeedbacks {
			if f.Rating > 0 {
				totalRating += f.Rating
				ratingCount++
			}
		}
		if ratingCount > 0 {
			analysis.OverallSatisfaction = float64(totalRating) / float64(ratingCount) / 5.0 // 归一化到0-1
		}
	}

	// 分析隐式反馈
	if len(implicitFeedbacks) > 0 {
		positiveCount := 0
		for _, f := range implicitFeedbacks {
			if f.IsPositive {
				positiveCount++
			}
		}
		analysis.ResponseStyleRating = float64(positiveCount) / float64(len(implicitFeedbacks))
	}

	// 提取工具偏好
	toolPrefs := s.extractToolPreferences(explicitFeedbacks)
	toolPrefsJSON, _ := json.Marshal(toolPrefs)
	analysis.ToolPreferences = string(toolPrefsJSON)

	// 识别频繁问题
	commonIssues := s.identifyCommonIssues(explicitFeedbacks)
	issuesJSON, _ := json.Marshal(commonIssues)
	analysis.FrequentIssues = string(issuesJSON)

	// 保存或更新分析结果
	existing, _ := s.repo.FeedbackAnalysis.GetByUserID(userID)
	if existing != nil {
		analysis.ID = existing.ID
		err = s.repo.FeedbackAnalysis.Update(analysis)
	} else {
		err = s.repo.FeedbackAnalysis.Create(analysis)
	}

	return analysis, err
}

// GetFeedbackSummary 获取反馈总结
func (s *FeedbackService) GetFeedbackSummary(userID uint) (map[string]interface{}, error) {
	analysis, err := s.repo.FeedbackAnalysis.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"overall_satisfaction":     analysis.OverallSatisfaction,
		"response_style_rating":    analysis.ResponseStyleRating,
		"tool_selection_accuracy":  analysis.ToolSelectionAccuracy,
		"processed_feedback_count": analysis.ProcessedFeedbackCount,
	}

	// 解析工具偏好
	if analysis.ToolPreferences != "" {
		var toolPrefs map[string]interface{}
		if err := json.Unmarshal([]byte(analysis.ToolPreferences), &toolPrefs); err == nil {
			summary["tool_preferences"] = toolPrefs
		}
	}

	// 解析风格偏好
	if analysis.StylePreferences != "" {
		var stylePrefs map[string]interface{}
		if err := json.Unmarshal([]byte(analysis.StylePreferences), &stylePrefs); err == nil {
			summary["style_preferences"] = stylePrefs
		}
	}

	return summary, nil
}

// 辅助方法

func (s *FeedbackService) updateFeedbackAnalysis(userID uint) error {
	_, err := s.AnalyzeFeedback(userID, 7)
	return err
}

func (s *FeedbackService) extractToolPreferences(feedbacks []domain.UserFeedback) map[string]float64 {
	toolScores := make(map[string]float64)
	toolCounts := make(map[string]int)

	for _, f := range feedbacks {
		// 这里假设在Comment或Aspects中包含了工具信息
		if f.Aspects != "" {
			var aspects map[string]int
			if err := json.Unmarshal([]byte(f.Aspects), &aspects); err == nil {
				for tool, score := range aspects {
					toolScores[tool] += float64(score)
					toolCounts[tool]++
				}
			}
		}
	}

	// 计算平均分数
	for tool, totalScore := range toolScores {
		count := toolCounts[tool]
		if count > 0 {
			toolScores[tool] = totalScore / float64(count) / 5.0
		}
	}

	return toolScores
}

func (s *FeedbackService) identifyCommonIssues(feedbacks []domain.UserFeedback) []string {
	issueMap := make(map[string]int)
	var issues []string

	for _, f := range feedbacks {
		if f.Comment != "" {
			// 简单的问题识别 - 实际应该用NLP
			if len(f.Comment) > 10 {
				issueMap[f.Comment[:30]]++
			}
		}
	}

	// 返回出现次数最多的问题
	for issue, count := range issueMap {
		if count > 2 {
			issues = append(issues, issue)
		}
	}

	return issues
}
