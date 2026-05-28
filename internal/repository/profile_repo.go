package repository

import (
	"cat-agent/internal/domain"

	"gorm.io/gorm"
)

// ========== 功能1：用户画像系统Repository定义 ==========

// UserProfileRepository 用户画像仓储接口
type UserProfileRepository interface {
	Create(profile *domain.UserProfile) error
	GetByUserID(userID uint) (*domain.UserProfile, error)
	Update(profile *domain.UserProfile) error
	UpdateOnboardingStatus(userID uint, status string, step int) error
}

// ProfileDimensionRepository 用户画像维度仓储接口
type ProfileDimensionRepository interface {
	Create(dim *domain.ProfileDimension) error
	GetByUserAndKey(userID uint, key string) (*domain.ProfileDimension, error)
	ListByUser(userID uint) ([]domain.ProfileDimension, error)
	Update(dim *domain.ProfileDimension) error
	DeleteByUserAndKey(userID uint, key string) error
}

// OnboardingQuestionRepository 引导式问卷仓储接口
type OnboardingQuestionRepository interface {
	Create(q *domain.OnboardingQuestion) error
	GetByStep(step int) (*domain.OnboardingQuestion, error)
	ListByCategory(category string) ([]domain.OnboardingQuestion, error)
	List() ([]domain.OnboardingQuestion, error)
	Update(q *domain.OnboardingQuestion) error
	Delete(id uint) error
}

// ProfileUpdateRepository 用户画像更新日志仓储接口
type ProfileUpdateRepository interface {
	Create(update *domain.ProfileUpdate) error
	ListByUser(userID uint, limit int) ([]domain.ProfileUpdate, error)
	ListByUserAndField(userID uint, fieldName string, limit int) ([]domain.ProfileUpdate, error)
}

// ========== Repository实现 ==========

type userProfileRepo struct {
	db *gorm.DB
}

func (r *userProfileRepo) Create(profile *domain.UserProfile) error {
	return r.db.Create(profile).Error
}

func (r *userProfileRepo) GetByUserID(userID uint) (*domain.UserProfile, error) {
	var profile domain.UserProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	return &profile, err
}

func (r *userProfileRepo) Update(profile *domain.UserProfile) error {
	return r.db.Save(profile).Error
}

func (r *userProfileRepo) UpdateOnboardingStatus(userID uint, status string, step int) error {
	return r.db.Model(&domain.UserProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"onboarding_status": status,
			"onboarding_step":   step,
		}).Error
}

type profileDimensionRepo struct {
	db *gorm.DB
}

func (r *profileDimensionRepo) Create(dim *domain.ProfileDimension) error {
	return r.db.Create(dim).Error
}

func (r *profileDimensionRepo) GetByUserAndKey(userID uint, key string) (*domain.ProfileDimension, error) {
	var dim domain.ProfileDimension
	err := r.db.Where("user_id = ? AND dimension_key = ?", userID, key).First(&dim).Error
	return &dim, err
}

func (r *profileDimensionRepo) ListByUser(userID uint) ([]domain.ProfileDimension, error) {
	var dims []domain.ProfileDimension
	err := r.db.Where("user_id = ?", userID).Find(&dims).Error
	return dims, err
}

func (r *profileDimensionRepo) Update(dim *domain.ProfileDimension) error {
	return r.db.Save(dim).Error
}

func (r *profileDimensionRepo) DeleteByUserAndKey(userID uint, key string) error {
	return r.db.Where("user_id = ? AND dimension_key = ?", userID, key).Delete(&domain.ProfileDimension{}).Error
}

type onboardingQuestionRepo struct {
	db *gorm.DB
}

func (r *onboardingQuestionRepo) Create(q *domain.OnboardingQuestion) error {
	return r.db.Create(q).Error
}

func (r *onboardingQuestionRepo) GetByStep(step int) (*domain.OnboardingQuestion, error) {
	var q domain.OnboardingQuestion
	err := r.db.Where("step = ?", step).First(&q).Error
	return &q, err
}

func (r *onboardingQuestionRepo) ListByCategory(category string) ([]domain.OnboardingQuestion, error) {
	var questions []domain.OnboardingQuestion
	err := r.db.Where("category = ?", category).Order("step ASC").Find(&questions).Error
	return questions, err
}

func (r *onboardingQuestionRepo) List() ([]domain.OnboardingQuestion, error) {
	var questions []domain.OnboardingQuestion
	err := r.db.Order("step ASC").Find(&questions).Error
	return questions, err
}

func (r *onboardingQuestionRepo) Update(q *domain.OnboardingQuestion) error {
	return r.db.Save(q).Error
}

func (r *onboardingQuestionRepo) Delete(id uint) error {
	return r.db.Delete(&domain.OnboardingQuestion{}, id).Error
}

type profileUpdateRepo struct {
	db *gorm.DB
}

func (r *profileUpdateRepo) Create(update *domain.ProfileUpdate) error {
	return r.db.Create(update).Error
}

func (r *profileUpdateRepo) ListByUser(userID uint, limit int) ([]domain.ProfileUpdate, error) {
	var updates []domain.ProfileUpdate
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&updates).Error
	return updates, err
}

func (r *profileUpdateRepo) ListByUserAndField(userID uint, fieldName string, limit int) ([]domain.ProfileUpdate, error) {
	var updates []domain.ProfileUpdate
	err := r.db.Where("user_id = ? AND field_name = ?", userID, fieldName).Order("created_at DESC").Limit(limit).Find(&updates).Error
	return updates, err
}
