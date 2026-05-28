# Cat-Agent 增强功能实现指南

## 概述
本文档介绍了五大核心功能的实现细节和使用方法。

## 功能1：用户画像系统

### 核心概念
通过引导式问卷和交互分析，为每个用户建立完整的个性化画像。

### 数据模型
- `UserProfile` - 用户基本画像
- `ProfileDimension` - 多维度属性
- `OnboardingQuestion` - 问卷题库
- `ProfileUpdate` - 更新日志

### 关键API

#### 初始化用户画像
```go
profileSvc := services.UserProfile
profile, err := profileSvc.InitializeProfile(userID)
```

#### 获取下一个问题
```go
question, err := profileSvc.GetNextOnboardingQuestion(userID)
```

#### 处理用户回答
```go
err := profileSvc.ProcessOnboardingAnswer(userID, questionID, answer)
```

#### 获取完整画像
```go
profile, err := profileSvc.GetUserProfile(userID)
// 返回: {profile, dimensions}
```

### 初始化引导式问卷
需要在数据库初始化时插入默认问卷：

```sql
INSERT INTO onboarding_questions (step, category, question, question_type, options, map_to_field, is_required)
VALUES 
  (1, 'profile', '你的职业是?', 'free_text', NULL, 'occupation', true),
  (2, 'profile', '你的兴趣领域?', 'multiple_choice', '["AI", "工程", "产品"]', 'interests', true),
  (3, 'style', '偏好交互风格?', 'single_choice', '["简洁", "详尽"]', 'interaction_style', true),
  (4, 'style', '沟通风格?', 'single_choice', '["正式", "友好"]', 'communication_tone', true),
  (5, 'expertise', '学习风格?', 'single_choice', '["视觉", "听觉", "阅读", "动手"]', 'learning_style', true);
```

---

## 功能2：情景记忆系统

### 核心概念
从单纯的语义记忆升级到结构化的情景记忆，自动总结和压缩交互。

### 数据模型
- `EpisodicMemory` - 情景记忆条目

### 关键API

#### 创建情景记忆
```go
episodeSvc := services.EpisodicMemory
err := episodeSvc.CreateEpisode(userID, sessionID, episodeID, startMessageID, endMessageID)
```

#### 对情景进行LLM总结
```go
err := episodeSvc.SummarizeEpisode(episodeID, context)
// 自动提取：总结、关键点、用户偏好、任务完成模式
```

#### 压缩情景记忆
```go
err := episodeSvc.CompressEpisode(episodeID)
// 压缩为紧凑格式，节省存储空间
```

#### 搜索相似情景
```go
episodes, err := episodeSvc.SearchSimilarEpisodes(userID, []string{"关键词1", "关键词2"}, limit)
```

### 工作流程
1. **会话进行中** - 连续保存消息
2. **会话结束** - 触发情景记忆创建
3. **异步处理** - LLM对完整对话进行总结
4. **定期压缩** - 将旧情景压缩为紧凑格式

---

## 功能3：反馈系统

### 核心概念
通过显式和隐式反馈驱动Agent的个性化微调。

### 数据模型
- `UserFeedback` - 显式反馈（点赞/点踩/评分）
- `ImplicitFeedback` - 隐式信号（完成率、交互时长等）
- `FeedbackAnalysis` - 反馈分析结果

### 关键API

#### 提交显式反馈
```go
feedbackSvc := services.Feedback
feedback, err := feedbackSvc.CreateExplicitFeedback(
  userID, sessionID, messageID,
  "rating",           // 类型: like, dislike, rating, comment
  4,                  // 评分: 1-5
  "回复很有帮助"      // 备注
)
```

#### 记录隐式反馈
```go
// 任务完成
err := feedbackSvc.TrackTaskCompletion(userID, sessionID, true, 0.95)

// 交互时长
err := feedbackSvc.TrackInteractionDuration(userID, sessionID, 180)
```

#### 分析反馈
```go
analysis, err := feedbackSvc.AnalyzeFeedback(userID, 7) // 过去7天
// 返回: {satisfaction, style_rating, tool_preferences, frequent_issues}
```

#### 获取反馈总结
```go
summary, err := feedbackSvc.GetFeedbackSummary(userID)
// 用于Agent的人格微调
```

### 反馈驱动的优化
系统应定期调用AnalyzeFeedback，并根据结果调整：
- Agent的system prompt
- 工具选择策略
- 回复风格参数

---

## 功能4：Agent评估框架

### 核心概念
参考Agent GPA框架，从Goal、Plan、Action三个维度评估Agent性能。

### 数据模型
- `AgentMetrics` - Agent性能指标（日级别）
- `EvaluationResult` - 单个会话的评估结果
- `DimensionalScore` - 多维度评分

### 关键API

