package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"cat-agent/internal/config"
	"cat-agent/internal/domain"
	"cat-agent/internal/repository"

	"github.com/google/uuid"
)

// ========== 多Agent协作编排服务 ==========

// OrchestrateService 工作流编排服务
type OrchestrateService struct {
	repo             *repository.Repository
	chatService      *ChatService
	cfg              *config.Config
	activeExecutions map[uint]context.CancelFunc // 活跃执行及其取消函数
	executionsMu     sync.RWMutex
}

// NewOrchestrateService 创建编排服务
func NewOrchestrateService(
	repo *repository.Repository,
	chatService *ChatService,
	cfg *config.Config,
) *OrchestrateService {
	return &OrchestrateService{
		repo:             repo,
		chatService:      chatService,
		cfg:              cfg,
		activeExecutions: make(map[uint]context.CancelFunc),
	}
}

// WorkflowInput 创建工作流输入
type WorkflowInput struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Steps       []domain.WorkflowStep `json:"steps"`
	InputSchema string                `json:"input_schema"`
	UserID      uint                  `json:"user_id"`
}

// WorkflowOutput 工作流输出
type WorkflowOutput struct {
	ID          uint                  `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Steps       []domain.WorkflowStep `json:"steps"`
	InputSchema string                `json:"input_schema"`
	Status      string                `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
}

