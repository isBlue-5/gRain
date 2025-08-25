package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryStore 内存存储实现
type MemoryStore struct {
	counters map[string]int
	expiry   map[string]time.Time
	mu       sync.RWMutex
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		counters: make(map[string]int),
		expiry:   make(map[string]time.Time),
	}
}

// Incr 实现RateLimiterStore接口
func (s *MemoryStore) Incr(key string, ttl time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否过期
	if expiry, ok := s.expiry[key]; ok && expiry.Before(time.Now()) {
		delete(s.counters, key)
		delete(s.expiry, key)
	}

	// 增加计数
	s.counters[key]++
	s.expiry[key] = time.Now().Add(ttl)

	return s.counters[key], nil
}

// Get 实现RateLimiterStore接口
func (s *MemoryStore) Get(key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查是否过期
	if expiry, ok := s.expiry[key]; ok && expiry.Before(time.Now()) {
		return 0, nil
	}

	return s.counters[key], nil
}

// Set 实现RateLimiterStore接口
func (s *MemoryStore) Set(key string, value int, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counters[key] = value
	s.expiry[key] = time.Now().Add(ttl)

	return nil
}

// Del 实现RateLimiterStore接口
func (s *MemoryStore) Del(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.counters, key)
	delete(s.expiry, key)

	return nil
}

// Cleanup 清理过期数据
func (s *MemoryStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, expiry := range s.expiry {
		if expiry.Before(now) {
			delete(s.counters, key)
			delete(s.expiry, key)
		}
	}
}

// StartCleanup 启动定期清理
func (s *MemoryStore) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.Cleanup()
		}
	}()
}

// RedisStore Redis存储实现
type RedisStore struct {
	client RedisClient
}

// RedisClient Redis客户端接口
type RedisClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// NewRedisStore 创建Redis存储
func NewRedisStore(client RedisClient) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// Incr 实现RateLimiterStore接口
func (s *RedisStore) Incr(key string, ttl time.Duration) (int, error) {
	ctx := context.Background()

	// 增加计数
	count, err := s.client.Incr(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("redis incr failed: %w", err)
	}

	// 设置过期时间
	err = s.client.Expire(ctx, key, ttl)
	if err != nil {
		return 0, fmt.Errorf("redis expire failed: %w", err)
	}

	return int(count), nil
}

// Get 实现RateLimiterStore接口
func (s *RedisStore) Get(key string) (int, error) {
	ctx := context.Background()

	value, err := s.client.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	var count int
	_, err = fmt.Sscanf(value, "%d", &count)
	if err != nil {
		return 0, fmt.Errorf("parse redis value failed: %w", err)
	}

	return count, nil
}

// Set 实现RateLimiterStore接口
func (s *RedisStore) Set(key string, value int, ttl time.Duration) error {
	ctx := context.Background()

	err := s.client.Set(ctx, key, value, ttl)
	if err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// Del 实现RateLimiterStore接口
func (s *RedisStore) Del(key string) error {
	ctx := context.Background()

	_, err := s.client.Del(ctx, key)
	if err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}

	return nil
}

// MockRedisClient 用于测试的Mock Redis客户端
type MockRedisClient struct {
	data map[string]string
	mu   sync.RWMutex
}

// NewMockRedisClient 创建Mock Redis客户端
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
	}
}

// Incr 实现RedisClient接口
func (m *MockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	if value, ok := m.data[key]; ok {
		fmt.Sscanf(value, "%d", &count)
	}

	count++
	m.data[key] = fmt.Sprintf("%d", count)

	return count, nil
}

// Get 实现RedisClient接口
func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, ok := m.data[key]; ok {
		return value, nil
	}

	return "", fmt.Errorf("key not found: %s", key)
}

// Set 实现RedisClient接口
func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = fmt.Sprintf("%v", value)
	return nil
}

// Del 实现RedisClient接口
func (m *MockRedisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			deleted++
		}
	}

	return deleted, nil
}

// Expire 实现RedisClient接口
func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	// Mock实现中忽略过期时间
	return nil
}
