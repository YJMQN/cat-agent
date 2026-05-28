package service

import (
	"encoding/json"
	"fmt"

	"cat-agent/internal/domain"
	"cat-agent/internal/repository"
)

// UserProfileService 用户画像服务
type UserProfileService struct {
	repo *repository.Repository
}

// NewUserProfileService 创建用户画像服务
func NewUserProfileService(repo *repository.Repository) *UserProfileService {
	return &UserProfileService{
		repo: repo,
	}
}

// InitializeProfile 初始化用户画像（引导式问卷）
func (s *UserProfileService) InitializeProfile(userID uint) (*domain.UserProfile, error) {
	// 检查是否已经存在
	existing, _ := s.repo.UserProfile.GetByUserID(userID)
	if existing != nil && existing.OnboardingStatus == "completed" {
		return existing, nil
	}

	profile := &domain.UserProfile{
		UserID:            userID,
		OnboardingStatus:  "pending",
		OnboardingStep:    0,
		PreferredLanguage: "zh-CN",
		InteractionStyle:  "concise",
		CommunicationTone: "professional",
		LearningStyle:     "visual",
		ResponseTime:      "moderate",
	}

	if err := s.repo.UserProfile.Create(profile); err != nil {
		return nil, fmt.Errorf("创建用户画像失败: %w", err)
	}

	return profile, nil
}

// GetNextOnboardingQuestion 获取下一个引导式问题
func (s *UserProfileService) GetNextOnboardingQuestion(userID uint) (*domain.OnboardingQuestion, error) {
	profile, err := s.repo.UserProfile.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户画像失败: %w", err)
	}

	nextStep := profile.OnboardingStep + 1
	question, err := s.repo.OnboardingQuestion.GetByStep(nextStep)
	if err != nil {
		return nil, fmt.Errorf("没有更多问题: %w", err)
	}

	return question, nil
}

// ProcessOnboardingAnswer 处理用户回答
func (s *UserProfileService) ProcessOnboardingAnswer(userID uint, questionID uint, answer string) error {
	question, err := s.repo.OnboardingQuestion.GetByStep(int(questionID))
	if err != nil {
		return fmt.Errorf("问题不存在: %w", err)
	}

	profile, err := s.repo.UserProfile.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("获取用户画像失败: %w", err)
	}

	// 根据问题类型处理答案
	switch question.MapToField {
	case "occupation":
		profile.Occupation = answer
	case "interests":
		// JSON格式的数组
		var interests []string
		if err := json.Unmarshal([]byte(answer), &interests); err != nil {
			// 简单处理，将answer作为单个值
			profile.Interests = answer
		} else {
			profile.Interests = answer
		}
	case "interaction_style":
		profile.InteractionStyle = answer
	case "communication_tone":
		profile.CommunicationTone = answer
	case "learning_style":
		profile.LearningStyle = answer
	case "response_time":
		profile.ResponseTime = answer
	}

	// 更新画像和进度
	profile.OnboardingStep = question.Step
	if question.Step >= 5 { // 假设有5个问题
		profile.OnboardingStatus = "completed"
	} else {
		profile.OnboardingStatus = "in_progress"
	}

	// 记录更新
	update := &domain.ProfileUpdate{
		UserID:    userID,
		FieldName: question.MapToField,
		NewValue:  answer,
		Source:    "onboarding",
		Reason:    fmt.Sprintf("问卷第%d步", question.Step),
	}
	if err := s.repo.ProfileUpdate.Create(update); err != nil {
		fmt.Printf("记录画像更新失败: %v\n", err)
	}

	return s.repo.UserProfile.Update(profile)
}

// UpdateProfileDimension 更新画像维度
func (s *UserProfileService) UpdateProfileDimension(userID uint, dimensionKey string, value string, score float64, evidence string) error {
	existing, err := s.repo.ProfileDimension.GetByUserAndKey(userID, dimensionKey)

	if err == nil && existing != nil {
		// 更新现有维度
		existing.Value = value
		existing.Score = score
		existing.Evidence = evidence
		return s.repo.ProfileDimension.Update(existing)
	}

	// 创建新维度
	dim := &domain.ProfileDimension{
		UserID:       userID,
		DimensionKey: dimensionKey,
		Value:        value,
		Score:        score,
		Evidence:     evidence,
	}
	return s.repo.ProfileDimension.Create(dim)
}

// InferProfileFromInteractions 从交互中推断用户画像（需要与ChatService集成）
func (s *UserProfileService) InferProfileFromInteractions(userID uint) error {
	// 这是一个高级功能，需要分析用户的历史交互
	// TODO: 实现基于交互历史的画像推断

	profile, err := s.repo.UserProfile.GetByUserID(userID)
	if err != nil {
		return err
	}

	// 示例：根据交互频率推断
	// 在实际应用中，应该从Message表分析

	return s.repo.UserProfile.Update(profile)
}

// GetUserProfile 获取用户完整画像
func (s *UserProfileService) GetUserProfile(userID uint) (map[string]interface{}, error) {
	profile, err := s.repo.UserProfile.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户画像失败: %w", err)
	}

	dimensions, err := s.repo.ProfileDimension.ListByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("获取画像维度失败: %w", err)
	}

	result := map[string]interface{}{
		"profile":    profile,
		"dimensions": dimensions,
	}

	return result, nil
}

// ExportOnboardingQuestions 导出引导式问卷（用于前端）
func (s *UserProfileService) ExportOnboardingQuestions() ([]map[string]interface{}, error) {
	questions, err := s.repo.OnboardingQuestion.List()
	if err != nil {
		return nil, fmt.Errorf("获取问题列表失败: %w", err)
	}

	var result []map[string]interface{}
	for _, q := range questions {
		item := map[string]interface{}{
			"id":            q.ID,
			"step":          q.Step,
			"category":      q.Category,
			"question":      q.Question,
			"question_type": q.QuestionType,
			"description":   q.Description,
		}

		// 如果是选择题，解析options
		if q.QuestionType == "single_choice" || q.QuestionType == "multiple_choice" {
			var options []string
			if err := json.Unmarshal([]byte(q.Options), &options); err == nil {
				item["options"] = options
			}
		}

		// 如果是量表题
		if q.QuestionType == "scale" {
			item["scale_min"] = q.ScaleMin
			item["scale_max"] = q.ScaleMax
		}

		result = append(result, item)
	}

	return result, nil
}

// ValidateProfileCompletion 验证画像完成度
func (s *UserProfileService) ValidateProfileCompletion(userID uint) (float64, error) {
	profile, err := s.repo.UserProfile.GetByUserID(userID)
	if err != nil {
		return 0, err
	}

	// 检查必填字段
	requiredFields := []string{
		profile.Occupation,
		profile.Interests,
		profile.InteractionStyle,
		profile.CommunicationTone,
	}

	completedCount := 0
	for _, field := range requiredFields {
		if field != "" {
			completedCount++
		}
	}

	completion := float64(completedCount) / float64(len(requiredFields))
	return completion, nil
}
