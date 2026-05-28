package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"cat-agent/internal/domain"
	"cat-agent/internal/repository"
)

// EpisodicMemoryService 情景记忆服务
type EpisodicMemoryService struct {
	repo    *repository.Repository
	chatSvc *ChatService // 用于调用LLM进行总结
}

// NewEpisodicMemoryService 创建情景记忆服务
func NewEpisodicMemoryService(repo *repository.Repository, chatSvc *ChatService) *EpisodicMemoryService {
	return &EpisodicMemoryService{
		repo:    repo,
		chatSvc: chatSvc,
	}
}

// CreateEpisode 创建一个交互情景记忆
func (s *EpisodicMemoryService) CreateEpisode(userID uint, sessionID uint, episodeID string, startMessageID, endMessageID uint) error {
	// 获取对应消息范围的内容用于上下文
	// MessageRepository 提供 GetBySession(sessionID, limit)
	messages, err := s.repo.Message.GetBySession(sessionID, 1000)
	if err != nil {
		return fmt.Errorf("获取会话消息失败: %w", err)
	}

	// 构建对话上下文
	var contextBuilder strings.Builder
	var startIdx, endIdx = -1, -1
	for i, msg := range messages {
		if msg.ID == startMessageID {
			startIdx = i
		}
		if msg.ID == endMessageID {
			endIdx = i
			break
		}
	}

	if startIdx < 0 || endIdx < 0 {
		return fmt.Errorf("消息范围不有效")
	}

	// 提取指定范围的消息
	selectedMessages := messages[startIdx : endIdx+1]
	for _, msg := range selectedMessages {
		contextBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	context := contextBuilder.String()

	// 创建情景记忆条目
	episode := &domain.EpisodicMemory{
		UserID:         userID,
		SessionID:      sessionID,
		EpisodeID:      episodeID,
		StartMessageID: startMessageID,
		EndMessageID:   endMessageID,
		Context:        context,
	}

	if err := s.repo.EpisodicMemory.Create(episode); err != nil {
		return fmt.Errorf("创建情景记忆失败: %w", err)
	}

	// 异步执行LLM总结（实际应该用goroutine）
	go func() {
		s.SummarizeEpisode(episode.ID, context)
	}()

	return nil
}

// SummarizeEpisode 使用LLM对情景进行总结
func (s *EpisodicMemoryService) SummarizeEpisode(episodeID uint, context string) error {
	episode, err := s.repo.EpisodicMemory.GetByID(episodeID)
	if err != nil {
		return fmt.Errorf("获取情景记忆失败: %w", err)
	}

	// 调用LLM进行总结 - 这里是模拟实现
	// 在实际应用中应该调用ChatService或直接调用模型
	summary := s.generateSummary(context)
	keyPoints := s.extractKeyPoints(context)
	userPreferences := s.extractUserPreferences(context)

	// 更新记忆
	episode.Summary = summary
	var keyPointsJSON []byte
	keyPointsJSON, _ = json.Marshal(keyPoints)
	episode.KeyPoints = string(keyPointsJSON)

	var prefsJSON []byte
	prefsJSON, _ = json.Marshal(userPreferences)
	episode.UserPreferences = string(prefsJSON)

	return s.repo.EpisodicMemory.Update(episode)
}

// CompressEpisode 对情景进行压缩
func (s *EpisodicMemoryService) CompressEpisode(episodeID uint) error {
	episode, err := s.repo.EpisodicMemory.GetByID(episodeID)
	if err != nil {
		return err
	}

	if episode.IsCompressed {
		return nil // 已经压缩过
	}

	// 创建压缩版本的信息
	compressed := map[string]interface{}{
		"summary":    episode.Summary,
		"key_points": episode.KeyPoints,
		"user_prefs": episode.UserPreferences,
		"timestamp":  episode.CreatedAt,
	}

	compressedJSON, _ := json.Marshal(compressed)
	episode.CompressedInfo = string(compressedJSON)
	episode.IsCompressed = true

	return s.repo.EpisodicMemory.Update(episode)
}

// SearchSimilarEpisodes 搜索相似的情景
func (s *EpisodicMemoryService) SearchSimilarEpisodes(userID uint, keywords []string, limit int) ([]domain.EpisodicMemory, error) {
	return s.repo.EpisodicMemory.SearchByKeywords(userID, keywords, limit)
}

// ExtractTaskCompletionMode 提取任务完成方式
func (s *EpisodicMemoryService) ExtractTaskCompletionMode(episodeID uint, tools []string) error {
	episode, err := s.repo.EpisodicMemory.GetByID(episodeID)
	if err != nil {
		return err
	}

	mode := map[string]interface{}{
		"tools_used": tools,
		"tool_count": len(tools),
		"sequence":   "linear", // 可以从ToolCalls分析出来
		"efficiency": "standard",
	}

	modeJSON, _ := json.Marshal(mode)
	episode.TaskCompletionMode = string(modeJSON)

	return s.repo.EpisodicMemory.Update(episode)
}

// GenerateMemoryReport 生成记忆报告（用于后续的个性化）
func (s *EpisodicMemoryService) GenerateMemoryReport(userID uint) (map[string]interface{}, error) {
	episodes, err := s.repo.EpisodicMemory.ListByUser(userID, 50)
	if err != nil {
		return nil, fmt.Errorf("获取情景记忆失败: %w", err)
	}

	report := map[string]interface{}{
		"total_episodes":          len(episodes),
		"compressed_count":        0,
		"key_insights":            []string{},
		"common_tools":            map[string]int{},
		"user_preference_summary": map[string]interface{}{},
	}

	var compressedCount int
	for _, ep := range episodes {
		if ep.IsCompressed {
			compressedCount++
		}
	}
	report["compressed_count"] = compressedCount

	return report, nil
}

// 辅助方法

func (s *EpisodicMemoryService) generateSummary(context string) string {
	// 简单的总结实现 - 实际应该调用LLM
	lines := strings.Split(context, "\n")
	if len(lines) > 3 {
		return strings.Join(lines[:3], "\n") + "\n..."
	}
	return context
}

func (s *EpisodicMemoryService) extractKeyPoints(context string) []string {
	// 简单的关键点提取 - 实际应该更复杂
	var points []string
	lines := strings.Split(context, "\n")
	for i, line := range lines {
		if i%2 == 0 && line != "" {
			points = append(points, line[:min(len(line), 50)])
		}
		if len(points) >= 5 {
			break
		}
	}
	return points
}

func (s *EpisodicMemoryService) extractUserPreferences(context string) map[string]interface{} {
	// 简单的偏好提取 - 实际应该分析对话内容
	prefs := map[string]interface{}{
		"interaction_length": "medium",
		"technical_level":    "intermediate",
		"explanation_style":  "detailed",
		"code_examples":      true,
	}
	return prefs
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
