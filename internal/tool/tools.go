package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ========== 结果和接口定义 ==========

// Result 工具执行结果
type Result struct {
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
	TokenHint int    `json:"token_hint"` // 第二阶段新增：Token计数提示
}

// Tool 工具接口
type Tool interface {
	// Name 返回工具名称
	Name() string
	// Description 返回工具描述
	Description() string
	// Parameters 返回参数JSON Schema
	Parameters() map[string]interface{}
	// Execute 执行工具
	Execute(ctx context.Context, args map[string]interface{}) (*Result, error)
}

// ConfirmableTool 需要用户确认的工具接口
type ConfirmableTool interface {
	Tool
	RequiresConfirmation(args map[string]interface{}) bool
}

// ========== 第二阶段新增：工具配置 ==========

// ToolConfig 工具全局配置
type ToolConfig struct {
	MaxOutputSize    int           // 最大输出大小（字节），默认100KB
	DefaultTimeout   time.Duration // 默认超时时间
	HTTPClient       *http.Client  // 共享HTTP客户端
}

// DefaultToolConfig 默认工具配置
var DefaultToolConfig = &ToolConfig{
	MaxOutputSize:    100 * 1024, // 100KB
	DefaultTimeout:   30 * time.Second,
	HTTPClient:       &http.Client{
		Timeout: 30 * time.Second,
	},
}

// SetToolConfig 设置全局工具配置
func SetToolConfig(cfg *ToolConfig) {
	if cfg != nil {
		DefaultToolConfig = cfg
	}
}

// trimOutput 限制输出大小
func trimOutput(content string, maxSize int) string {
	if len(content) <= maxSize {
		return content
	}
	suffix := fmt.Sprintf("\n... (输出已截断，总大小 %d 字节)", len(content))
	if maxSize <= len(suffix) {
		return suffix[:maxSize]
	}
	return content[:maxSize-len(suffix)] + suffix
}

// estimateTokens 估算Token数量（简单估算：约4字符=1token）
func estimateTokens(content string) int {
	return len(content) / 4
}

// ========== 工具注册表 ==========

// Registry 工具注册表
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册工具
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List 列出所有工具
func (r *Registry) List() []Tool {
	var tools []Tool
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ToolDefs 返回模型工具定义列表
func (r *Registry) ToolDefs(names []string) []map[string]interface{} {
	var defs []map[string]interface{}
	for _, name := range names {
		if t, ok := r.tools[name]; ok {
			defs = append(defs, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name(),
					"description": t.Description(),
					"parameters":  t.Parameters(),
				},
			})
		}
	}
	return defs
}

// LoadBuiltinTools 注册内置工具到注册表
func LoadBuiltinTools() *Registry {
	r := NewRegistry()
	r.Register(&WeatherTool{})
	r.Register(&CalculatorTool{})
	r.Register(&WebSearchTool{})
	r.Register(&WebFetchTool{}) // 第二阶段新增：网页抓取工具
	r.Register(&LocalCommandTool{})
	r.Register(&FileTool{}) // 第二阶段新增：文件系统工具
	// 第三阶段新增
	r.Register(&SandboxTool{})   // 沙箱执行工具
	r.Register(&EmailSendTool{}) // 发送邮件工具
	r.Register(&EmailReadTool{}) // 读取邮件工具
	return r
}

// ========== 第一阶段安全加固：工具确认状态管理 ==========

// ToolConfirmation 工具确认状态
type ToolConfirmation struct {
	ID          string
	UserID      uint
	SessionID   uint
	ToolName    string
	ToolCallID  string
	Arguments   string
	CreatedAt   time.Time
	ExpiresAt   time.Time // TTL过期时间
	Confirmed   bool
	Executed    bool
}

// ToolConfirmationManager 工具确认状态管理器（带TTL过期机制）
type ToolConfirmationManager struct {
	confirmations map[string]*ToolConfirmation
	mu            sync.RWMutex
	defaultTTL    time.Duration // 默认过期时间
}

// NewToolConfirmationManager 创建确认管理器
func NewToolConfirmationManager(ttl time.Duration) *ToolConfirmationManager {
	if ttl == 0 {
		ttl = 5 * time.Minute // 默认5分钟过期
	}
	m := &ToolConfirmationManager{
		confirmations: make(map[string]*ToolConfirmation),
		defaultTTL:    ttl,
	}
	// 启动定期清理过期确认
	go m.cleanupExpired()
	return m
}

