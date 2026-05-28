package repository

import (
	"cat-agent/internal/domain"
	"time"

	"gorm.io/gorm"
)

// ========== 功能3：反馈系统Repository定义 ==========

// UserFeedbackRepository 用户反馈仓储接口
type UserFeedbackRepository interface {
	Create(feedback *domain.UserFeedback) error
	GetByID(id uint) (*domain.UserFeedback, error)
	ListBySession(sessionID uint) ([]domain.UserFeedback, error)
	ListByUser(userID uint, limit int) ([]domain.UserFeedback, error)
	ListByMessage(messageID uint) ([]domain.UserFeedback, error)
	GetAggregateStats(userID uint, days int) (map[string]interface{}, error)
}

// ImplicitFeedbackRepository 隐式反馈仓储接口
type ImplicitFeedbackRepository interface {
	Create(feedback *domain.ImplicitFeedback) error
	ListByUser(userID uint, limit int) ([]domain.ImplicitFeedback, error)
	ListBySession(sessionID uint) ([]domain.ImplicitFeedback, error)
	GetRecentByUser(userID uint, days int) ([]domain.ImplicitFeedback, error)
}

// FeedbackAnalysisRepository 反馈分析结果仓储接口
type FeedbackAnalysisRepository interface {
	Create(analysis *domain.FeedbackAnalysis) error
	GetByUserID(userID uint) (*domain.FeedbackAnalysis, error)
	Update(analysis *domain.FeedbackAnalysis) error
	ListByDateRange(userID uint, startDate, endDate string) ([]domain.FeedbackAnalysis, error)
}

// ========== Repository实现 ==========

type userFeedbackRepo struct {
	db *gorm.DB
}

func (r *userFeedbackRepo) Create(feedback *domain.UserFeedback) error {
	return r.db.Create(feedback).Error
}

func (r *userFeedbackRepo) GetByID(id uint) (*domain.UserFeedback, error) {
	var feedback domain.UserFeedback
	err := r.db.First(&feedback, id).Error
	return &feedback, err
}

func (r *userFeedbackRepo) ListBySession(sessionID uint) ([]domain.UserFeedback, error) {
	var feedbacks []domain.UserFeedback
	err := r.db.Where("session_id = ?", sessionID).Order("created_at DESC").Find(&feedbacks).Error
	return feedbacks, err
}

func (r *userFeedbackRepo) ListByUser(userID uint, limit int) ([]domain.UserFeedback, error) {
	var feedbacks []domain.UserFeedback
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&feedbacks).Error
	return feedbacks, err
}

func (r *userFeedbackRepo) ListByMessage(messageID uint) ([]domain.UserFeedback, error) {
	var feedbacks []domain.UserFeedback
	err := r.db.Where("message_id = ?", messageID).Find(&feedbacks).Error
	return feedbacks, err
}

func (r *userFeedbackRepo) GetAggregateStats(userID uint, days int) (map[string]interface{}, error) {
	var result map[string]interface{}
	cutoffTime := time.Now().AddDate(0, 0, -days)

	var totalFeedbacks int64
	var avgRating float64

	r.db.Model(&domain.UserFeedback{}).
		Where("user_id = ? AND created_at >= ?", userID, cutoffTime).
		Count(&totalFeedbacks)

	r.db.Model(&domain.UserFeedback{}).
		Where("user_id = ? AND created_at >= ? AND rating > 0", userID, cutoffTime).
		Select("AVG(rating) as avg_rating").
		Scan(&avgRating)

	result = map[string]interface{}{
		"total_feedbacks": totalFeedbacks,
		"avg_rating":      avgRating,
	}

	return result, nil
}

type implicitFeedbackRepo struct {
	db *gorm.DB
}

func (r *implicitFeedbackRepo) Create(feedback *domain.ImplicitFeedback) error {
	return r.db.Create(feedback).Error
}

func (r *implicitFeedbackRepo) ListByUser(userID uint, limit int) ([]domain.ImplicitFeedback, error) {
	var feedbacks []domain.ImplicitFeedback
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&feedbacks).Error
	return feedbacks, err
}

func (r *implicitFeedbackRepo) ListBySession(sessionID uint) ([]domain.ImplicitFeedback, error) {
	var feedbacks []domain.ImplicitFeedback
	err := r.db.Where("session_id = ?", sessionID).Find(&feedbacks).Error
	return feedbacks, err
}

func (r *implicitFeedbackRepo) GetRecentByUser(userID uint, days int) ([]domain.ImplicitFeedback, error) {
	var feedbacks []domain.ImplicitFeedback
	cutoffTime := time.Now().AddDate(0, 0, -days)
	err := r.db.Where("user_id = ? AND created_at >= ?", userID, cutoffTime).
		Order("created_at DESC").
		Find(&feedbacks).Error
	return feedbacks, err
}

type feedbackAnalysisRepo struct {
	db *gorm.DB
}

func (r *feedbackAnalysisRepo) Create(analysis *domain.FeedbackAnalysis) error {
	return r.db.Create(analysis).Error
}

func (r *feedbackAnalysisRepo) GetByUserID(userID uint) (*domain.FeedbackAnalysis, error) {
	var analysis domain.FeedbackAnalysis
	err := r.db.Where("user_id = ?", userID).Order("analysis_date DESC").First(&analysis).Error
	return &analysis, err
}

func (r *feedbackAnalysisRepo) Update(analysis *domain.FeedbackAnalysis) error {
	return r.db.Save(analysis).Error
}

func (r *feedbackAnalysisRepo) ListByDateRange(userID uint, startDate, endDate string) ([]domain.FeedbackAnalysis, error) {
	var analyses []domain.FeedbackAnalysis
	err := r.db.Where("user_id = ? AND analysis_date >= ? AND analysis_date <= ?", userID, startDate, endDate).
		Order("analysis_date DESC").
		Find(&analyses).Error
	return analyses, err
}
