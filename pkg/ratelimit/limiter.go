package ratelimit

import (
	"fmt"
	"time"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 检查请求是否允许通过
	Allow(key string) bool

	// Reset 重置限流状态
	Reset(key string)

	// GetQuota 获取剩余配额
	GetQuota(key string) int

	// GetLimit 获取限流阈值
	GetLimit() int

	// GetPeriod 获取时间周期
	GetPeriod() time.Duration
}

// RateLimiterStore 限流状态存储接口
type RateLimiterStore interface {
	// Incr 增加计数器
	Incr(key string, ttl time.Duration) (int, error)

	// Get 获取当前计数
	Get(key string) (int, error)

	// Set 设置计数值
	Set(key string, value int, ttl time.Duration) error

	// Del 删除计数器
	Del(key string) error
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Limit     int           `json:"limit" yaml:"limit"`         // 允许的请求数
	Period    time.Duration `json:"period" yaml:"period"`       // 时间周期
	Algorithm string        `json:"algorithm" yaml:"algorithm"` // 限流算法 (fixed, sliding, token_bucket, leaky_bucket)
	Key       string        `json:"key" yaml:"key"`             // 键生成表达式
}

// FixedWindowLimiter 固定窗口限流器
type FixedWindowLimiter struct {
	store     RateLimiterStore
	limit     int
	period    time.Duration
	keyPrefix string
}

// NewFixedWindowLimiter 创建固定窗口限流器
func NewFixedWindowLimiter(store RateLimiterStore, limit int, period time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		store:     store,
		limit:     limit,
		period:    period,
		keyPrefix: "fixed_window",
	}
}

// Allow 实现RateLimiter接口
func (l *FixedWindowLimiter) Allow(key string) bool {
	// 计算当前时间窗口
	window := time.Now().Unix() / int64(l.period.Seconds())
	actualKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, window)

	count, err := l.store.Incr(actualKey, l.period)
	if err != nil {
		// 错误时默认放行，避免影响服务可用性
		return true
	}

	return count <= l.limit
}

// Reset 重置限流状态
func (l *FixedWindowLimiter) Reset(key string) {
	window := time.Now().Unix() / int64(l.period.Seconds())
	actualKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, window)
	l.store.Del(actualKey)
}

// GetQuota 获取剩余配额
func (l *FixedWindowLimiter) GetQuota(key string) int {
	window := time.Now().Unix() / int64(l.period.Seconds())
	actualKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, window)

	count, err := l.store.Get(actualKey)
	if err != nil {
		return l.limit
	}

	remaining := l.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetLimit 获取限流阈值
func (l *FixedWindowLimiter) GetLimit() int {
	return l.limit
}

// GetPeriod 获取时间周期
func (l *FixedWindowLimiter) GetPeriod() time.Duration {
	return l.period
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	store     RateLimiterStore
	limit     int
	period    time.Duration
	precision int // 窗口细分数量
	keyPrefix string
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(store RateLimiterStore, limit int, period time.Duration, precision int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		store:     store,
		limit:     limit,
		period:    period,
		precision: precision,
		keyPrefix: "sliding_window",
	}
}

// Allow 实现RateLimiter接口
func (l *SlidingWindowLimiter) Allow(key string) bool {
	now := time.Now()
	windowSize := l.period / time.Duration(l.precision)
	currentWindow := now.Unix() / int64(windowSize.Seconds())

	// 获取当前窗口计数
	currentKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, currentWindow)
	currentCount, err := l.store.Incr(currentKey, windowSize)
	if err != nil {
		return true
	}

	// 获取上一个窗口的计数
	prevKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, currentWindow-1)
	prevCount, err := l.store.Get(prevKey)
	if err != nil {
		prevCount = 0
	}

	// 计算滑动窗口总请求数
	windowStart := time.Unix(currentWindow*int64(windowSize.Seconds()), 0)
	weight := float64(l.period-now.Sub(windowStart)) / float64(l.period)
	totalCount := currentCount + int(float64(prevCount)*weight)

	return totalCount <= l.limit
}