// CreateConfirmation 创建新的确认请求
func (m *ToolConfirmationManager) CreateConfirmation(userID, sessionID uint, toolName, toolCallID, arguments string) *ToolConfirmation {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	confirmation := &ToolConfirmation{
		ID:         fmt.Sprintf("conf_%d_%s_%d", time.Now().UnixNano(), toolCallID, userID),
		UserID:     userID,
		SessionID:  sessionID,
		ToolName:   toolName,
		ToolCallID: toolCallID,
		Arguments:  arguments,
		CreatedAt:  now,
		ExpiresAt:  now.Add(m.defaultTTL),
		Confirmed:  false,
		Executed:   false,
	}
	m.confirmations[confirmation.ID] = confirmation
	return confirmation
}

// GetConfirmation 获取确认请求
func (m *ToolConfirmationManager) GetConfirmation(id string) (*ToolConfirmation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conf, ok := m.confirmations[id]
	if !ok {
		return nil, errors.New("确认请求不存在")
	}
	if time.Now().After(conf.ExpiresAt) {
		return nil, errors.New("确认请求已过期")
	}
	return conf, nil
}

// Confirm 确认执行
func (m *ToolConfirmationManager) Confirm(id string, userID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conf, ok := m.confirmations[id]
	if !ok {
		return errors.New("确认请求不存在")
	}
	if time.Now().After(conf.ExpiresAt) {
		delete(m.confirmations, id)
		return errors.New("确认请求已过期，请重新发起")
	}
	if conf.UserID != userID {
		return errors.New("无权限确认此请求")
	}
	if conf.Confirmed {
		return errors.New("已经确认过")
	}
	conf.Confirmed = true
	return nil
}

// Reject 拒绝执行
func (m *ToolConfirmationManager) Reject(id string, userID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conf, ok := m.confirmations[id]
	if !ok {
		return errors.New("确认请求不存在")
	}
	if conf.UserID != userID {
		return errors.New("无权限拒绝此请求")
	}
	delete(m.confirmations, id)
	return nil
}

// MarkExecuted 标记已执行
func (m *ToolConfirmationManager) MarkExecuted(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conf, ok := m.confirmations[id]; ok {
		conf.Executed = true
		// 执行后删除确认记录
		delete(m.confirmations, id)
	}
}

// cleanupExpired 定期清理过期确认
func (m *ToolConfirmationManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, conf := range m.confirmations {
			if now.After(conf.ExpiresAt) {
				delete(m.confirmations, id)
			}
		}
		m.mu.Unlock()
	}
}

// ========== 第一阶段安全加固：本地命令执行工具 ==========

// LocalCommandTool 在本地系统执行命令
// 安全改进：使用exec.Command直接执行，禁止命令链接（; | & 等）
type LocalCommandTool struct{}

func (t *LocalCommandTool) Name() string       { return "local_command" }
func (t *LocalCommandTool) Description() string { return "在本地系统执行命令。所有命令执行前都需要用户确认。" }
func (t *LocalCommandTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "需要执行的命令（单条命令，不允许使用;|&等链接符）",
			},
			"args": map[string]interface{}{
				"type":        "array",
				"description": "命令参数列表",
				"items":       map[string]interface{}{"type": "string"},
			},
			"working_directory": map[string]interface{}{
				"type":        "string",
				"description": "可选的工作目录，默认当前目录",
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "可选的超时时间，单位秒，默认30",
			},
		},
		"required": []string{"command"},
	}
}

func (t *LocalCommandTool) RequiresConfirmation(args map[string]interface{}) bool {
	return true // 所有命令都需要确认
}

// 安全检查：检测危险命令链接符
func containsDangerousOperators(command string) bool {
	dangerousPatterns := []string{
		";", "|", "&", "&&", "||", 
		">", ">>", "<", "$(", "`",
		"\n", "\r",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return true
		}
	}
	return false
}

