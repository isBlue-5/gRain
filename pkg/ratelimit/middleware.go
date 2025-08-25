package ratelimit

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitingConfig 限流配置
type RateLimitingConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled" env:"RATE_LIMIT_ENABLED" default:"true"`
	StoreType string `json:"storeType" yaml:"storeType" env:"RATE_LIMIT_STORE" default:"memory"` // memory, redis

	// 全局限流设置
	Global struct {
		Enabled   bool          `json:"enabled" yaml:"enabled" env:"RATE_LIMIT_GLOBAL_ENABLED" default:"true"`
		Limit     int           `json:"limit" yaml:"limit" env:"RATE_LIMIT_GLOBAL_LIMIT" default:"1000"`
		Period    time.Duration `json:"period" yaml:"period" env:"RATE_LIMIT_GLOBAL_PERIOD" default:"1m"`
		Algorithm string        `json:"algorithm" yaml:"algorithm" env:"RATE_LIMIT_GLOBAL_ALGORITHM" default:"token_bucket"`
	} `json:"global" yaml:"global"`

	// IP限流设置
	IP struct {
		Enabled   bool          `json:"enabled" yaml:"enabled" env:"RATE_LIMIT_IP_ENABLED" default:"true"`
		Limit     int           `json:"limit" yaml:"limit" env:"RATE_LIMIT_IP_LIMIT" default:"100"`
		Period    time.Duration `json:"period" yaml:"period" env:"RATE_LIMIT_IP_PERIOD" default:"1m"`
		Algorithm string        `json:"algorithm" yaml:"algorithm" env:"RATE_LIMIT_IP_ALGORITHM" default:"sliding_window"`
		Whitelist []string      `json:"whitelist" yaml:"whitelist" env:"RATE_LIMIT_IP_WHITELIST"`
	} `json:"ip" yaml:"ip"`

	// 路由限流规则
	Routes []struct {
		Path      string        `json:"path" yaml:"path"`
		Method    string        `json:"method" yaml:"method"`
		Limit     int           `json:"limit" yaml:"limit"`
		Period    time.Duration `json:"period" yaml:"period"`
		Algorithm string        `json:"algorithm" yaml:"algorithm"`
		Key       string        `json:"key" yaml:"key" default:"ip"`
	} `json:"routes" yaml:"routes"`
}

// RateLimitRule 限流规则
type RateLimitRule struct {
	Path      string        `json:"path" yaml:"path"`
	Method    string        `json:"method" yaml:"method"`
	Limit     int           `json:"limit" yaml:"limit"`
	Period    time.Duration `json:"period" yaml:"period"`
	Algorithm string        `json:"algorithm" yaml:"algorithm"`
	Key       string        `json:"key" yaml:"key"`
}

// RateLimitMiddleware 全局限流中间件
func RateLimitMiddleware(limiter RateLimiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.GetLimit()))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(limiter.GetQuota(key)))
			c.Header("Retry-After", "60") // 建议客户端等待时间
			return
		}

		// 添加限流相关头信息
		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.GetLimit()))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(limiter.GetQuota(key)))

		c.Next()
	}
}

// RouteRateLimitMiddleware 路由级限流中间件
func RouteRateLimitMiddleware(limiterFactory func(RateLimitConfig) RateLimiter, rules []RateLimitRule) gin.HandlerFunc {
	limiters := make(map[string]RateLimiter)

	// 为每个规则创建限流器
	for _, rule := range rules {
		config := RateLimitConfig{
			Limit:     rule.Limit,
			Period:    rule.Period,
			Algorithm: rule.Algorithm,
		}
		limiters[rule.Path] = limiterFactory(config)
	}

	return func(c *gin.Context) {
		path := c.FullPath()
		method := c.Request.Method

		// 寻找匹配的规则
		var matchedLimiter RateLimiter
		var matchedRule RateLimitRule

		for _, rule := range rules {
			if (rule.Path == path || (strings.HasSuffix(rule.Path, "*") && strings.HasPrefix(path, rule.Path[:len(rule.Path)-1]))) &&
				(rule.Method == "" || rule.Method == method) {
				matchedLimiter = limiters[rule.Path]
				matchedRule = rule
				break
			}
		}

		// 未找到匹配规则，直接放行
		if matchedLimiter == nil {
			c.Next()
			return
		}

		// 计算限流键
		key := generateKey(c, matchedRule.Key)

		if !matchedLimiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded for this route",
			})
			c.Header("X-RateLimit-Limit", strconv.Itoa(matchedLimiter.GetLimit()))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(matchedLimiter.GetQuota(key)))
			return
		}

		c.Next()
	}
}

// generateKey 根据规则生成键
func generateKey(c *gin.Context, keyTemplate string) string {
	switch {
	case keyTemplate == "ip":
		return c.ClientIP()
	case keyTemplate == "path":
		return c.FullPath()
	case strings.HasPrefix(keyTemplate, "user:"):
		if user, exists := c.Get("user"); exists {
			if id, ok := user.(string); ok {
				return id
			}
		}
		return "anonymous"
	case strings.HasPrefix(keyTemplate, "header:"):
		headerName := keyTemplate[7:]
		return c.GetHeader(headerName)
	default:
		return keyTemplate
	}
}

// IPKeyFunc 基于IP的键生成函数
func IPKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// GlobalKeyFunc 全局键生成函数
func GlobalKeyFunc(c *gin.Context) string {
	return "global"
}

// UserKeyFunc 基于用户的键生成函数
func UserKeyFunc(c *gin.Context) string {
	if user, exists := c.Get("user"); exists {
		if id, ok := user.(string); ok {
			return "user:" + id
		}
	}
	return "anonymous"
}

// HeaderKeyFunc 基于请求头的键生成函数
func HeaderKeyFunc(headerName string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		return c.GetHeader(headerName)
	}
}

// PathKeyFunc 基于路径的键生成函数
func PathKeyFunc(c *gin.Context) string {
	return c.FullPath()
}

// CombinedKeyFunc 组合键生成函数
func CombinedKeyFunc(keyFuncs ...func(*gin.Context) string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		var keys []string
		for _, keyFunc := range keyFuncs {
			keys = append(keys, keyFunc(c))
		}
		return strings.Join(keys, ":")
	}
}

// RateLimitOption 限流中间件选项
type RateLimitOption func(*RateLimitConfig)

// WithLimit 设置限流阈值
func WithLimit(limit int) RateLimitOption {
	return func(config *RateLimitConfig) {
		config.Limit = limit
	}
}

// WithPeriod 设置时间周期
func WithPeriod(period time.Duration) RateLimitOption {
	return func(config *RateLimitConfig) {
		config.Period = period
	}
}

// WithAlgorithm 设置限流算法
func WithAlgorithm(algorithm string) RateLimitOption {
	return func(config *RateLimitConfig) {
		config.Algorithm = algorithm
	}
}

// WithKey 设置键生成表达式
func WithKey(key string) RateLimitOption {
	return func(config *RateLimitConfig) {
		config.Key = key
	}
}

// NewRateLimitConfig 创建限流配置
func NewRateLimitConfig(options ...RateLimitOption) RateLimitConfig {
	config := RateLimitConfig{
		Limit:     60,
		Period:    time.Minute,
		Algorithm: "fixed_window",
		Key:       "ip",
	}

	for _, option := range options {
		option(&config)
	}

	return config
}
