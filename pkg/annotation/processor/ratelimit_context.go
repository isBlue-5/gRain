package processor

import (
	"fmt"
	"sync"
	"time"

	"github.com/isBlue-5/grain/pkg/ratelimit"
)

// RateLimitManager 限流器管理器
type RateLimitManager struct {
	limiters map[string]ratelimit.RateLimiter
	store    ratelimit.RateLimiterStore
	mutex    sync.RWMutex
}

// NewRateLimitManager 创建限流器管理器
func NewRateLimitManager() *RateLimitManager {
	return &RateLimitManager{
		limiters: make(map[string]ratelimit.RateLimiter),
		store:    ratelimit.NewMemoryStore(),
	}
}

// GetLimiter 获取或创建限流器
func (m *RateLimitManager) GetLimiter(name string, limit int, period string, algorithm string) ratelimit.RateLimiter {
	key := m.generateLimiterKey(name, limit, period, algorithm)

	m.mutex.RLock()
	if limiter, exists := m.limiters[key]; exists {
		m.mutex.RUnlock()
		return limiter
	}
	m.mutex.RUnlock()

	// 创建新的限流器
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 双重检查
	if limiter, exists := m.limiters[key]; exists {
		return limiter
	}

	// 解析周期
	periodDuration, err := time.ParseDuration(period)
	if err != nil {
		periodDuration = time.Minute // 默认1分钟
	}

	var limiter ratelimit.RateLimiter
	switch algorithm {
	case "sliding_window":
		limiter = ratelimit.NewSlidingWindowLimiter(m.store, limit, periodDuration, 10) // 10个子窗口
	case "token_bucket":
		limiter = ratelimit.NewTokenBucketLimiter(m.store, limit, float64(limit)/periodDuration.Seconds()) // 每秒补充速率
	case "leaky_bucket":
		limiter = ratelimit.NewLeakyBucketLimiter(m.store, limit, float64(limit)/periodDuration.Seconds()) // 每秒漏出速率
	case "fixed_window":
		limiter = ratelimit.NewFixedWindowLimiter(m.store, limit, periodDuration)
	default:
		// 默认使用固定窗口（最简单）
		limiter = ratelimit.NewFixedWindowLimiter(m.store, limit, periodDuration)
	}

	m.limiters[key] = limiter
	return limiter
}

// generateLimiterKey 生成限流器键
func (m *RateLimitManager) generateLimiterKey(name string, limit int, period string, algorithm string) string {
	return fmt.Sprintf("%s_%d_%s_%s", name, limit, period, algorithm)
}

// 全局限流器管理器
var globalRateLimitManager *RateLimitManager
var rateLimitOnce sync.Once

// GetGlobalRateLimitManager 获取全局限流器管理器
func GetGlobalRateLimitManager() *RateLimitManager {
	rateLimitOnce.Do(func() {
		globalRateLimitManager = NewRateLimitManager()
	})
	return globalRateLimitManager
}

// 导出的辅助函数，用于注解生成的代码

// GetRateLimiter 获取限流器（用于注解生成的代码）
func GetRateLimiter(name string, limit int, period string, algorithm string) ratelimit.RateLimiter {
	manager := GetGlobalRateLimitManager()
	return manager.GetLimiter(name, limit, period, algorithm)
}

// CheckRateLimit 检查限流（用于注解生成的代码）
func CheckRateLimit(limiterKey string, limit int, period string, algorithm string, userKey string) bool {
	limiter := GetRateLimiter(limiterKey, limit, period, algorithm)
	return limiter.Allow(userKey)
}

// GetRateLimitQuota 获取剩余配额（用于注解生成的代码）
func GetRateLimitQuota(limiterKey string, limit int, period string, algorithm string, userKey string) int {
	limiter := GetRateLimiter(limiterKey, limit, period, algorithm)
	return limiter.GetQuota(userKey)
}

// GetRateLimitInfo 获取限流信息（用于注解生成的代码）
func GetRateLimitInfo(limiterKey string, limit int, period string, algorithm string) (int, int) {
	limiter := GetRateLimiter(limiterKey, limit, period, algorithm)
	return limiter.GetLimit(), limiter.GetQuota("")
}
