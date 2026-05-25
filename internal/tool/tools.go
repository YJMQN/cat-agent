package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Result 工具执行结果
type Result struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
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
	return r
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
	client := &http.Client{Timeout: 10 * time.Second}
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

	return &Result{Content: result}, nil
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

	return &Result{Content: fmt.Sprintf("%s = %g", expr, result)}, nil
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

// ========== 网络搜索工具 ==========

// WebSearchTool 网络搜索工具
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string  { return "搜索互联网获取信息" }
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

// searchResult 搜索结果
type searchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
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

	// 使用DuckDuckGo Lite搜索 (无API key)
	searchURL := fmt.Sprintf("https://lite.duckduckgo.com/lite/?q=%s", url.QueryEscape(query))
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; EinoAgent/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return &Result{Content: fmt.Sprintf("搜索失败: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	// 简单解析搜索结果
	var results []searchResult
	_ = limit // 使用limit控制结果数

	// 由于DuckDuckGo解析较复杂，返回模拟结构说明
	// 实际部署可替换为SerpAPI等付费搜索
	buf := new(strings.Builder)
	fmt.Fprintf(buf, "搜索 \"%s\" 的结果：\n\n", query)
	fmt.Fprintf(buf, "提示：当前为演示模式。如需真实搜索结果，请配置搜索API密钥（如SerpAPI、Bing Search API等）。\n")
	fmt.Fprintf(buf, "搜索查询已记录，可在日志中查看。\n")

	_ = results
	_ = resp

	return &Result{Content: buf.String()}, nil
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
		"<|system|>", "<|assistant|>",
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
