package ratelimit

import (
	"fmt"
	"time"
)

// NewRateLimiter 创建限流器
func NewRateLimiter(config RateLimitConfig, store RateLimiterStore) RateLimiter {
	switch config.Algorithm {
	case "sliding_window":
		return NewSlidingWindowLimiter(store, config.Limit, config.Period, 10)
	case "token_bucket":
		rate := float64(config.Limit) / config.Period.Seconds()
		return NewTokenBucketLimiter(store, config.Limit, rate)
	case "leaky_bucket":
		rate := float64(config.Limit) / config.Period.Seconds()
		return NewLeakyBucketLimiter(store, config.Limit, rate)
	case "fixed_window":
		fallthrough
	default:
		return NewFixedWindowLimiter(store, config.Limit, config.Period)
	}
}

// NewRateLimiterFactory 创建限流器工厂
func NewRateLimiterFactory(store RateLimiterStore) func(RateLimitConfig) RateLimiter {
	return func(config RateLimitConfig) RateLimiter {
		return NewRateLimiter(config, store)
	}
}

// NewDefaultRateLimiter 创建默认限流器
func NewDefaultRateLimiter() RateLimiter {
	store := NewMemoryStore()
	config := RateLimitConfig{
		Limit:     60,
		Period:    time.Minute,
		Algorithm: "fixed_window",
		Key:       "ip",
	}
	return NewRateLimiter(config, store)
}

// NewRateLimiterFromConfig 从配置创建限流器
func NewRateLimiterFromConfig(config RateLimitingConfig) (RateLimiter, error) {
	var store RateLimiterStore

	// 创建存储
	switch config.StoreType {
	case "redis":
		// 这里需要实际的Redis客户端
		// 暂时使用Mock客户端
		store = NewRedisStore(NewMockRedisClient())
	case "memory":
		fallthrough
	default:
		store = NewMemoryStore()
	}

	// 创建全局限流器
	globalConfig := RateLimitConfig{
		Limit:     config.Global.Limit,
		Period:    config.Global.Period,
		Algorithm: config.Global.Algorithm,
		Key:       "global",
	}

	return NewRateLimiter(globalConfig, store), nil
}

// NewIPRateLimiter 创建IP限流器
func NewIPRateLimiter(config RateLimitingConfig) (RateLimiter, error) {
	var store RateLimiterStore

	// 创建存储
	switch config.StoreType {
	case "redis":
		store = NewRedisStore(NewMockRedisClient())
	case "memory":
		fallthrough
	default:
		store = NewMemoryStore()
	}

	// 创建IP限流器
	ipConfig := RateLimitConfig{
		Limit:     config.IP.Limit,
		Period:    config.IP.Period,
		Algorithm: config.IP.Algorithm,
		Key:       "ip",
	}

	return NewRateLimiter(ipConfig, store), nil
}

// NewRouteRateLimiters 创建路由限流器
func NewRouteRateLimiters(config RateLimitingConfig) (map[string]RateLimiter, error) {
	var store RateLimiterStore

	// 创建存储
	switch config.StoreType {
	case "redis":
		store = NewRedisStore(NewMockRedisClient())
	case "memory":
		fallthrough
	default:
		store = NewMemoryStore()
	}

	limiters := make(map[string]RateLimiter)

	// 为每个路由规则创建限流器
	for _, route := range config.Routes {
		routeConfig := RateLimitConfig{
			Limit:     route.Limit,
			Period:    route.Period,
			Algorithm: route.Algorithm,
			Key:       route.Key,
		}

		limiter := NewRateLimiter(routeConfig, store)
		limiters[route.Path] = limiter
	}

	return limiters, nil
}

// ValidateRateLimitConfig 验证限流配置
func ValidateRateLimitConfig(config RateLimitConfig) error {
	if config.Limit <= 0 {
		return fmt.Errorf("limit must be positive, got %d", config.Limit)
	}

	if config.Period <= 0 {
		return fmt.Errorf("period must be positive, got %v", config.Period)
	}

	validAlgorithms := []string{"fixed_window", "sliding_window", "token_bucket", "leaky_bucket"}
	valid := false
	for _, algo := range validAlgorithms {
		if config.Algorithm == algo {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid algorithm: %s, valid algorithms: %v", config.Algorithm, validAlgorithms)
	}

	return nil
}

// ValidateRateLimitingConfig 验证限流配置
func ValidateRateLimitingConfig(config RateLimitingConfig) error {
	if config.Global.Enabled {
		globalConfig := RateLimitConfig{
			Limit:     config.Global.Limit,
			Period:    config.Global.Period,
			Algorithm: config.Global.Algorithm,
		}
		if err := ValidateRateLimitConfig(globalConfig); err != nil {
			return fmt.Errorf("global rate limit config error: %w", err)
		}
	}

	if config.IP.Enabled {
		ipConfig := RateLimitConfig{
			Limit:     config.IP.Limit,
			Period:    config.IP.Period,
			Algorithm: config.IP.Algorithm,
		}
		if err := ValidateRateLimitConfig(ipConfig); err != nil {
			return fmt.Errorf("IP rate limit config error: %w", err)
		}
	}

	for i, route := range config.Routes {
		routeConfig := RateLimitConfig{
			Limit:     route.Limit,
			Period:    route.Period,
			Algorithm: route.Algorithm,
			Key:       route.Key,
		}
		if err := ValidateRateLimitConfig(routeConfig); err != nil {
			return fmt.Errorf("route %d rate limit config error: %w", i, err)
		}
	}

	return nil
}

// DefaultRateLimitingConfig 创建默认限流配置
func DefaultRateLimitingConfig() RateLimitingConfig {
	config := RateLimitingConfig{
		Enabled:   true,
		StoreType: "memory",
	}

	// 全局限流默认配置
	config.Global.Enabled = true
	config.Global.Limit = 1000
	config.Global.Period = time.Minute
	config.Global.Algorithm = "token_bucket"

	// IP限流默认配置
	config.IP.Enabled = true
	config.IP.Limit = 100
	config.IP.Period = time.Minute
	config.IP.Algorithm = "sliding_window"
	config.IP.Whitelist = []string{}

	// 路由限流默认规则
	config.Routes = []struct {
		Path      string        `json:"path" yaml:"path"`
		Method    string        `json:"method" yaml:"method"`
		Limit     int           `json:"limit" yaml:"limit"`
		Period    time.Duration `json:"period" yaml:"period"`
		Algorithm string        `json:"algorithm" yaml:"algorithm"`
		Key       string        `json:"key" yaml:"key" default:"ip"`
	}{
		{
			Path:      "/api/search",
			Method:    "GET",
			Limit:     10,
			Period:    time.Minute,
			Algorithm: "fixed_window",
			Key:       "ip",
		},
		{
			Path:      "/api/users/*",
			Method:    "",
			Limit:     50,
			Period:    time.Minute,
			Algorithm: "sliding_window",
			Key:       "ip",
		},
	}

	return config
}
