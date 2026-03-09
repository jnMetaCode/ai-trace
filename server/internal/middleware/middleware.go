package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig 默认 CORS 配置
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key", "X-Tenant-ID", "X-User-ID", "X-Request-ID", "X-Upstream-API-Key"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS 返回 CORS 中间件
func CORS(config ...CORSConfig) gin.HandlerFunc {
	cfg := DefaultCORSConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否允许的来源
		allowed := false
		for _, o := range cfg.AllowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", joinStrings(cfg.AllowMethods))
		c.Header("Access-Control-Allow-Headers", joinStrings(cfg.AllowHeaders))
		c.Header("Access-Control-Expose-Headers", joinStrings(cfg.ExposeHeaders))

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", string(rune(cfg.MaxAge)))
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimiterConfig 速率限制配置
type RateLimiterConfig struct {
	RequestsPerMinute int           // 每分钟请求数
	BurstSize         int           // 突发大小
	CleanupInterval   time.Duration // 清理间隔
}

// DefaultRateLimiterConfig 默认速率限制配置
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		CleanupInterval:   5 * time.Minute,
	}
}

// clientInfo 客户端信息
type clientInfo struct {
	tokens     float64
	lastUpdate time.Time
}

// RateLimiter 速率限制器
type RateLimiter struct {
	clients map[string]*clientInfo
	mu      sync.RWMutex
	config  RateLimiterConfig
	rate    float64 // tokens per second
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(config ...RateLimiterConfig) *RateLimiter {
	cfg := DefaultRateLimiterConfig()
	if len(config) > 0 {
		cfg = config[0]
		// Use default cleanup interval if not set
		if cfg.CleanupInterval <= 0 {
			cfg.CleanupInterval = DefaultRateLimiterConfig().CleanupInterval
		}
	}

	rl := &RateLimiter{
		clients: make(map[string]*clientInfo),
		config:  cfg,
		rate:    float64(cfg.RequestsPerMinute) / 60.0,
	}

	// 启动清理协程
	go rl.cleanup()

	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[key]

	if !exists {
		rl.clients[key] = &clientInfo{
			tokens:     float64(rl.config.BurstSize) - 1,
			lastUpdate: now,
		}
		return true
	}

	// 计算自上次更新以来添加的 tokens
	elapsed := now.Sub(client.lastUpdate).Seconds()
	client.tokens += elapsed * rl.rate

	// 限制最大 tokens
	if client.tokens > float64(rl.config.BurstSize) {
		client.tokens = float64(rl.config.BurstSize)
	}

	client.lastUpdate = now

	// 检查是否有足够的 tokens
	if client.tokens >= 1 {
		client.tokens--
		return true
	}

	return false
}

// cleanup 定期清理过期的客户端记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.config.CleanupInterval)
		for key, client := range rl.clients {
			if client.lastUpdate.Before(cutoff) {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit 返回速率限制中间件
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用客户端 IP 或 API Key 作为限制键
		key := c.ClientIP()
		if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
			key = "apikey:" + apiKey
		}

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestID 添加请求 ID 中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Timeout 请求超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置超时
		c.Request = c.Request.WithContext(c.Request.Context())

		// 使用 channel 监控超时
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-time.After(timeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "Request timeout",
			})
			c.Abort()
		}
	}
}

// SecurityHeaders 安全头中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// 辅助函数

func joinStrings(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result
}

func generateRequestID() string {
	// 简单实现：时间戳 + 随机数
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