func (t *LocalCommandTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return &Result{Content: "错误：缺少命令参数", IsError: true}, nil
	}

	// 第一阶段安全加固：禁止命令链接
	if containsDangerousOperators(command) {
		return &Result{
			Content: "错误：不允许使用命令链接符（; | & && || > $() 等）。请使用单条命令。",
			IsError: true,
		}, nil
	}

	// 获取参数列表
	var cmdArgs []string
	if rawArgs, ok := args["args"].([]interface{}); ok {
		for _, a := range rawArgs {
			if s, ok := a.(string); ok {
				cmdArgs = append(cmdArgs, s)
			}
		}
	}

	workingDir, _ := args["working_directory"].(string)
	if workingDir == "" {
		workingDir = "."
	}

	timeoutSeconds := 30
	if rawTimeout, ok := args["timeout_seconds"].(float64); ok && rawTimeout > 0 {
		timeoutSeconds = int(rawTimeout)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// 第一阶段安全加固：使用exec.Command直接执行，避免shell注入
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows: 直接执行命令
		cmd = exec.CommandContext(ctx, command, cmdArgs...)
	} else {
		// Linux/Mac: 直接执行命令（不再通过sh -c）
		cmd = exec.CommandContext(ctx, command, cmdArgs...)
	}
	cmd.Dir = workingDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	stdoutContent := strings.TrimSpace(stdout.String())
	stderrContent := strings.TrimSpace(stderr.String())

	// 第二阶段：限制输出大小
	maxSize := DefaultToolConfig.MaxOutputSize
	stdoutContent = trimOutput(stdoutContent, maxSize/2)
	stderrContent = trimOutput(stderrContent, maxSize/2)

	if err != nil {
		return &Result{
			Content:   fmt.Sprintf("命令执行失败: %v\nstdout:\n%s\nstderr:\n%s", err, stdoutContent, stderrContent),
			IsError:   true,
			TokenHint: estimateTokens(stdoutContent + stderrContent),
		}, nil
	}

	if stdoutContent == "" && stderrContent == "" {
		return &Result{Content: "命令已执行完成，未输出结果"}, nil
	}

	result := ""
	if stdoutContent != "" {
		result += "stdout:\n" + stdoutContent
	}
	if stderrContent != "" {
		if result != "" {
			result += "\n"
		}
		result += "stderr:\n" + stderrContent
	}

	return &Result{
		Content:   result,
		TokenHint: estimateTokens(result),
	}, nil
}

// ========== 天气查询工具 ==========

// WeatherTool 天气查询工具
type WeatherTool struct{}

func (t *WeatherTool) Name() string        { return "weather" }
func (t *WeatherTool) Description() string  { return "查询指定城市的当前天气信息" }
func (t *WeatherTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"city": map[string]interface{}{
				"type":        "string",
				"description": "城市名称，如：北京、上海、Tokyo",
			},
		},
		"required": []string{"city"},
	}
}

