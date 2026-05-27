package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// JWTAuth JWT认证中间件 (从handler包引用以避免循环依赖)
// 实际实现在handler包中通过闭包注入
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证令牌"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			c.Abort()
			return
		}

		tokenStr := parts[1]

		// 使用标准JWT解析
		parser := &jwtParser{secret: secret}
		claims, err := parser.Parse(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌无效或已过期"})
			c.Abort()
			return
		}

		// 将用户信息注入上下文
		if userID, ok := claims["user_id"].(float64); ok {
			c.Set("user_id", uint(userID))
		}
		if username, ok := claims["username"].(string); ok {
			c.Set("username", username)
		}
		if role, ok := claims["role"].(string); ok {
			c.Set("role", role)
		}

		c.Next()
	}
}

// AdminOnly 管理员权限中间件
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(string) != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ========== 第一阶段安全加固: 速率限制中间件 ==========

// RateLimiter 速率限制器结构
type RateLimiter struct {
	limiters sync.Map // key: IP地址, value: *rate.Limiter
	mu       sync.Mutex
	rps      int // 每秒请求数
	burst    int // 突发上限
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(rps, burst int) *RateLimiter {
	return &RateLimiter{
		rps:   rps,
		burst: burst,
	}
}

// GetLimiter 获取或创建指定IP的限流器
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	if limiter, ok := rl.limiters.Load(ip); ok {
		return limiter.(*rate.Limiter)
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 再次检查避免并发创建
	if limiter, ok := rl.limiters.Load(ip); ok {
		return limiter.(*rate.Limiter)
	}

	limiter := rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters.Store(ip, limiter)
	return limiter
}

// CleanupLimiters 定期清理过期的限流器
func (rl *RateLimiter) CleanupLimiters(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			rl.limiters.Clear()
		}
	}()
}

// globalRateLimiter 全局速率限制器实例
var globalRateLimiter *RateLimiter

// RateLimit 速率限制中间件
func RateLimit(rps, burst int) gin.HandlerFunc {
	globalRateLimiter = NewRateLimiter(rps, burst)
	globalRateLimiter.CleanupLimiters(10 * time.Minute)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := globalRateLimiter.GetLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
				"retry_after": "1s",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}