// Reset 重置限流状态
func (l *SlidingWindowLimiter) Reset(key string) {
	now := time.Now()
	windowSize := l.period / time.Duration(l.precision)
	currentWindow := now.Unix() / int64(windowSize.Seconds())

	// 删除当前和上一个窗口
	currentKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, currentWindow)
	prevKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, currentWindow-1)

	l.store.Del(currentKey)
	l.store.Del(prevKey)
}

// GetQuota 获取剩余配额
func (l *SlidingWindowLimiter) GetQuota(key string) int {
	now := time.Now()
	windowSize := l.period / time.Duration(l.precision)
	currentWindow := now.Unix() / int64(windowSize.Seconds())

	currentKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, currentWindow)
	currentCount, err := l.store.Get(currentKey)
	if err != nil {
		currentCount = 0
	}

	prevKey := fmt.Sprintf("%s:%s:%d", l.keyPrefix, key, currentWindow-1)
	prevCount, err := l.store.Get(prevKey)
	if err != nil {
		prevCount = 0
	}

	windowStart := time.Unix(currentWindow*int64(windowSize.Seconds()), 0)
	weight := float64(l.period-now.Sub(windowStart)) / float64(l.period)
	totalCount := currentCount + int(float64(prevCount)*weight)

	remaining := l.limit - totalCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetLimit 获取限流阈值
func (l *SlidingWindowLimiter) GetLimit() int {
	return l.limit
}

// GetPeriod 获取时间周期
func (l *SlidingWindowLimiter) GetPeriod() time.Duration {
	return l.period
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	store     RateLimiterStore
	limit     int     // 桶容量
	rate      float64 // 令牌填充速率（个/秒）
	keyPrefix string
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(store RateLimiterStore, limit int, rate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		store:     store,
		limit:     limit,
		rate:      rate,
		keyPrefix: "token_bucket",
	}
}

// Allow 实现RateLimiter接口
func (l *TokenBucketLimiter) Allow(key string) bool {
	bucketKey := fmt.Sprintf("%s:%s:tokens", l.keyPrefix, key)
	lastUpdateKey := fmt.Sprintf("%s:%s:last_update", l.keyPrefix, key)

	// 获取当前桶中令牌数和上次更新时间
	tokens, _ := l.store.Get(bucketKey)
	lastUpdateStr, _ := l.store.Get(lastUpdateKey)

	lastUpdate := time.Now()
	if lastUpdateStr != 0 {
		lastUpdate = time.Unix(int64(lastUpdateStr), 0)
	}

	// 计算新增令牌
	now := time.Now()
	elapsed := now.Sub(lastUpdate).Seconds()
	newTokens := int(elapsed * l.rate)

	if newTokens > 0 {
		// 添加新令牌，不超过桶容量
		if tokens+newTokens > l.limit {
			tokens = l.limit
		} else {
			tokens += newTokens
		}
		l.store.Set(bucketKey, tokens, 24*time.Hour)
		l.store.Set(lastUpdateKey, int(now.Unix()), 24*time.Hour)
	}

	// 尝试获取令牌
	if tokens > 0 {
		l.store.Set(bucketKey, tokens-1, 24*time.Hour)
		return true
	}

	return false
}

// Reset 重置限流状态
func (l *TokenBucketLimiter) Reset(key string) {
	bucketKey := fmt.Sprintf("%s:%s:tokens", l.keyPrefix, key)
	lastUpdateKey := fmt.Sprintf("%s:%s:last_update", l.keyPrefix, key)

	l.store.Del(bucketKey)
	l.store.Del(lastUpdateKey)
}

// GetQuota 获取剩余配额
func (l *TokenBucketLimiter) GetQuota(key string) int {
	bucketKey := fmt.Sprintf("%s:%s:tokens", l.keyPrefix, key)
	lastUpdateKey := fmt.Sprintf("%s:%s:last_update", l.keyPrefix, key)

	tokens, _ := l.store.Get(bucketKey)
	lastUpdateStr, _ := l.store.Get(lastUpdateKey)

	lastUpdate := time.Now()
	if lastUpdateStr != 0 {
		lastUpdate = time.Unix(int64(lastUpdateStr), 0)
	}

	// 计算新增令牌
	now := time.Now()
	elapsed := now.Sub(lastUpdate).Seconds()
	newTokens := int(elapsed * l.rate)

	if newTokens > 0 {
		if tokens+newTokens > l.limit {
			tokens = l.limit
		} else {
			tokens += newTokens
		}
	}

	return tokens
}

