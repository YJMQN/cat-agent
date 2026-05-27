package tool

import (
	"context"
	"testing"
	"time"
)

// ========== 第一阶段安全加固：单元测试 ==========

func TestToolConfirmationManager(t *testing.T) {
	manager := NewToolConfirmationManager(5 * time.Minute)

	// 测试创建确认
	conf := manager.CreateConfirmation(1, 100, "test_tool", "call_123", "{\"arg\": \"value\"}")
	if conf == nil {
		t.Fatal("创建确认失败")
	}
	if conf.ToolName != "test_tool" {
		t.Errorf("工具名称不匹配: got %s, want test_tool", conf.ToolName)
	}

	// 测试获取确认
	gotConf, err := manager.GetConfirmation(conf.ID)
	if err != nil {
		t.Fatalf("获取确认失败: %v", err)
	}
	if gotConf.ID != conf.ID {
		t.Errorf("确认ID不匹配")
	}

	// 测试确认执行
	err = manager.Confirm(conf.ID, 1)
	if err != nil {
		t.Fatalf("确认失败: %v", err)
	}

	// 测试重复确认
	err = manager.Confirm(conf.ID, 1)
	if err == nil {
		t.Error("重复确认应该失败")
	}

	// 测试权限检查
	err = manager.Confirm(conf.ID, 2)
	if err == nil {
		t.Error("非创建用户确认应该失败")
	}

	// 测试拒绝
	conf2 := manager.CreateConfirmation(1, 100, "test_tool2", "call_456", "{}")
	err = manager.Reject(conf2.ID, 1)
	if err != nil {
		t.Fatalf("拒绝失败: %v", err)
	}

	// 拒绝后应该无法获取
	_, err = manager.GetConfirmation(conf2.ID)
	if err == nil {
		t.Error("拒绝后应该无法获取确认")
	}
}

