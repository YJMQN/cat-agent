package security

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Guard 安全守卫接口
type Guard interface {
	ValidateToolCall(ctx context.Context, userID, toolName string, args json.RawMessage) error
	DetectInjection(content string) bool
	CheckRateLimit(ctx context.Context, userID string) error
}

// SecurityGuard 安全守卫实现
type SecurityGuard struct {
	rateLimiter     *RateLimiter
	blockedPatterns []*regexp.Regexp
}

// NewSecurityGuard 创建安全守卫
func NewSecurityGuard() *SecurityGuard {
	return &SecurityGuard{
		rateLimiter:     NewRateLimiter(100, 60),
		blockedPatterns: compileBlockedPatterns(),
	}
}

// ValidateToolCall 校验工具调用
func (g *SecurityGuard) ValidateToolCall(ctx context.Context, userID, toolName string, args json.RawMessage) error {
	if !isValidToolName(toolName) {
		return ErrInvalidToolName
	}

	var argsMap map[string]interface{}
	if err := json.Unmarshal(args, &argsMap); err == nil {
		if g.containsInjection(argsMap) {
			return ErrInjectionDetected
		}
	}

	return nil
}

// DetectInjection 检测注入攻击
func (g *SecurityGuard) DetectInjection(content string) bool {
	sqlPatterns := []string{
		`(?i)('|")?\s*(or|and)\s+[\d\w]+\s*=\s*[\d\w]+`,
		`(?i)union\s+(all\s+)?select`,
		`(?i)drop\s+table`,
	}

	for _, pattern := range sqlPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return true
		}
	}

	for _, bp := range g.blockedPatterns {
		if bp.MatchString(content) {
			return true
		}
	}

	return false
}

// containsInjection 递归检查注入
func (g *SecurityGuard) containsInjection(data interface{}) bool {
	switch v := data.(type) {
	case string:
		return g.DetectInjection(v)
	case map[string]interface{}:
		for _, val := range v {
			if g.containsInjection(val) {
				return true
			}
		}
	case []interface{}:
		for _, val := range v {
			if g.containsInjection(val) {
				return true
			}
		}
	}
	return false
}

// CheckRateLimit 检查速率限制
func (g *SecurityGuard) CheckRateLimit(ctx context.Context, userID string) error {
	if !g.rateLimiter.Allow(userID) {
		return ErrRateLimitExceeded
	}
	return nil
}

func isValidToolName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func isSensitiveTool(name string) bool {
	sensitiveTools := []string{"http_caller", "db_query", "file_read", "file_write"}
	for _, t := range sensitiveTools {
		if strings.EqualFold(name, t) {
			return true
		}
	}
	return false
}

func compileBlockedPatterns() []*regexp.Regexp {
	patterns := []string{
		`(?i)system\s+prompt`,
		`(?i)bypass\s+instruction`,
		`(?i)ignore\s+previous`,
	}

	var compiled []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// RateLimiter 令牌桶限流器
type RateLimiter struct {
	maxTokens  int
	refillRate int
	tokens     map[string]int
	lastRefill map[string]time.Time
	mu         sync.Mutex
}

func NewRateLimiter(maxTokens, refillRate int) *RateLimiter {
	return &RateLimiter{
		maxTokens:  maxTokens,
		refillRate: refillRate,
		tokens:     make(map[string]int),
		lastRefill: make(map[string]time.Time),
	}
}

func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if _, exists := rl.tokens[userID]; !exists {
		rl.tokens[userID] = rl.maxTokens
		rl.lastRefill[userID] = now
	}

	elapsed := now.Sub(rl.lastRefill[userID]).Minutes()
	newTokens := int(elapsed * float64(rl.refillRate))
	if newTokens > 0 {
		rl.tokens[userID] = minInt(rl.maxTokens, rl.tokens[userID]+newTokens)
		rl.lastRefill[userID] = now
	}

	if rl.tokens[userID] > 0 {
		rl.tokens[userID]--
		return true
	}

	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var (
	ErrInvalidToolName   = &SecurityError{Code: "INVALID_TOOL_NAME", Message: "非法的工具名称"}
	ErrInjectionDetected = &SecurityError{Code: "INJECTION_DETECTED", Message: "检测到注入攻击"}
	ErrRateLimitExceeded = &SecurityError{Code: "RATE_LIMIT_EXCEEDED", Message: "请求频率超限"}
)

type SecurityError struct {
	Code    string
	Message string
	Err     error
}

func (e *SecurityError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + " - " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *SecurityError) Unwrap() error {
	return e.Err
}