#### 评估单个会话（Agent GPA）
```go
evalSvc := services.Evaluation
result, err := evalSvc.EvaluateSession(
  agentID, sessionID,
  0.95, // goalAchievement (目标达成度)
  0.88, // planQuality (计划质量)
  0.92  // executionQuality (执行质量)
)
// 自动判断失败点: goal, plan, action, none
// 自动生成改进建议
```

#### 记录Agent指标
```go
metrics := map[string]interface{}{
  "task_completion_rate": 0.92,
  "task_quality_score": 0.88,
  "average_steps_per_task": 3.5,
  "unnecessary_tool_call_rate": 0.05,
  "tool_selection_accuracy": 0.95,
  "memory_retrieval_hit_rate": 0.82,
  "average_latency_ms": 2500.0,
  "total_token_used": 15000,
  "error_rate": 0.02,
}
_, err := evalSvc.RecordAgentMetrics(agentID, metrics)
```

#### 获取Agent评分
```go
score, err := evalSvc.GetAgentScore(agentID)
// 返回多维度评分卡
```

#### 计算多维度评分
```go
dimensions := map[string]float64{
  "task_quality": 0.90,
  "step_efficiency": 0.85,
  "tool_selection": 0.92,
  "memory_retrieval": 0.80,
  "personalization": 0.88,
  "cost_efficiency": 0.75,
}
err := evalSvc.ComputeDimensionalScores(evaluationID, dimensions)
```

#### 性能对比与趋势
```go
// 比较多个Agent
comparison, err := evalSvc.CompareAgentPerformance([]uint{agentID1, agentID2})

// 获取性能趋势
trend, err := evalSvc.GetPerformanceTrend(agentID, 30) // 过去30天
```

### 评估维度说明

| 维度 | 说明 | 权重 | 数据来源 |
|------|------|------|---------|
| 任务完成质量 | 目标达成度、质量分数 | 30% | EvaluationResult |
| 步骤效率 | 工具调用次数、无效率 | 20% | ExecutionTrace |
| 工具选择正确性 | 工具准确率 | 20% | ImplicitFeedback |
| 记忆检索效果 | 命中率、相关度 | 15% | AgentMetrics |
| 个性化对齐度 | 风格一致性 | 10% | UserFeedback |
| 成本效率 | Token/成本 | 5% | ModelMetrics |

---

## 功能5：可观测性（OpenTelemetry集成）

### 核心概念
全调用链追踪，实现Agent执行轨迹的可视化回放。

### 数据模型
- `ExecutionTrace` - 完整对话的追踪
- `TraceSpan` - 单个操作的跨度
- `ToolMetrics` - 工具性能指标
- `ModelMetrics` - 模型性能指标

### 关键API

#### 创建执行轨迹
```go
obsSvc := services.Observability
trace, err := obsSvc.CreateExecutionTrace(userID, sessionID, agentID)
// 返回 TraceID，用于后续操作
```

#### 添加追踪跨度（在每个操作时调用）
```go
// 模型调用
span, err := obsSvc.AddTraceSpan(traceID, "", "model_call", "gpt-4o-mini",
  map[string]interface{}{"prompt": "..."},
  map[string]interface{}{"response": "..."},
)

// 工具调用
span, err := obsSvc.AddTraceSpan(traceID, parentSpanID, "tool_call", "file_read",
  map[string]interface{}{"path": "/path/to/file"},
  map[string]interface{}{"content": "..."},
)

// 记忆检索
span, err := obsSvc.AddTraceSpan(traceID, "", "memory_retrieval", "semantic_search",
  map[string]interface{}{"query": "..."},
  map[string]interface{}{"results": []string{...}},
)
```

#### 完成跨度
```go
err := obsSvc.FinishTraceSpan(spanID, "success", "")
// 自动计算duration_ms
```

#### 完成执行轨迹
```go
err := obsSvc.FinishExecutionTrace(traceID, "success", stepCount, toolCallCount,
  inputTokens, outputTokens, modelLatencyMs)
```

#### 记录工具和模型指标
```go
// 工具指标
err := obsSvc.RecordToolMetrics("file_read", 150, true) // 工具名, 延迟ms, 是否成功

// 模型指标
err := obsSvc.RecordModelMetrics("gpt-4o-mini", "openai", 2000,
  500,    // inputTokens
  250,    // outputTokens
  0.95)   // qualityScore
```

#### 获取追踪详情（用于轨迹回放）
```go
details, err := obsSvc.GetExecutionTraceDetails(traceID)
// 返回完整的执行链路，包括所有spans
// 可用于在前端可视化Agent思考过程
```

#### 生成报告
```go
// 工具报告
toolReport, err := obsSvc.GetToolMetricsReport()

// 模型报告
modelReport, err := obsSvc.GetModelMetricsReport()

// 会话指标
sessionMetrics, err := obsSvc.GetSessionMetrics(sessionID)
```

### 集成到ChatService中

在`HandleChat`方法中添加追踪：