// GetLimit 获取限流阈值
func (l *TokenBucketLimiter) GetLimit() int {
	return l.limit
}

// GetPeriod 获取时间周期
func (l *TokenBucketLimiter) GetPeriod() time.Duration {
	return time.Duration(float64(l.limit) / l.rate * float64(time.Second))
}

// LeakyBucketLimiter 漏桶限流器
type LeakyBucketLimiter struct {
	store     RateLimiterStore
	capacity  int     // 桶容量
	rate      float64 // 漏水速率（请求/秒）
	keyPrefix string
}

// NewLeakyBucketLimiter 创建漏桶限流器
func NewLeakyBucketLimiter(store RateLimiterStore, capacity int, rate float64) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		store:     store,
		capacity:  capacity,
		rate:      rate,
		keyPrefix: "leaky_bucket",
	}
}

// Allow 实现RateLimiter接口
func (l *LeakyBucketLimiter) Allow(key string) bool {
	waterKey := fmt.Sprintf("%s:%s:water", l.keyPrefix, key)
	lastLeakKey := fmt.Sprintf("%s:%s:last_leak", l.keyPrefix, key)

	// 获取当前水量和上次漏水时间
	water, _ := l.store.Get(waterKey)
	lastLeakStr, _ := l.store.Get(lastLeakKey)

	lastLeak := time.Now()
	if lastLeakStr != 0 {
		lastLeak = time.Unix(int64(lastLeakStr), 0)
	}

	// 计算漏出的水量
	now := time.Now()
	elapsed := now.Sub(lastLeak).Seconds()
	leakedWater := int(elapsed * l.rate)

	if leakedWater > 0 {
		// 减少水量，不低于0
		if water-leakedWater < 0 {
			water = 0
		} else {
			water -= leakedWater
		}
		l.store.Set(waterKey, water, 24*time.Hour)
		l.store.Set(lastLeakKey, int(now.Unix()), 24*time.Hour)
	}

	// 尝试加水
	if water < l.capacity {
		l.store.Set(waterKey, water+1, 24*time.Hour)
		return true
	}

	return false
}

// Reset 重置限流状态
func (l *LeakyBucketLimiter) Reset(key string) {
	waterKey := fmt.Sprintf("%s:%s:water", l.keyPrefix, key)
	lastLeakKey := fmt.Sprintf("%s:%s:last_leak", l.keyPrefix, key)

	l.store.Del(waterKey)
	l.store.Del(lastLeakKey)
}

// GetQuota 获取剩余配额
func (l *LeakyBucketLimiter) GetQuota(key string) int {
	waterKey := fmt.Sprintf("%s:%s:water", l.keyPrefix, key)
	lastLeakKey := fmt.Sprintf("%s:%s:last_leak", l.keyPrefix, key)

	water, _ := l.store.Get(waterKey)
	lastLeakStr, _ := l.store.Get(lastLeakKey)

	lastLeak := time.Now()
	if lastLeakStr != 0 {
		lastLeak = time.Unix(int64(lastLeakStr), 0)
	}

	// 计算漏出的水量
	now := time.Now()
	elapsed := now.Sub(lastLeak).Seconds()
	leakedWater := int(elapsed * l.rate)

	if leakedWater > 0 {
		if water-leakedWater < 0 {
			water = 0
		} else {
			water -= leakedWater
		}
	}

	return l.capacity - water
}

// GetLimit 获取限流阈值
func (l *LeakyBucketLimiter) GetLimit() int {
	return l.capacity
}

// GetPeriod 获取时间周期
func (l *LeakyBucketLimiter) GetPeriod() time.Duration {
	return time.Duration(float64(l.capacity) / l.rate * float64(time.Second))
}
