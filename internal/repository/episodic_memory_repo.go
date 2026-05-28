package repository

import (
	"cat-agent/internal/domain"

	"gorm.io/gorm"
)

// ========== 功能2：情景记忆Repository定义 ==========

// EpisodicMemoryRepository 情景记忆仓储接口
type EpisodicMemoryRepository interface {
	Create(memory *domain.EpisodicMemory) error
	GetByID(id uint) (*domain.EpisodicMemory, error)
	GetByEpisodeID(episodeID string) (*domain.EpisodicMemory, error)
	ListBySession(sessionID uint, limit int) ([]domain.EpisodicMemory, error)
	ListByUser(userID uint, limit int) ([]domain.EpisodicMemory, error)
	Update(memory *domain.EpisodicMemory) error
	MarkAsCompressed(id uint) error
	SearchByKeywords(userID uint, keywords []string, limit int) ([]domain.EpisodicMemory, error)
}

// ========== Repository实现 ==========

type episodicMemoryRepo struct {
	db *gorm.DB
}

func (r *episodicMemoryRepo) Create(memory *domain.EpisodicMemory) error {
	return r.db.Create(memory).Error
}

func (r *episodicMemoryRepo) GetByID(id uint) (*domain.EpisodicMemory, error) {
	var memory domain.EpisodicMemory
	err := r.db.First(&memory, id).Error
	return &memory, err
}

func (r *episodicMemoryRepo) GetByEpisodeID(episodeID string) (*domain.EpisodicMemory, error) {
	var memory domain.EpisodicMemory
	err := r.db.Where("episode_id = ?", episodeID).First(&memory).Error
	return &memory, err
}

func (r *episodicMemoryRepo) ListBySession(sessionID uint, limit int) ([]domain.EpisodicMemory, error) {
	var memories []domain.EpisodicMemory
	err := r.db.Where("session_id = ?", sessionID).Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

func (r *episodicMemoryRepo) ListByUser(userID uint, limit int) ([]domain.EpisodicMemory, error) {
	var memories []domain.EpisodicMemory
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

func (r *episodicMemoryRepo) Update(memory *domain.EpisodicMemory) error {
	return r.db.Save(memory).Error
}

func (r *episodicMemoryRepo) MarkAsCompressed(id uint) error {
	return r.db.Model(&domain.EpisodicMemory{}).Where("id = ?", id).Update("is_compressed", true).Error
}

func (r *episodicMemoryRepo) SearchByKeywords(userID uint, keywords []string, limit int) ([]domain.EpisodicMemory, error) {
	var memories []domain.EpisodicMemory
	query := r.db.Where("user_id = ?", userID)
	for _, keyword := range keywords {
		query = query.Where("key_points LIKE ? OR summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	err := query.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}