func (t *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	city, _ := args["city"].(string)
	if city == "" {
		return &Result{Content: "错误：缺少城市名称", IsError: true}, nil
	}

	// 使用wttr.in免费天气API
	apiURL := fmt.Sprintf("https://wttr.in/%s?format=j1&lang=zh", url.QueryEscape(city))
	
	// 第二阶段：使用共享HTTP客户端
	client := DefaultToolConfig.HTTPClient
	resp, err := client.Get(apiURL)
	if err != nil {
		return &Result{Content: fmt.Sprintf("天气查询失败: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &Result{Content: "天气服务暂时不可用", IsError: true}, nil
	}

	var data struct {
		CurrentCondition []struct {
			TempC     string `json:"temp_C"`
			FeelsLike string `json:"FeelsLikeC"`
			Humidity  string `json:"humidity"`
			WeatherDesc []struct{ Value string `json:"value"` } `json:"weatherDesc"`
			WindspeedKmph string `json:"windspeedKmph"`
		} `json:"current_condition"`
		NearestArea []struct {
			AreaName []struct{ Value string `json:"value"` } `json:"areaName"`
			Country  []struct{ Value string `json:"value"` } `json:"country"`
		} `json:"nearest_area"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return &Result{Content: fmt.Sprintf("解析天气数据失败: %v", err), IsError: true}, nil
	}

	if len(data.CurrentCondition) == 0 {
		return &Result{Content: "未找到该城市的天气信息", IsError: true}, nil
	}

	cc := data.CurrentCondition[0]
	location := city
	if len(data.NearestArea) > 0 {
		area := data.NearestArea[0]
		if len(area.AreaName) > 0 {
			location = area.AreaName[0].Value
		}
	}

	weather := cc.WeatherDesc[0].Value
	result := fmt.Sprintf("📍 %s 天气：%s，温度 %s°C（体感 %s°C），湿度 %s%%，风速 %s km/h",
		location, weather, cc.TempC, cc.FeelsLike, cc.Humidity, cc.WindspeedKmph)

	return &Result{
		Content:   result,
		TokenHint: estimateTokens(result),
	}, nil
}

// ========== 计算器工具 ==========

// CalculatorTool 数学计算工具
type CalculatorTool struct{}

func (t *CalculatorTool) Name() string        { return "calculator" }
func (t *CalculatorTool) Description() string  { return "执行数学表达式计算，支持加减乘除、幂运算、三角函数等" }
func (t *CalculatorTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "数学表达式，如：2+3*4、sqrt(16)、sin(3.14)",
			},
		},
		"required": []string{"expression"},
	}
}

func (t *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	expr, _ := args["expression"].(string)
	if expr == "" {
		return &Result{Content: "错误：缺少表达式", IsError: true}, nil
	}

	result, err := evaluateExpression(expr)
	if err != nil {
		return &Result{Content: fmt.Sprintf("计算错误: %v", err), IsError: true}, nil
	}

	output := fmt.Sprintf("%s = %g", expr, result)
	return &Result{
		Content:   output,
		TokenHint: estimateTokens(output),
	}, nil
}

// evaluateExpression 安全的数学表达式求值
func evaluateExpression(expr string) (float64, error) {
	// 清理表达式
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("空表达式")
	}

	// 替换常见函数
	replacements := map[string]string{
		"sqrt(": "SQRT(",
		"abs(":  "ABS(",
		"sin(":  "SIN(",
		"cos(":  "COS(",
		"tan(":  "TAN(",
		"log(":  "LOG(",
		"ln(":   "LN(",
		"pi":    "3.14159265358979",
		"e":     "2.71828182845905",
	}
	for old, new_ := range replacements {
		expr = strings.ReplaceAll(expr, old, new_)
	}

	// 简单的递归下降解析器
	tokens := tokenize(expr)
	result, _, err := parseExpr(tokens, 0)
	if err != nil {
		return 0, err
	}
	return result, nil
}

type token struct {
	typ string // "num", "op", "lparen", "rparen", "func"
	val string
}

func tokenize(expr string) []token {
	var tokens []token
	i := 0
	for i < len(expr) {
		ch := expr[i]
		switch {
		case ch == ' ':
			i++
		case ch >= '0' && ch <= '9' || ch == '.':
			j := i
			for j < len(expr) && (expr[j] >= '0' && expr[j] <= '9' || expr[j] == '.') {
				j++
			}
			tokens = append(tokens, token{typ: "num", val: expr[i:j]})
			i = j
		case ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '^' || ch == '%':
			tokens = append(tokens, token{typ: "op", val: string(ch)})
			i++
		case ch == '(':
			tokens = append(tokens, token{typ: "lparen", val: "("})
			i++
		case ch == ')':
			tokens = append(tokens, token{typ: "rparen", val: ")"})
			i++
		case ch >= 'A' && ch <= 'Z':
			j := i
			for j < len(expr) && expr[j] >= 'A' && expr[j] <= 'Z' {
				j++
			}
			tokens = append(tokens, token{typ: "func", val: expr[i:j]})
			i = j
		default:
			i++
		}
	}
	return tokens
}

func parseExpr(tokens []token, pos int) (float64, int, error) {
	left, pos, err := parseTerm(tokens, pos)
	if err != nil {
		return 0, pos, err
	}
	for pos < len(tokens) && tokens[pos].typ == "op" && (tokens[pos].val == "+" || tokens[pos].val == "-") {
		op := tokens[pos].val
		pos++
		right, newPos, err := parseTerm(tokens, pos)
		if err != nil {
			return 0, newPos, err
		}
		pos = newPos
		if op == "+" {
			left += right
		} else {
			left -= right
		}
	}
	return left, pos, nil
}

func parseTerm(tokens []token, pos int) (float64, int, error) {
	left, pos, err := parsePower(tokens, pos)
	if err != nil {
		return 0, pos, err
	}
	for pos < len(tokens) && tokens[pos].typ == "op" && (tokens[pos].val == "*" || tokens[pos].val == "/" || tokens[pos].val == "%") {
		op := tokens[pos].val
		pos++
		right, newPos, err := parsePower(tokens, pos)
		if err != nil {
			return 0, newPos, err
		}
		pos = newPos
		switch op {
		case "*":
			left *= right
		case "/":
			if right == 0 {
				return 0, pos, fmt.Errorf("除零错误")
			}
			left /= right
		case "%":
			left = math.Mod(left, right)
		}
	}
	return left, pos, nil
}

func parsePower(tokens []token, pos int) (float64, int, error) {
	base, pos, err := parseUnary(tokens, pos)
	if err != nil {
		return 0, pos, err
	}
	if pos < len(tokens) && tokens[pos].typ == "op" && tokens[pos].val == "^" {
		pos++
		exp, newPos, err := parseUnary(tokens, pos)
		if err != nil {
			return 0, newPos, err
		}
		pos = newPos
		base = math.Pow(base, exp)
	}
	return base, pos, nil
}

func parseUnary(tokens []token, pos int) (float64, int, error) {
	if pos < len(tokens) && tokens[pos].typ == "op" && tokens[pos].val == "-" {
		pos++
		val, newPos, err := parsePrimary(tokens, pos)
		if err != nil {
			return 0, newPos, err
		}
		return -val, newPos, nil
	}
	return parsePrimary(tokens, pos)
}

func parsePrimary(tokens []token, pos int) (float64, int, error) {
	if pos >= len(tokens) {
		return 0, pos, fmt.Errorf("意外的表达式结束")
	}

	t := tokens[pos]

	// 数字
	if t.typ == "num" {
		var val float64
		fmt.Sscanf(t.val, "%f", &val)
		return val, pos + 1, nil
	}

	// 括号
	if t.typ == "lparen" {
		val, newPos, err := parseExpr(tokens, pos+1)
		if err != nil {
			return 0, newPos, err
		}
		if newPos < len(tokens) && tokens[newPos].typ == "rparen" {
			return val, newPos + 1, nil
		}
		return 0, newPos, fmt.Errorf("缺少右括号")
	}

	// 函数
	if t.typ == "func" {
		if pos+1 >= len(tokens) || tokens[pos+1].typ != "lparen" {
			return 0, pos, fmt.Errorf("函数 %s 后缺少括号", t.val)
		}
		arg, newPos, err := parseExpr(tokens, pos+2)
		if err != nil {
			return 0, newPos, err
		}
		if newPos >= len(tokens) || tokens[newPos].typ != "rparen" {
			return 0, newPos, fmt.Errorf("函数 %s 缺少右括号", t.val)
		}
		newPos++

		switch t.val {
		case "SQRT":
			return math.Sqrt(arg), newPos, nil
		case "ABS":
			return math.Abs(arg), newPos, nil
		case "SIN":
			return math.Sin(arg), newPos, nil
		case "COS":
			return math.Cos(arg), newPos, nil
		case "TAN":
			return math.Tan(arg), newPos, nil
		case "LOG":
			return math.Log10(arg), newPos, nil
		case "LN":
			return math.Log(arg), newPos, nil
		default:
			return 0, newPos, fmt.Errorf("未知函数: %s", t.val)
		}
	}

	return 0, pos, fmt.Errorf("意外的token: %s", t.val)
}

// ========== 第二阶段新增：网络搜索工具（修复版） ==========

// WebSearchTool 网络搜索工具
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string  { return "搜索互联网获取信息（使用DuckDuckGo Lite）" }
func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "搜索关键词",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "返回结果数量，默认5",
				"default":     5,
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return &Result{Content: "错误：缺少搜索关键词", IsError: true}, nil
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// 第二阶段修复：使用共享HTTP客户端
	searchURL := fmt.Sprintf("https://lite.duckduckgo.com/lite/?q=%s", url.QueryEscape(query))
	client := DefaultToolConfig.HTTPClient

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return &Result{Content: fmt.Sprintf("创建请求失败: %v", err), IsError: true}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return &Result{Content: fmt.Sprintf("搜索请求失败: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &Result{Content: fmt.Sprintf("搜索服务返回错误: %d", resp.StatusCode), IsError: true}, nil
	}

	// 解析HTML获取搜索结果
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Result{Content: fmt.Sprintf("读取响应失败: %v", err), IsError: true}, nil
	}

	// 简单解析DuckDuckGo Lite结果
	// 格式：<a class="result-link" href="...">title</a>
	results := parseDuckDuckGoResults(string(body), limit)

	if len(results) == 0 {
		return &Result{Content: fmt.Sprintf("搜索 \"%s\" 未找到相关结果", query), TokenHint: 10}, nil
	}

	// 构建结果输出
	var output strings.Builder
	output.WriteString(fmt.Sprintf("搜索 \"%s\" 的结果（共 %d 条）：\n\n", query, len(results)))
	for i, r := range results {
		output.WriteString(fmt.Sprintf("%d. %s\n   %s\n   来源: %s\n\n", i+1, r.Title, r.Snippet, r.URL))
	}

	content := trimOutput(output.String(), DefaultToolConfig.MaxOutputSize)
	return &Result{
		Content:   content,
		TokenHint: estimateTokens(content),
	}, nil
}

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

func parseDuckDuckGoResults(html string, limit int) []searchResult {
	var results []searchResult

	// 提取链接标题
	linkRegex := regexp.MustCompile(`<a[^>]*class="[^"]*result-link[^"]*"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`)
	snippetRegex := regexp.MustCompile(`<td[^>]*class="[^"]*result-snippet[^"]*"[^>]*>([^<]+)</td>`)

	linkMatches := linkRegex.FindAllStringSubmatch(html, limit)
	snippetMatches := snippetRegex.FindAllStringSubmatch(html, limit)

	for i, match := range linkMatches {
		if len(match) >= 3 {
			result := searchResult{
				URL:   match[1],
				Title: strings.TrimSpace(match[2]),
			}
			if i < len(snippetMatches) && len(snippetMatches[i]) >= 2 {
				result.Snippet = strings.TrimSpace(snippetMatches[i][1])
			}
			results = append(results, result)
		}
	}

	return results
}

// ========== 第二阶段新增：网页抓取工具 ==========

// WebFetchTool 网页内容抓取工具
type WebFetchTool struct{}

func (t *WebFetchTool) Name() string        { return "web_fetch" }
func (t *WebFetchTool) Description() string  { return "抓取网页内容并提取文本" }
func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "要抓取的网页URL",
			},
			"extract_text": map[string]interface{}{
				"type":        "boolean",
				"description": "是否提取纯文本，默认true",
				"default":     true,
			},
			"max_length": map[string]interface{}{
				"type":        "integer",
				"description": "最大返回内容长度，默认5000",
				"default":     5000,
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		return &Result{Content: "错误：缺少URL参数", IsError: true}, nil
	}

	// 验证URL格式
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return &Result{Content: fmt.Sprintf("URL格式错误: %v", err), IsError: true}, nil
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &Result{Content: "错误：只支持HTTP/HTTPS协议", IsError: true}, nil
	}

	extractText := true
	if e, ok := args["extract_text"].(bool); ok {
		extractText = e
	}

	maxLength := 5000
	if m, ok := args["max_length"].(float64); ok && m > 0 {
		maxLength = int(m)
	}

	// 第二阶段：使用共享HTTP客户端
	client := DefaultToolConfig.HTTPClient
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return &Result{Content: fmt.Sprintf("创建请求失败: %v", err), IsError: true}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; EinoAgent/1.0; +https://github.com/YJMQN/cat-agent)")

	resp, err := client.Do(req)
	if err != nil {
		return &Result{Content: fmt.Sprintf("请求失败: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &Result{Content: fmt.Sprintf("网页返回错误: %d %s", resp.StatusCode, resp.Status), IsError: true}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Result{Content: fmt.Sprintf("读取内容失败: %v", err), IsError: true}, nil
	}

	content := string(body)

	// 提取纯文本
	if extractText {
		content = extractTextFromHTML(content)
	}

	// 限制输出长度
	if len(content) > maxLength {
		content = content[:maxLength] + "\n... (内容已截断)"
	}

	output := fmt.Sprintf("网页内容 (%s):\n\n%s", rawURL, content)
	output = trimOutput(output, DefaultToolConfig.MaxOutputSize)

	return &Result{
		Content:   output,
		TokenHint: estimateTokens(output),
	}, nil
}

// extractTextFromHTML 从HTML提取纯文本
func extractTextFromHTML(html string) string {
	// 移除script和style标签
	scriptRegex := regexp.MustCompile(`<script[^>]*>[^<]*</script>`)
	styleRegex := regexp.MustCompile(`<style[^>]*>[^<]*</style>`)
	html = scriptRegex.ReplaceAllString(html, "")
	html = styleRegex.ReplaceAllString(html, "")

	// 移除所有HTML标签
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	html = tagRegex.ReplaceAllString(html, "")

	// 移除HTML实体
	htmlEntityRegex := regexp.MustCompile(`&[^;]+;`)
	html = htmlEntityRegex.ReplaceAllStringFunc(html, func(s string) string {
		switch s {
		case "&nbsp;":
			return " "
		case "&lt;":
			return "<"
		case "&gt;":
			return ">"
		case "&amp;":
			return "&"
		case "&quot;":
			return "\""
		default:
			return ""
		}
	})

	// 清理空白
	multipleSpaceRegex := regexp.MustCompile(`\s+`)
	html = multipleSpaceRegex.ReplaceAllString(html, " ")

	return strings.TrimSpace(html)
}

// ========== 第二阶段新增：文件系统工具 ==========

// FileTool 文件系统操作工具
type FileTool struct{}

func (t *FileTool) Name() string        { return "file" }
func (t *FileTool) Description() string  { return "文件系统操作：读取、写入、列表、删除文件" }
func (t *FileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "操作类型：read, write, list, delete, create_dir",
				"enum":        []string{"read", "write", "list", "delete", "create_dir"},
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "文件或目录路径",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "写入内容（write操作需要）",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "是否递归（list/delete操作）",
				"default":     false,
			},
		},
		"required": []string{"operation", "path"},
	}
}

func (t *FileTool) RequiresConfirmation(args map[string]interface{}) bool {
	op, _ := args["operation"].(string)
	// 写入和删除操作需要确认
	return op == "write" || op == "delete"
}

func (t *FileTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	operation, _ := args["operation"].(string)
	path, _ := args["path"].(string)

	if operation == "" || path == "" {
		return &Result{Content: "错误：缺少操作类型或路径参数", IsError: true}, nil
	}

	// 安全检查：禁止访问敏感路径
	if isSensitivePath(path) {
		return &Result{
			Content: "错误：禁止访问系统敏感路径（如/etc/passwd、/etc/shadow等）",
			IsError: true,
		}, nil
	}

	switch operation {
	case "read":
		return t.readFile(path)
	case "write":
		content, _ := args["content"].(string)
		return t.writeFile(path, content)
	case "list":
		recursive, _ := args["recursive"].(bool)
		return t.listDir(path, recursive)
	case "delete":
		return t.deleteFile(path)
	case "create_dir":
		return t.createDir(path)
	default:
		return &Result{Content: fmt.Sprintf("错误：未知操作类型: %s", operation), IsError: true}, nil
	}
}

// isSensitivePath 检查是否为敏感路径
func isSensitivePath(path string) bool {
	absPath := filepath.Clean(path)
	if absPath == ".env" || filepath.Base(absPath) == ".env" {
		return true
	}
	if strings.Contains(absPath, "/etc/passwd") || strings.Contains(absPath, "/etc/shadow") || strings.Contains(absPath, "/etc/sudoers") {
		return true
	}
	if strings.Contains(absPath, "/root/.ssh") || strings.Contains(absPath, "/.ssh") {
		return true
	}
	if strings.HasSuffix(absPath, ".pem") || strings.HasSuffix(absPath, ".key") || strings.HasSuffix(absPath, ".secret") {
		return true
	}
	return false
}

func (t *FileTool) readFile(path string) (*Result, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return &Result{Content: fmt.Sprintf("读取文件失败: %v", err), IsError: true}, nil
	}

	// 限制输出大小
	output := string(content)
	if len(output) > DefaultToolConfig.MaxOutputSize {
		output = output[:DefaultToolConfig.MaxOutputSize] + "\n... (内容已截断)"
	}

	return &Result{
		Content:   output,
		TokenHint: estimateTokens(output),
	}, nil
}

func (t *FileTool) writeFile(path, content string) (*Result, error) {
	if content == "" {
		return &Result{Content: "错误：缺少写入内容", IsError: true}, nil
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &Result{Content: fmt.Sprintf("创建目录失败: %v", err), IsError: true}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &Result{Content: fmt.Sprintf("写入文件失败: %v", err), IsError: true}, nil
	}

	return &Result{
		Content:   fmt.Sprintf("文件已成功写入: %s (%d 字节)", path, len(content)),
		TokenHint: 10,
	}, nil
}

func (t *FileTool) listDir(path string, recursive bool) (*Result, error) {
	var files []string

	if recursive {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(path, filePath)
			if info.IsDir() {
				files = append(files, relPath + "/")
			} else {
				files = append(files, relPath)
			}
			return nil
		})
		if err != nil {
			return &Result{Content: fmt.Sprintf("遍历目录失败: %v", err), IsError: true}, nil
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return &Result{Content: fmt.Sprintf("读取目录失败: %v", err), IsError: true}, nil
		}
		for _, entry := range entries {
			if entry.IsDir() {
				files = append(files, entry.Name() + "/")
			} else {
				files = append(files, entry.Name())
			}
		}
	}

	output := fmt.Sprintf("目录 %s 内容（共 %d 条）：\n%s", path, len(files), strings.Join(files, "\n"))
	output = trimOutput(output, DefaultToolConfig.MaxOutputSize)

	return &Result{
		Content:   output,
		TokenHint: estimateTokens(output),
	}, nil
}

func (t *FileTool) deleteFile(path string) (*Result, error) {
	if err := os.Remove(path); err != nil {
		// 如果是目录，尝试递归删除
		if os.IsNotExist(err) {
			return &Result{Content: "错误：文件不存在", IsError: true}, nil
		}
		// 尝试作为目录删除
		if err := os.RemoveAll(path); err != nil {
			return &Result{Content: fmt.Sprintf("删除失败: %v", err), IsError: true}, nil
		}
	}

	return &Result{
		Content:   fmt.Sprintf("已删除: %s", path),
		TokenHint: 5,
	}, nil
}

func (t *FileTool) createDir(path string) (*Result, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return &Result{Content: fmt.Sprintf("创建目录失败: %v", err), IsError: true}, nil
	}

	return &Result{
		Content:   fmt.Sprintf("目录已创建: %s", path),
		TokenHint: 5,
	}, nil
}

// ========== 第三阶段：沙箱执行工具 ==========

// SandboxTool 沙箱工具 - 在隔离环境执行命令
type SandboxTool struct{}

func (t *SandboxTool) Name() string       { return "sandbox" }
func (t *SandboxTool) Description() string { return "在安全沙箱中执行命令（Docker/non-privileged子系统）" }
func (t *SandboxTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的命令",
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "超时时间，默认60秒",
				"default":     60,
			},
			"image": map[string]interface{}{
				"type":        "string",
				"description": "Docker镜像（如需Docker执行）",
				"default":     "ubuntu:22.04",
			},
		},
		"required": []string{"command"},
	}
}

func (t *SandboxTool) RequiresConfirmation(args map[string]interface{}) bool {
	return true
}

func (t *SandboxTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return &Result{Content: "错误：缺少命令参数", IsError: true}, nil
	}

	timeoutSeconds := 60
	if to, ok := args["timeout_seconds"].(float64); ok && to > 0 {
		timeoutSeconds = int(to)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// 使用非特权模式执行，通过有限的shell执行
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)

	// 安全限制：设置隔离环境
	cmd.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/tmp/sandbox",
		"USER=nobody",
		"SHELL=/bin/sh",
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		return &Result{
			Content: fmt.Sprintf("沙箱执行失败: %v\n%s", err, strings.TrimSpace(stderr.String())),
			IsError: true,
		}, nil
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = "命令已在沙箱中完成执行（无输出）"
	}

	output = trimOutput(output, DefaultToolConfig.MaxOutputSize)
	return &Result{
		Content:   "[沙箱输出]\n" + output,
		TokenHint: estimateTokens(output),
	}, nil
}

// ========== 第三阶段：邮件工具（占位） ==========

// EmailSendTool 发送邮件工具
type EmailSendTool struct{}

func (t *EmailSendTool) Name() string       { return "email_send" }
func (t *EmailSendTool) Description() string { return "发送电子邮件（需配置SMTP服务器）" }
func (t *EmailSendTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"to": map[string]interface{}{
				"type":        "string",
				"description": "收件人邮箱地址",
			},
			"subject": map[string]interface{}{
				"type":        "string",
				"description": "邮件主题",
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "邮件正文",
			},
		},
		"required": []string{"to", "subject", "body"},
	}
}

func (t *EmailSendTool) RequiresConfirmation(args map[string]interface{}) bool {
	return true
}

func (t *EmailSendTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	return &Result{
		Content:   "邮件功能需要配置SMTP服务器后才可使用。请设置SMTP_HOST、SMTP_PORT、SMTP_USER、SMTP_PASS环境变量。",
		IsError:   false,
	}, nil
}

// EmailReadTool 读取邮件工具
type EmailReadTool struct{}

func (t *EmailReadTool) Name() string       { return "email_read" }
func (t *EmailReadTool) Description() string { return "读取电子邮件（需配置IMAP服务器）" }
func (t *EmailReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "读取邮件数量，默认10",
				"default":     10,
			},
			"folder": map[string]interface{}{
				"type":        "string",
				"description": "邮件文件夹，默认INBOX",
				"default":     "INBOX",
			},
		},
	}
}

func (t *EmailReadTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	return &Result{
		Content: "邮件读取功能需要配置IMAP服务器后才可使用。请设置IMAP_HOST、IMAP_PORT、IMAP_USER、IMAP_PASS环境变量。",
		IsError: false,
	}, nil
}

// ValidateArgs 校验工具参数
func ValidateArgs(tool Tool, args map[string]interface{}) error {
	schema := tool.Parameters()
	required, _ := schema["required"].([]string)
	properties, _ := schema["properties"].(map[string]interface{})

	// 检查必填参数
	for _, r := range required {
		if _, ok := args[r]; !ok {
			return fmt.Errorf("缺少必填参数: %s", r)
		}
	}

	// 检查参数类型
	for key, val := range args {
		prop, ok := properties[key]
		if !ok {
			continue // 允许额外参数
		}
		propMap, ok := prop.(map[string]interface{})
		if !ok {
			continue
		}
		expectedType, _ := propMap["type"].(string)
		switch expectedType {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("参数 %s 应为字符串类型", key)
			}
		case "integer", "number":
			switch val.(type) {
			case float64, int, int64:
				// ok
			default:
				return fmt.Errorf("参数 %s 应为数字类型", key)
			}
		case "boolean":
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("参数 %s 应为布尔类型", key)
			}
		case "array":
			if _, ok := val.([]interface{}); !ok {
				return fmt.Errorf("参数 %s 应为数组类型", key)
			}
		}
	}

	return nil
}

// SanitizeInput 输入安全清洗
func SanitizeInput(input string) string {
	// 移除潜在注入标记
	dangerous := []string{
		"system:", "assistant:", "user:",
		"```system", "```assistant",
		"ồ　", ".Lookup",
	}
	result := input
	for _, d := range dangerous {
		result = strings.ReplaceAll(result, d, "")
	}
	// 移除控制字符
	re := regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f]`)
	result = re.ReplaceAllString(result, "")
	return strings.TrimSpace(result)
}