```go
// 1. 创建追踪
trace, _ := s.observabilitySvc.CreateExecutionTrace(input.UserID, session.ID, input.AgentID)
traceID := trace.TraceID

// 2. 在关键操作点添加spans
for eachStep {
    span, _ := s.observabilitySvc.AddTraceSpan(traceID, "", operationType, operationName, input, output)
    
    // ... 执行操作 ...
    
    s.observabilitySvc.FinishTraceSpan(span.SpanID, status, errorMsg)
}

// 3. 记录指标
s.observabilitySvc.RecordToolMetrics(toolName, latency, success)
s.observabilitySvc.RecordModelMetrics(modelName, provider, latency, inputTokens, outputTokens, qualityScore)

// 4. 完成追踪
s.observabilitySvc.FinishExecutionTrace(traceID, "success", stepCount, toolCallCount, inputTokens, outputTokens, modelLatencyMs)
```

---

## 前端集成建议

### 1. 用户画像模块
- 创建`OnboardingFlow.vue` - 引导式问卷流程
- 创建`ProfileView.vue` - 用户画像查看/编辑
- 创建`DimensionChart.vue` - 画像维度可视化

### 2. 情景记忆模块
- 创建`EpisodeTimeline.vue` - 情景时间线
- 创建`MemorySearch.vue` - 记忆搜索
- 创建`EpisodeDetail.vue` - 情景详情（包括总结和关键点）

### 3. 反馈模块
- 创建`FeedbackPanel.vue` - 反馈提交面板
- 创建`FeedbackStats.vue` - 反馈统计
- 创建`FeedbackTrend.vue` - 反馈趋势图表

### 4. 评估模块
- 创建`AgentScorecard.vue` - Agent评分卡
- 创建`MetricsChart.vue` - 指标可视化
- 创建`ComparisonDashboard.vue` - 多Agent对比

### 5. 可观测性模块
- 创建`TraceViewer.vue` - 执行轨迹可视化
- 创建`SpanTimeline.vue` - 跨度时间线
- 创建`MetricsReport.vue` - 指标报告

---

## 示例使用流程

### 完整的用户交互流程

```
1. 新用户注册
   └─ InitializeProfile(userID)
   
2. 引导式问卷
   └─ GetNextOnboardingQuestion() -> ProcessOnboardingAnswer() [重复5次]
   
3. 首次交互
   ├─ CreateExecutionTrace()
   ├─ Chat流程中添加Spans
   ├─ 记录ToolMetrics和ModelMetrics
   └─ FinishExecutionTrace()
   
4. 交互结束
   ├─ CreateEpisode()
   ├─ SummarizeEpisode()
   ├─ CreateExplicitFeedback()
   └─ TrackInteractionDuration()
   
5. 定期分析（每日/周）
   ├─ AnalyzeFeedback()
   ├─ RecordAgentMetrics()
   ├─ GetAgentScore()
   └─ EvaluateSession()
   
6. Agent个性化微调
   └─ 根据Feedback和Evaluation结果调整system prompt
```

---

## 数据库初始化脚本

所有数据表会在应用启动时自动迁移（GORM AutoMigrate）。

初始化默认问卷数据：

```go
// 在database.go的InitDB函数中调用
func initOnboardingQuestions(db *gorm.DB) {
	questions := []domain.OnboardingQuestion{
		{
			Step:       1,
			Category:   "profile",
			Question:   "你的职业或主要工作方向是?",
			QuestionType: "free_text",
			MapToField: "occupation",
		},
		{
			Step:       2,
			Category:   "profile",
			Question:   "你的主要兴趣领域是? (可多选)",
			QuestionType: "multiple_choice",
			Options:    `["AI/机器学习", "软件工程", "产品设计", "数据分析", "其他"]`,
			MapToField: "interests",
		},
		// ... 更多问题
	}
	db.CreateInBatches(questions, 100)
}
```

---

## 性能优化建议

1. **向量嵌入**: 对MemoryItem和EpisodicMemory的content添加向量嵌入支持
2. **异步处理**: 将LLM总结、反馈分析等放在后台任务队列
3. **缓存策略**: 缓存AgentMetrics和用户画像
4. **批量操作**: 批量插入TraceSpan记录
5. **索引优化**: 为查询频繁的字段添加数据库索引

---

## 测试建议

### 单元测试
```
- ProfileService_test.go
- EpisodicMemoryService_test.go
- FeedbackService_test.go
- EvaluationService_test.go
- ObservabilityService_test.go
```

### 集成测试
- 完整用户旅程测试
- Agent评估的准确性
- 追踪的完整性

---

## 监控和告警

关键指标监控：
- Agent整体评分趋势
- 用户满意度（反馈评分）
- 系统延迟分布
- 错误率趋势
- Token消耗趋势

建议集成Prometheus/Grafana进行可视化监控。