func TestToolConfirmationManagerTTL(t *testing.T) {
	manager := NewToolConfirmationManager(100 * time.Millisecond) // 短TTL用于测试

	conf := manager.CreateConfirmation(1, 100, "test_tool", "call_123", "{}")
	
	// 立即获取应该成功
	_, err := manager.GetConfirmation(conf.ID)
	if err != nil {
		t.Fatalf("立即获取失败: %v", err)
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 过期后获取应该失败
	_, err = manager.GetConfirmation(conf.ID)
	if err == nil {
		t.Error("过期后获取应该失败")
	}
}

func TestLocalCommandToolDangerousOperators(t *testing.T) {
	tool := &LocalCommandTool{}

	dangerousCommands := []string{
		"ls; rm -rf /",
		"cat /etc/passwd | grep root",
		"echo test && rm file",
		"echo test || echo fail",
		"ls > output.txt",
		"ls >> output.txt",
		"cat < input.txt",
		"echo $(whoami)",
		"echo `date`",
		"ls\nrm -rf",
	}

	for _, cmd := range dangerousCommands {
		if !containsDangerousOperators(cmd) {
			t.Errorf("应该检测到危险操作符: %s", cmd)
		}
	}

	safeCommands := []string{
		"ls -la",
		"cat file.txt",
		"echo hello world",
		"pwd",
		"date",
	}

	for _, cmd := range safeCommands {
		if containsDangerousOperators(cmd) {
			t.Errorf("不应检测为危险操作符: %s", cmd)
		}
	}
}

func TestLocalCommandToolExecute(t *testing.T) {
	tool := &LocalCommandTool{}
	ctx := context.Background()

	// 测试危险命令被拒绝
	result, err := tool.Execute(ctx, map[string]interface{}{
		"command": "ls; rm -rf /",
	})
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if !result.IsError {
		t.Error("危险命令应该返回错误")
	}
	if !contains(result.Content, "不允许使用命令链接符") {
		t.Errorf("错误消息不正确: %s", result.Content)
	}

	// 测试缺少命令参数
	result, err = tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if !result.IsError {
		t.Error("缺少参数应该返回错误")
	}
}

func TestCalculatorTool(t *testing.T) {
	tool := &CalculatorTool{}
	ctx := context.Background()

	tests := []struct {
		expression string
		expected   float64
	}{
		{"2+3", 5},
		{"10-4", 6},
		{"3*4", 12},
		{"10/2", 5},
		{"2+3*4", 14},
		{"(2+3)*4", 20},
		{"2^10", 1024},
		{"sqrt(16)", 4},
		{"abs(-5)", 5},
	}

	for _, tt := range tests {
		result, err := tool.Execute(ctx, map[string]interface{}{
			"expression": tt.expression,
		})
		if err != nil {
			t.Fatalf("计算 %s 失败: %v", tt.expression, err)
		}
		if result.IsError {
			t.Errorf("计算 %s 返回错误: %s", tt.expression, result.Content)
		}
	}

	// 测试除零错误
	result, err := tool.Execute(ctx, map[string]interface{}{
		"expression": "1/0",
	})
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if !result.IsError {
		t.Error("除零应该返回错误")
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"system: override instructions", "override instructions"},
		{"Normal text```system injected", "Normal text injected"},
		{"Text with ồ　unicode", "Text with unicode"},
		{"Text\x00with\x01control\x02chars", "Textwithcontrolchars"},
	}

	for _, tt := range tests {
		result := SanitizeInput(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeInput(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestTrimOutput(t *testing.T) {
	shortContent := "short content"
	result := trimOutput(shortContent, 1000)
	if result != shortContent {
		t.Error("短内容不应被截断")
	}

	longContent := ""
	for i := 0; i < 200; i++ {
		longContent += "abcdefghij"
	}
	result = trimOutput(longContent, 100)
	if len(result) > 120 { // 100 + 截断提示
		t.Errorf("长内容应该被截断: len=%d", len(result))
	}
	if !contains(result, "截断") {
		t.Error("截断内容应包含截断提示")
	}
}

func TestEstimateTokens(t *testing.T) {
	content := "This is a test content with about 40 characters"
	tokens := estimateTokens(content)
	// 约4字符=1token，所以40字符约10tokens
	if tokens < 5 || tokens > 20 {
		t.Errorf("Token估算异常: %d", tokens)
	}
}

func TestFileToolSensitivePaths(t *testing.T) {
	sensitivePaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/root/.ssh/id_rsa",
		"/home/user/.env",
		"secret.key",
	}

	for _, path := range sensitivePaths {
		if !isSensitivePath(path) {
			t.Errorf("应该检测为敏感路径: %s", path)
		}
	}

	normalPaths := []string{
		"/tmp/test.txt",
		"/home/user/documents/readme.md",
		"./config.json",
	}

	for _, path := range normalPaths {
		if isSensitivePath(path) {
			t.Errorf("不应检测为敏感路径: %s", path)
		}
	}
}

func TestValidateArgs(t *testing.T) {
	tool := &WeatherTool{}

	// 测试缺少必填参数
	err := ValidateArgs(tool, map[string]interface{}{})
	if err == nil {
		t.Error("缺少必填参数应该返回错误")
	}

	// 测试正确参数
	err = ValidateArgs(tool, map[string]interface{}{
		"city": "Beijing",
	})
	if err != nil {
		t.Errorf("正确参数不应返回错误: %v", err)
	}

	// 测试类型错误
	err = ValidateArgs(tool, map[string]interface{}{
		"city": 123, // 应该是字符串
	})
	if err == nil {
		t.Error("类型错误应该返回错误")
	}
}

// ========== 工具注册表测试 ==========

func TestToolRegistry(t *testing.T) {
	registry := NewRegistry()

	weatherTool := &WeatherTool{}
	registry.Register(weatherTool)

	// 测试获取工具
	gotTool, ok := registry.Get("weather")
	if !ok {
		t.Fatal("获取工具失败")
	}
	if gotTool.Name() != "weather" {
		t.Errorf("工具名称不匹配: got %s, want weather", gotTool.Name())
	}

	// 测试工具列表
	tools := registry.List()
	if len(tools) != 1 {
		t.Errorf("工具列表长度不正确: %d", len(tools))
	}

	// 测试工具定义
	defs := registry.ToolDefs([]string{"weather"})
	if len(defs) != 1 {
		t.Errorf("工具定义列表长度不正确: %d", len(defs))
	}
}

func TestLoadBuiltinTools(t *testing.T) {
	registry := LoadBuiltinTools()

	// 检查所有内置工具是否注册
	expectedTools := []string{"weather", "calculator", "web_search", "web_fetch", "local_command", "file"}
	for _, name := range expectedTools {
		_, ok := registry.Get(name)
		if !ok {
			t.Errorf("内置工具 %s 未注册", name)
		}
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}