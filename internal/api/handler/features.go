package handler

import (
	"net/http"
	"strconv"

	"cat-agent/internal/service"

	"github.com/gin-gonic/gin"
)

// ========== 功能1：用户画像Handler ==========

type UserProfileHandler struct {
	svc *service.UserProfileService
}

func NewUserProfileHandler(svc *service.UserProfileService) *UserProfileHandler {
	return &UserProfileHandler{svc: svc}
}

func (h *UserProfileHandler) InitializeProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	profile, err := h.svc.InitializeProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *UserProfileHandler) GetOnboardingQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	question, err := h.svc.GetNextOnboardingQuestion(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, question)
}

func (h *UserProfileHandler) SubmitOnboardingAnswer(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		QuestionID uint   `json:"question_id" binding:"required"`
		Answer     string `json:"answer" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.ProcessOnboardingAnswer(userID, req.QuestionID, req.Answer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "答案已保存"})
}

func (h *UserProfileHandler) GetUserProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	profile, err := h.svc.GetUserProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *UserProfileHandler) ExportOnboardingQuestions(c *gin.Context) {
	qs, err := h.svc.ExportOnboardingQuestions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, qs)
}

func (h *UserProfileHandler) GetProfileCompletion(c *gin.Context) {
	userID := c.GetUint("user_id")
	rate, err := h.svc.ValidateProfileCompletion(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"completion_rate": rate})
}

// ========== 功能3：反馈 Handler ==========

type FeedbackHandler struct {
	svc *service.FeedbackService
}

func NewFeedbackHandler(svc *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{svc: svc}
}

func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		SessionID    uint   `json:"session_id" binding:"required"`
		MessageID    uint   `json:"message_id" binding:"required"`
		FeedbackType string `json:"feedback_type" binding:"required"`
		Rating       int    `json:"rating"`
		Comment      string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fb, err := h.svc.CreateExplicitFeedback(userID, req.SessionID, req.MessageID, req.FeedbackType, req.Rating, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, fb)
}

func (h *FeedbackHandler) GetFeedbackSummary(c *gin.Context) {
	userID := c.GetUint("user_id")
	summary, err := h.svc.GetFeedbackSummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// ========== 功能4：评估 Handler ==========

type EvaluationHandler struct {
	svc *service.AgentEvaluationService
}

func NewEvaluationHandler(svc *service.AgentEvaluationService) *EvaluationHandler {
	return &EvaluationHandler{svc: svc}
}

func (h *EvaluationHandler) GetAgentMetrics(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, _ := strconv.ParseUint(agentIDStr, 10, 32)
	score, err := h.svc.GetAgentScore(uint(agentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, score)
}

func (h *EvaluationHandler) GetAgentPerformanceTrend(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, _ := strconv.ParseUint(agentIDStr, 10, 32)
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}
	trend, err := h.svc.GetPerformanceTrend(uint(agentID), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, trend)
}

func (h *EvaluationHandler) CompareAgents(c *gin.Context) {
	var req struct{ AgentIDs []uint `json:"agent_ids" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.svc.CompareAgentPerformance(req.AgentIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ========== 功能5：可观测性 Handler ==========

type ObservabilityHandler struct {
	svc *service.ObservabilityService
}

func NewObservabilityHandler(svc *service.ObservabilityService) *ObservabilityHandler {
	return &ObservabilityHandler{svc: svc}
}

func (h *ObservabilityHandler) GetExecutionTrace(c *gin.Context) {
	traceID := c.Param("trace_id")
	details, err := h.svc.GetExecutionTraceDetails(traceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, details)
}

func (h *ObservabilityHandler) GetToolMetricsReport(c *gin.Context) {
	report, err := h.svc.GetToolMetricsReport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *ObservabilityHandler) GetModelMetricsReport(c *gin.Context) {
	report, err := h.svc.GetModelMetricsReport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *ObservabilityHandler) GetSessionMetrics(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, _ := strconv.ParseUint(sessionIDStr, 10, 32)
	metrics, err := h.svc.GetSessionMetrics(uint(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// RegisterEnhancedRoutes 注册增强相关路由，包括画像、反馈、评估、可观测性
func RegisterFeatureRoutes(r *gin.RouterGroup, handlers *Handlers) {
	// 用户画像
	profile := r.Group("/profile")
	{
		profile.POST("/init", handlers.UserProfile.InitializeProfile)
		profile.GET("/next", handlers.UserProfile.GetOnboardingQuestion)
		profile.POST("/answer", handlers.UserProfile.SubmitOnboardingAnswer)
		profile.GET("/me", handlers.UserProfile.GetUserProfile)
		profile.GET("/questions", handlers.UserProfile.ExportOnboardingQuestions)
		profile.GET("/completion", handlers.UserProfile.GetProfileCompletion)
	}

	// 反馈
	fb := r.Group("/feedback")
	{
		fb.POST("/submit", handlers.Feedback.SubmitFeedback)
		fb.GET("/summary", handlers.Feedback.GetFeedbackSummary)
	}

	// 评估
	eval := r.Group("/evaluation")
	{
		eval.GET("/agent/:agent_id/metrics", handlers.Evaluation.GetAgentMetrics)
		eval.GET("/agent/:agent_id/trend", handlers.Evaluation.GetAgentPerformanceTrend)
		eval.POST("/compare", handlers.Evaluation.CompareAgents)
	}

	// 可观测性
	obs := r.Group("/observability")
	{
		obs.GET("/trace/:trace_id", handlers.Observability.GetExecutionTrace)
		obs.GET("/tools", handlers.Observability.GetToolMetricsReport)
		obs.GET("/models", handlers.Observability.GetModelMetricsReport)
		obs.GET("/session/:session_id", handlers.Observability.GetSessionMetrics)
	}
}