// CreateWorkflow 创建工作流
func (s *OrchestrateService) CreateWorkflow(ctx context.Context, input *WorkflowInput) (*WorkflowOutput, error) {
	// 验证步骤
	if len(input.Steps) == 0 {
		return nil, fmt.Errorf("工作流必须至少包含一个步骤")
	}

	// 验证步骤依赖关系
	if err := s.validateSteps(input.Steps); err != nil {
		return nil, fmt.Errorf("步骤验证失败: %w", err)
	}

	// 序列化步骤
	stepsJSON, err := json.Marshal(input.Steps)
	if err != nil {
		return nil, fmt.Errorf("序列化步骤失败: %w", err)
	}

	workflow := &domain.Workflow{
		Name:        input.Name,
		Description: input.Description,
		StepsJSON:   string(stepsJSON),
		InputSchema: input.InputSchema,
		Status:      "draft",
		CreatedBy:   input.UserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Workflow.Create(workflow); err != nil {
		return nil, fmt.Errorf("保存工作流失败: %w", err)
	}

	return s.workflowToOutput(workflow, input.Steps), nil
}

// validateSteps 验证步骤依赖关系
func (s *OrchestrateService) validateSteps(steps []domain.WorkflowStep) error {
	stepIDs := make(map[string]bool)
	for _, step := range steps {
		if step.ID == "" {
			return fmt.Errorf("步骤必须有ID")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("步骤ID重复: %s", step.ID)
		}
		stepIDs[step.ID] = true

		// 验证Agent存在
		if step.AgentID == 0 {
			return fmt.Errorf("步骤 %s 必须指定AgentID", step.ID)
		}
		agent, err := s.repo.Agent.GetByID(step.AgentID)
		if err != nil {
			return fmt.Errorf("步骤 %s 的Agent不存在: %w", step.ID, err)
		}
		if agent.Status != "running" {
			return fmt.Errorf("步骤 %s 的Agent未运行，请先启动Agent", step.ID)
		}
	}

	// 验证输入来源
	for _, step := range steps {
		for _, from := range step.InputFrom {
			if !stepIDs[from] && from != "input" { // "input"表示来自工作流输入
				return fmt.Errorf("步骤 %s 的输入来源 %s 不存在", step.ID, from)
			}
		}
	}

	return nil
}

// GetWorkflow 获取工作流详情
func (s *OrchestrateService) GetWorkflow(ctx context.Context, workflowID uint) (*WorkflowOutput, error) {
	workflow, err := s.repo.Workflow.GetByID(workflowID)
	if err != nil {
		return nil, fmt.Errorf("工作流不存在: %w", err)
	}

	var steps []domain.WorkflowStep
	if err := json.Unmarshal([]byte(workflow.StepsJSON), &steps); err != nil {
		return nil, fmt.Errorf("解析步骤失败: %w", err)
	}

	return s.workflowToOutput(workflow, steps), nil
}

// ListWorkflows 列出工作流
func (s *OrchestrateService) ListWorkflows(ctx context.Context, userID uint) ([]WorkflowOutput, error) {
	workflows, err := s.repo.Workflow.List(userID)
	if err != nil {
		return nil, fmt.Errorf("获取工作流列表失败: %w", err)
	}

	var outputs []WorkflowOutput
	for _, wf := range workflows {
		var steps []domain.WorkflowStep
		if err := json.Unmarshal([]byte(wf.StepsJSON), &steps); err != nil {
			continue // 跳过解析失败的
		}
		outputs = append(outputs, *s.workflowToOutput(&wf, steps))
	}

	return outputs, nil
}

// DeleteWorkflow 删除工作流
func (s *OrchestrateService) DeleteWorkflow(ctx context.Context, workflowID uint, userID uint) error {
	workflow, err := s.repo.Workflow.GetByID(workflowID)
	if err != nil {
		return fmt.Errorf("工作流不存在: %w", err)
	}
	if workflow.CreatedBy != userID {
		return fmt.Errorf("无权限删除此工作流")
	}
	return s.repo.Workflow.Delete(workflowID)
}

// ========== 工作流执行 ==========

// ExecutionInput 执行输入
type ExecutionInput struct {
	WorkflowID uint                   `json:"workflow_id"`
	UserID     uint                   `json:"user_id"`
	Input      map[string]interface{} `json:"input"`
}

// ExecutionOutput 执行输出
type ExecutionOutput struct {
	ID          uint                   `json:"id"`
	UUID        string                 `json:"uuid"`
	WorkflowID  uint                   `json:"workflow_id"`
	Status      string                 `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	StepResults []StepResultOutput     `json:"step_results,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	Duration    int                    `json:"duration"`
}

// StepResultOutput 步骤结果输出
type StepResultOutput struct {
	StepID   string                 `json:"step_id"`
	AgentID  uint                   `json:"agent_id"`
	Status   string                 `json:"status"`
	Output   map[string]interface{} `json:"output,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Duration int                    `json:"duration"`
}

// ExecuteWorkflow 执行工作流
func (s *OrchestrateService) ExecuteWorkflow(ctx context.Context, input *ExecutionInput) (*ExecutionOutput, error) {
	// 获取工作流
	workflow, err := s.repo.Workflow.GetByID(input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("工作流不存在: %w", err)
	}

	var steps []domain.WorkflowStep
	if err := json.Unmarshal([]byte(workflow.StepsJSON), &steps); err != nil {
		return nil, fmt.Errorf("解析步骤失败: %w", err)
	}

	// 序列化输入
	inputJSON, err := json.Marshal(input.Input)
	if err != nil {
		return nil, fmt.Errorf("序列化输入失败: %w", err)
	}

	// 创建执行记录
	execution := &domain.WorkflowExecution{
		WorkflowID: input.WorkflowID,
		UUID:       uuid.New().String(),
		UserID:     input.UserID,
		InputJSON:  string(inputJSON),
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	if err := s.repo.WorkflowExecution.Create(execution); err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	// 创建带超时的执行上下文
	execCtx, cancel := context.WithTimeout(ctx, s.cfg.AgentTimeout)
	s.executionsMu.Lock()
	s.activeExecutions[execution.ID] = cancel
	s.executionsMu.Unlock()

	// 开始执行
	execution.Status = "running"
	execution.StartedAt = time.Now()
	s.repo.WorkflowExecution.Update(execution)

	// 异步执行工作流
	go s.executeWorkflowAsync(execCtx, execution, steps, input.Input)

	return &ExecutionOutput{
		ID:         execution.ID,
		UUID:       execution.UUID,
		WorkflowID: execution.WorkflowID,
		Status:     execution.Status,
		StartedAt:  execution.StartedAt,
	}, nil
}

// executeWorkflowAsync 异步执行工作流
func (s *OrchestrateService) executeWorkflowAsync(ctx context.Context, execution *domain.WorkflowExecution, steps []domain.WorkflowStep, input map[string]interface{}) {
	defer func() {
		s.executionsMu.Lock()
		delete(s.activeExecutions, execution.ID)
		s.executionsMu.Unlock()
	}()

	// 存储每个步骤的输出
	stepOutputs := make(map[string]map[string]interface{})
	stepOutputs["input"] = input

	var stepResults []StepResultOutput
	var finalOutput map[string]interface{}
	var execErr error
	// 并行执行可同时运行的步骤：按照依赖关系动态调度
	executed := make(map[string]bool)
	muOutputs := sync.Mutex{}
	muResults := sync.Mutex{}

	totalSteps := len(steps)
	stepByID := make(map[string]domain.WorkflowStep)
	for _, st := range steps {
		stepByID[st.ID] = st
	}

	// 计算拓扑顺序以便后续选择最终输出（不用于调度）
	executionOrder := s.buildExecutionOrder(steps)

	for len(executed) < totalSteps {
		// 检查取消
		if ctx.Err() != nil {
			execution.Status = "cancelled"
			execErr = fmt.Errorf("工作流被取消")
			break
		}

		// 收集当前可执行且未执行的步骤
		runnable := make([]domain.WorkflowStep, 0)
		for _, step := range steps {
			if executed[step.ID] {
				continue
			}
			// 检查条件：如果存在且为 false，则跳过执行（标记为已执行为 skipped）
			if step.Condition != "" {
				if !s.evalCondition(step.Condition, stepOutputs, input) {
					// 标记为已执行，记录为 skipped
					executed[step.ID] = true
					se := StepResultOutput{StepID: step.ID, AgentID: step.AgentID, Status: "skipped", Output: map[string]interface{}{}, Duration: 0}
					muResults.Lock()
					stepResults = append(stepResults, se)
					muResults.Unlock()
					continue
				}
			}

			canExecute := true
			for _, from := range step.InputFrom {
				if from != "input" && !executed[from] {
					canExecute = false
					break
				}
			}
			if canExecute {
				runnable = append(runnable, step)
			}
		}

		if len(runnable) == 0 {
			// 环形依赖或无法继续
			execErr = fmt.Errorf("无法找到可执行步骤，可能存在未满足的依赖或环形依赖")
			execution.Status = "failed"
			break
		}

		// 并发执行这一批可运行步骤
		var wg sync.WaitGroup
		for _, step := range runnable {
			wg.Add(1)
			go func(st domain.WorkflowStep) {
				defer wg.Done()

				// 构建步骤输入（读 stepOutputs 需要锁）
				muOutputs.Lock()
				stepInput := s.buildStepInput(st, stepOutputs, input)
				muOutputs.Unlock()

				// 创建步骤执行记录
				stepExec := &domain.StepExecution{
					ExecutionID: execution.ID,
					StepID:      st.ID,
					AgentID:     st.AgentID,
					InputJSON:   "{}",
					Status:      "pending",
					CreatedAt:   time.Now(),
				}
				inputJSON, _ := json.Marshal(stepInput)
				stepExec.InputJSON = string(inputJSON)
				_ = s.repo.StepExecution.Create(stepExec)

				// 更新为 running
				stepExec.Status = "running"
				stepExec.StartedAt = time.Now()
				_ = s.repo.StepExecution.Update(stepExec)

				// per-step timeout 与重试
				timeoutSec := st.Timeout
				if timeoutSec <= 0 {
					timeoutSec = 60
				}
				retryCount := st.RetryCount

				// 执行并重试
				res := s.executeStepWithRetries(ctx, st, stepInput, time.Duration(timeoutSec)*time.Second, retryCount)

				stepExec.CompletedAt = time.Now()
				stepExec.Status = res.Status
				stepExec.Error = res.Error
				outputJSON, _ := json.Marshal(res.Output)
				stepExec.OutputJSON = string(outputJSON)
				_ = s.repo.StepExecution.Update(stepExec)

				// 保存结果并将输出合并到全局 outputs（需锁）
				muResults.Lock()
				stepResults = append(stepResults, res)
				muResults.Unlock()

				muOutputs.Lock()
				stepOutputs[st.ID] = res.Output
				muOutputs.Unlock()

				// 处理失败策略
				if res.Status == "failed" {
					switch st.OnError {
					case "skip":
						// do nothing, already marked
					case "stop":
						execution.Status = "failed"
						execErr = fmt.Errorf("步骤 %s 失败: %s", st.ID, res.Error)
					default:
						execution.Status = "failed"
						execErr = fmt.Errorf("步骤 %s 失败: %s", st.ID, res.Error)
					}
				}

				// 标记为已执行
				executed[st.ID] = true
			}(step)
		}
		wg.Wait()

		// 若执行已由于某步失败而被设置为 failed，则停止循环
		if execErr != nil {
			break
		}
	}

	// 汇总最终输出
	if execution.Status == "running" {
		execution.Status = "completed"
		// 找到最后一个步骤的输出作为最终输出
		if len(executionOrder) > 0 {
			lastStepID := executionOrder[len(executionOrder)-1].ID
			if out, ok := stepOutputs[lastStepID]; ok {
				finalOutput = out
			}
		}
	}

	// 更新执行记录
	execution.CompletedAt = time.Now()
	execution.Duration = int(execution.CompletedAt.Sub(execution.StartedAt).Seconds())
	if finalOutput != nil {
		outputJSON, _ := json.Marshal(finalOutput)
		execution.OutputJSON = string(outputJSON)
	}
	if execErr != nil {
		execution.OutputJSON = fmt.Sprintf("{\"error\": \"%s\"}", execErr.Error())
	}
	s.repo.WorkflowExecution.Update(execution)

	log.Printf("工作流执行完成: %s, 状态: %s, 耗时: %d秒", execution.UUID, execution.Status, execution.Duration)
}

// buildExecutionOrder 构建步骤执行顺序
func (s *OrchestrateService) buildExecutionOrder(steps []domain.WorkflowStep) []domain.WorkflowStep {
	// 简单拓扑排序：先执行没有依赖的步骤
	visited := make(map[string]bool)
	order := make([]domain.WorkflowStep, 0, len(steps))

	for len(order) < len(steps) {
		for _, step := range steps {
			if visited[step.ID] {
				continue
			}
			// 检查所有依赖是否已执行
			canExecute := true
			for _, from := range step.InputFrom {
				if from != "input" && !visited[from] {
					canExecute = false
					break
				}
			}
			if canExecute {
				order = append(order, step)
				visited[step.ID] = true
			}
		}
	}

	return order
}

// buildStepInput 构建步骤输入
func (s *OrchestrateService) buildStepInput(step domain.WorkflowStep, stepOutputs map[string]map[string]interface{}, workflowInput map[string]interface{}) map[string]interface{} {
	input := make(map[string]interface{})

	// 从工作流输入获取
	for k, v := range workflowInput {
		input[k] = v
	}

	// 从依赖步骤获取
	for _, from := range step.InputFrom {
		if from == "input" {
			continue
		}
		if output, ok := stepOutputs[from]; ok {
			// 应用映射
			if step.InputMap != nil {
				for srcKey, dstKey := range step.InputMap {
					if val, ok := output[srcKey]; ok {
						input[dstKey] = val
					}
				}
			} else {
				for k, v := range output {
					input[k] = v
				}
			}
		}
	}

	return input
}

// executeStep 执行单个步骤
func (s *OrchestrateService) executeStep(ctx context.Context, step domain.WorkflowStep, input map[string]interface{}) StepResultOutput {
	result := StepResultOutput{
		StepID:  step.ID,
		AgentID: step.AgentID,
	}

	// 构建对话输入
	content := s.formatInputAsPrompt(input)

	chatInput := &ChatInput{
		AgentID: step.AgentID,
		UserID:  0, // 系统执行
		Content: content,
	}

	// 获取Agent配置
	agent, err := s.repo.Agent.GetByID(step.AgentID)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Agent不存在: %v", err)
		return result
	}

	chatInput.VendorKey = agent.ModelProvider
	chatInput.ModelName = agent.ModelName

	// 执行对话（在 goroutine 中运行 HandleChat，并在返回后关闭 channel）
	eventCh := make(chan domain.StreamEvent, 100)
	doneCh := make(chan error, 1)
	go func() {
		err := s.chatService.HandleChat(ctx, chatInput, eventCh)
		doneCh <- err
		close(eventCh)
	}()

	// 收集输出
	var outputContent string
	// 监听事件与结束信号
	for event := range eventCh {
		if event.Type == "text" || event.Type == "content" {
			outputContent += event.Content
		}
	}

	// 等待 HandleChat 的返回结果
	if err = <-doneCh; err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("对话执行失败: %v", err)
		return result
	}

	result.Status = "completed"
	// 解析输出为JSON（如果可能）
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(outputContent), &output); err == nil {
		result.Output = output
	} else {
		result.Output = map[string]interface{}{
			"content": outputContent,
		}
	}

	return result

}

// executeStepWithRetries 支持 per-step 超时与重试
func (s *OrchestrateService) executeStepWithRetries(parentCtx context.Context, step domain.WorkflowStep, input map[string]interface{}, timeout time.Duration, retryCount int) StepResultOutput {
	var lastErr error
	var res StepResultOutput
	for attempt := 0; attempt <= retryCount; attempt++ {
		// 每次使用独立的上下文，以便单步超时可独立触发
		stepCtx, cancel := context.WithTimeout(parentCtx, timeout)
		start := time.Now()
		res = s.executeStep(stepCtx, step, input)
		res.Duration = int(time.Since(start).Seconds())
		cancel()

		if res.Status == "completed" {
			return res
		}
		lastErr = fmt.Errorf("%s", res.Error)
		// 简单退避：等待短时间后重试（可改进为指数退避）
		time.Sleep(500 * time.Millisecond)
	}

	if lastErr != nil {
		res.Status = "failed"
		res.Error = lastErr.Error()
		if res.Output == nil {
			res.Output = map[string]interface{}{}
		}
	}
	return res
}

// evalCondition 简单条件求值，支持形如 "input.foo == 'bar'" 或 "steps.step1.value > 10" 的表达式
func (s *OrchestrateService) evalCondition(cond string, stepOutputs map[string]map[string]interface{}, workflowInput map[string]interface{}) bool {
	// 支持的操作符
	ops := []string{"==", "!=", ">=", "<=", ">", "<"}
	var opFound string
	for _, op := range ops {
		if idx := strings.Index(cond, op); idx >= 0 {
			opFound = op
			break
		}
	}
	if opFound == "" {
		// 无操作符，尝试将整条作为布尔变量存在性判断
		v := s.getVarValue(strings.TrimSpace(cond), stepOutputs, workflowInput)
		if b, ok := v.(bool); ok {
			return b
		}
		return v != nil && v != ""
	}

	parts := strings.SplitN(cond, opFound, 2)
	if len(parts) != 2 {
		return false
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	lval := s.getVarValue(left, stepOutputs, workflowInput)
	rval := s.parseLiteralOrVar(right, stepOutputs, workflowInput)

	// 尝试数值比较
	lf, lerr := toFloat(lval)
	rf, rerr := toFloat(rval)
	if lerr == nil && rerr == nil {
		switch opFound {
		case "==":
			return lf == rf
		case "!=":
			return lf != rf
		case ">":
			return lf > rf
		case "<":
			return lf < rf
		case ">=":
			return lf >= rf
		case "<=":
			return lf <= rf
		}
	}

	// 字符串比较
	ls := fmt.Sprintf("%v", lval)
	rs := fmt.Sprintf("%v", rval)
	switch opFound {
	case "==":
		return ls == rs
	case "!=":
		return ls != rs
	case ">":
		return ls > rs
	case "<":
		return ls < rs
	case ">=":
		return ls >= rs
	case "<=":
		return ls <= rs
	}
	return false
}

func toFloat(v interface{}) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case float32:
		return float64(t), nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case string:
		return strconv.ParseFloat(t, 64)
	default:
		return 0, fmt.Errorf("无法转换为数字")
	}
}

// parseLiteralOrVar 解析右侧是字面量还是变量名
func (s *OrchestrateService) parseLiteralOrVar(token string, stepOutputs map[string]map[string]interface{}, workflowInput map[string]interface{}) interface{} {
	t := strings.TrimSpace(token)
	// 字符串字面量
	if strings.HasPrefix(t, "'") && strings.HasSuffix(t, "'") {
		return strings.Trim(t, "'")
	}
	if strings.HasPrefix(t, "\"") && strings.HasSuffix(t, "\"") {
		return strings.Trim(t, "\"")
	}
	// 布尔字面量
	if t == "true" {
		return true
	}
	if t == "false" {
		return false
	}
	// 数值尝试
	if f, err := strconv.ParseFloat(t, 64); err == nil {
		return f
	}
	// 否则视为变量名
	return s.getVarValue(t, stepOutputs, workflowInput)
}

// getVarValue 支持访问 input.key 或 steps.stepID.key 或 stepID.key
func (s *OrchestrateService) getVarValue(name string, stepOutputs map[string]map[string]interface{}, workflowInput map[string]interface{}) interface{} {
	if name == "input" {
		return workflowInput
	}
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return nil
	}
	if parts[0] == "input" {
		if len(parts) == 1 {
			return workflowInput
		}
		return deepGet(workflowInput, parts[1:])
	}
	if parts[0] == "steps" && len(parts) >= 3 {
		// steps.<stepID>.<key>
		sid := parts[1]
		return deepGet(stepOutputs[sid], parts[2:])
	}
	// 支持直接 stepID.key
	if out, ok := stepOutputs[parts[0]]; ok {
		if len(parts) == 1 {
			return out
		}
		return deepGet(out, parts[1:])
	}
	return nil
}

func deepGet(m map[string]interface{}, path []string) interface{} {
	if m == nil {
		return nil
	}
	cur := m
	for i, p := range path {
		if i == len(path)-1 {
			if v, ok := cur[p]; ok {
				return v
			}
			return nil
		}
		if nxt, ok := cur[p]; ok {
			if nm, ok2 := nxt.(map[string]interface{}); ok2 {
				cur = nm
				continue
			}
			return nil
		}
		return nil
	}
	return nil
}

// formatInputAsPrompt 将输入格式化为提示词
func (s *OrchestrateService) formatInputAsPrompt(input map[string]interface{}) string {
	jsonInput, _ := json.Marshal(input)
	return fmt.Sprintf("请处理以下输入数据并返回结果（JSON格式）：\n%s", string(jsonInput))
}

// GetWorkflowStatus 获取工作流执行状态
func (s *OrchestrateService) GetWorkflowStatus(ctx context.Context, executionID uint) (*ExecutionOutput, error) {
	execution, err := s.repo.WorkflowExecution.GetByID(executionID)
	if err != nil {
		return nil, fmt.Errorf("执行记录不存在: %w", err)
	}

	var input map[string]interface{}
	json.Unmarshal([]byte(execution.InputJSON), &input)

	var output map[string]interface{}
	json.Unmarshal([]byte(execution.OutputJSON), &output)

	// 获取步骤执行结果
	stepExecs, err := s.repo.StepExecution.ListByExecution(executionID)
	if err == nil {
		var stepResults []StepResultOutput
		for _, se := range stepExecs {
			var stepOutput map[string]interface{}
			json.Unmarshal([]byte(se.OutputJSON), &stepOutput)
			stepResults = append(stepResults, StepResultOutput{
				StepID:   se.StepID,
				AgentID:  se.AgentID,
				Status:   se.Status,
				Output:   stepOutput,
				Error:    se.Error,
				Duration: int(se.CompletedAt.Sub(se.StartedAt).Seconds()),
			})
		}
		return &ExecutionOutput{
			ID:          execution.ID,
			UUID:        execution.UUID,
			WorkflowID:  execution.WorkflowID,
			Status:      execution.Status,
			Input:       input,
			Output:      output,
			StepResults: stepResults,
			StartedAt:   execution.StartedAt,
			CompletedAt: execution.CompletedAt,
			Duration:    execution.Duration,
		}, nil
	}

	return &ExecutionOutput{
		ID:          execution.ID,
		UUID:        execution.UUID,
		WorkflowID:  execution.WorkflowID,
		Status:      execution.Status,
		Input:       input,
		Output:      output,
		StartedAt:   execution.StartedAt,
		CompletedAt: execution.CompletedAt,
		Duration:    execution.Duration,
	}, nil
}

// StopWorkflow 停止工作流执行
func (s *OrchestrateService) StopWorkflow(ctx context.Context, executionID uint, userID uint) error {
	execution, err := s.repo.WorkflowExecution.GetByID(executionID)
	if err != nil {
		return fmt.Errorf("执行记录不存在: %w", err)
	}
	if execution.UserID != userID {
		return fmt.Errorf("无权限停止此执行")
	}
	if execution.Status != "running" && execution.Status != "pending" {
		return fmt.Errorf("执行已完成，无法停止")
	}

	s.executionsMu.Lock()
	cancel, ok := s.activeExecutions[executionID]
	s.executionsMu.Unlock()

	if ok {
		cancel()
	}

	execution.Status = "cancelled"
	execution.CompletedAt = time.Now()
	s.repo.WorkflowExecution.Update(execution)

	return nil
}

// workflowToOutput 转换工作流到输出格式
func (s *OrchestrateService) workflowToOutput(workflow *domain.Workflow, steps []domain.WorkflowStep) *WorkflowOutput {
	return &WorkflowOutput{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Steps:       steps,
		InputSchema: workflow.InputSchema,
		Status:      workflow.Status,
		CreatedAt:   workflow.CreatedAt,
	}
}
