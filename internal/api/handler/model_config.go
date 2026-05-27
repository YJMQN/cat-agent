package handler

import (
	"cat-agent/internal/domain"
	"cat-agent/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ModelConfigHandler 模型配置管理处理器
type ModelConfigHandler struct {
	repo *repository.Repository
}

// NewModelConfigHandler 创建处理器
func NewModelConfigHandler(repo *repository.Repository) *ModelConfigHandler {
	return &ModelConfigHandler{repo: repo}
}

// List 列出所有模型配置
func (h *ModelConfigHandler) List(c *gin.Context) {
	configs, err := h.repo.ModelConfig.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型配置列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

// Create 创建模型配置
func (h *ModelConfigHandler) Create(c *gin.Context) {
	var req struct {
		Provider     string `json:"provider" binding:"required"`
		BaseURL      string `json:"base_url" binding:"required"`
		APIKey       string `json:"api_key"`
		DefaultModel string `json:"default_model" binding:"required"`
		IsDefault    bool   `json:"is_default"`
		Enabled      bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := &domain.GlobalModelConfig{
		Provider:     req.Provider,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		DefaultModel: req.DefaultModel,
		IsDefault:    req.IsDefault,
		Enabled:      req.Enabled,
	}
	if err := h.repo.ModelConfig.Create(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建模型配置失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": cfg})
}

// GetByProvider 根据提供者名称获取模型配置
func (h *ModelConfigHandler) GetByProvider(c *gin.Context) {
	provider := c.Param("provider")
	cfg, err := h.repo.ModelConfig.GetByProvider(provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cfg})
}

// Update 更新模型配置（按provider查询更新）
func (h *ModelConfigHandler) Update(c *gin.Context) {
	provider := c.Param("provider")
	cfg, err := h.repo.ModelConfig.GetByProvider(provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	var req struct {
		BaseURL      string `json:"base_url"`
		APIKey       string `json:"api_key"`
		DefaultModel string `json:"default_model"`
		IsDefault    *bool  `json:"is_default"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BaseURL != "" {
		cfg.BaseURL = req.BaseURL
	}
	if req.APIKey != "" {
		cfg.APIKey = req.APIKey
	}
	if req.DefaultModel != "" {
		cfg.DefaultModel = req.DefaultModel
	}
	if req.IsDefault != nil {
		cfg.IsDefault = *req.IsDefault
	}
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}

	if err := h.repo.ModelConfig.Update(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cfg})
}

// Delete 删除模型配置
func (h *ModelConfigHandler) Delete(c *gin.Context) {
	provider := c.Param("provider")
	cfg, err := h.repo.ModelConfig.GetByProvider(provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	if err := h.repo.ModelConfig.Delete(cfg.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
