package config

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ConfigCache 配置缓存
type ConfigCache struct {
	// 缓存数据
	data map[string]*CacheEntry

	// 缓存锁
	mu sync.RWMutex

	// 缓存统计
	stats *CacheStats

	// 缓存选项
	options *CacheOptions
}

// CacheEntry 缓存条目
type CacheEntry struct {
	// 缓存的值
	Value interface{}

	// 缓存时间
	CreatedAt time.Time

	// 最后访问时间
	LastAccessedAt time.Time

	// 访问次数
	AccessCount int64

	// 过期时间
	ExpiresAt *time.Time

	// 数据哈希
	Hash string
}

// CacheStats 缓存统计
type CacheStats struct {
	// 缓存命中次数
	Hits int64

	// 缓存未命中次数
	Misses int64

	// 缓存设置次数
	Sets int64

	// 缓存删除次数
	Deletes int64

	// 缓存过期次数
	Expirations int64
}

// CacheOptions 缓存选项
type CacheOptions struct {
	// 最大缓存条目数
	MaxEntries int

	// 默认过期时间
	DefaultTTL time.Duration

	// 是否启用LRU淘汰
	EnableLRU bool

	// 清理间隔
	CleanupInterval time.Duration
}

// DefaultCacheOptions 默认缓存选项
func DefaultCacheOptions() *CacheOptions {
	return &CacheOptions{
		MaxEntries:      1000,
		DefaultTTL:      5 * time.Minute,
		EnableLRU:       true,
		CleanupInterval: 1 * time.Minute,
	}
}

// NewConfigCache 创建新的配置缓存
func NewConfigCache() *ConfigCache {
	cache := &ConfigCache{
		data:    make(map[string]*CacheEntry),
		stats:   &CacheStats{},
		options: DefaultCacheOptions(),
	}

	// 启动清理协程
	go cache.startCleanup()

	return cache
}

// NewConfigCacheWithOptions 使用自定义选项创建配置缓存
func NewConfigCacheWithOptions(options *CacheOptions) *ConfigCache {
	if options == nil {
		options = DefaultCacheOptions()
	}

	cache := &ConfigCache{
		data:    make(map[string]*CacheEntry),
		stats:   &CacheStats{},
		options: options,
	}

	// 启动清理协程
	go cache.startCleanup()

	return cache
}

// Set 设置缓存
func (cc *ConfigCache) Set(key string, value interface{}) {
	cc.SetWithTTL(key, value, cc.options.DefaultTTL)
}

// SetWithTTL 设置缓存并指定过期时间
func (cc *ConfigCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// 检查缓存大小限制
	if len(cc.data) >= cc.options.MaxEntries {
		cc.evictLRU()
	}

	// 计算过期时间
	var expiresAt *time.Time
	if ttl > 0 {
		exp := time.Now().Add(ttl)
		expiresAt = &exp
	}

	// 计算数据哈希
	hash := cc.calculateHash(value)

	// 创建缓存条目
	entry := &CacheEntry{
		Value:          value,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		AccessCount:    0,
		ExpiresAt:      expiresAt,
		Hash:           hash,
	}

	cc.data[key] = entry
	cc.stats.Sets++
}

// Get 获取缓存
func (cc *ConfigCache) Get(key string) (interface{}, bool) {
	cc.mu.RLock()
	entry, exists := cc.data[key]
	cc.mu.RUnlock()

	if !exists {
		cc.stats.Misses++
		return nil, false
	}

	// 检查是否过期
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		cc.mu.Lock()
		delete(cc.data, key)
		cc.stats.Expirations++
		cc.mu.Unlock()
		cc.stats.Misses++
		return nil, false
	}

	// 更新访问统计
	cc.mu.Lock()
	entry.LastAccessedAt = time.Now()
	entry.AccessCount++
	cc.mu.Unlock()

	cc.stats.Hits++
	return entry.Value, true
}

// GetWithType 获取缓存并指定类型
func (cc *ConfigCache) GetWithType(key string, target interface{}) bool {
	value, exists := cc.Get(key)
	if !exists {
		return false
	}

	// 尝试将值转换为目标类型
	if err := cc.convertValue(value, target); err != nil {
		return false
	}

	return true
}

// Delete 删除缓存
func (cc *ConfigCache) Delete(key string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, exists := cc.data[key]; exists {
		delete(cc.data, key)
		cc.stats.Deletes++
	}
}

// Clear 清空缓存
func (cc *ConfigCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.data = make(map[string]*CacheEntry)
	cc.stats = &CacheStats{}
}

// Exists 检查缓存是否存在
func (cc *ConfigCache) Exists(key string) bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	entry, exists := cc.data[key]
	if !exists {
		return false
	}

	// 检查是否过期
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return false
	}

	return true
}

// GetKeys 获取所有缓存键
func (cc *ConfigCache) GetKeys() []string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	var keys []string
	for key := range cc.data {
		keys = append(keys, key)
	}

	return keys
}

// GetSize 获取缓存大小
func (cc *ConfigCache) GetSize() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return len(cc.data)
}

// GetStats 获取缓存统计
func (cc *ConfigCache) GetStats() *CacheStats {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	// 返回统计信息的副本
	stats := *cc.stats
	return &stats
}

// GetHits 获取缓存命中次数
func (cc *ConfigCache) GetHits() int64 {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.stats.Hits
}

// GetMisses 获取缓存未命中次数
func (cc *ConfigCache) GetMisses() int64 {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.stats.Misses
}

// GetHitRate 获取缓存命中率
func (cc *ConfigCache) GetHitRate() float64 {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	total := cc.stats.Hits + cc.stats.Misses
	if total == 0 {
		return 0.0
	}

	return float64(cc.stats.Hits) / float64(total) * 100.0
}

// evictLRU 淘汰最近最少使用的缓存条目
func (cc *ConfigCache) evictLRU() {
	if !cc.options.EnableLRU {
		// 如果不启用LRU，随机删除一个条目
		for key := range cc.data {
			delete(cc.data, key)
			break
		}
		return
	}

	// 找到最近最少使用的条目
	var oldestKey string
	var oldestTime time.Time
	var oldestAccessCount int64

	for key, entry := range cc.data {
		if oldestKey == "" ||
			entry.LastAccessedAt.Before(oldestTime) ||
			(entry.LastAccessedAt.Equal(oldestTime) && entry.AccessCount < oldestAccessCount) {
			oldestKey = key
			oldestTime = entry.LastAccessedAt
			oldestAccessCount = entry.AccessCount
		}
	}

	if oldestKey != "" {
		delete(cc.data, oldestKey)
	}
}

// startCleanup 启动清理协程
func (cc *ConfigCache) startCleanup() {
	ticker := time.NewTicker(cc.options.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		cc.cleanup()
	}
}

// cleanup 清理过期的缓存条目
func (cc *ConfigCache) cleanup() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	now := time.Now()
	var expiredKeys []string

	for key, entry := range cc.data {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// 删除过期的条目
	for _, key := range expiredKeys {
		delete(cc.data, key)
		cc.stats.Expirations++
	}
}

// calculateHash 计算数据哈希
func (cc *ConfigCache) calculateHash(value interface{}) string {
	// 将值转换为JSON字符串
	jsonData, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	// 计算MD5哈希
	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:])
}

// convertValue 转换值类型
func (cc *ConfigCache) convertValue(value interface{}, target interface{}) error {
	// 将值转换为JSON，然后解析到目标类型
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}

	return nil
}

// SetOptions 设置缓存选项
func (cc *ConfigCache) SetOptions(options *CacheOptions) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.options = options
}

// GetOptions 获取缓存选项
func (cc *ConfigCache) GetOptions() *CacheOptions {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.options
}

// Refresh 刷新缓存条目
func (cc *ConfigCache) Refresh(key string) bool {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	entry, exists := cc.data[key]
	if !exists {
		return false
	}

	// 更新访问时间和计数
	entry.LastAccessedAt = time.Now()
	entry.AccessCount++

	return true
}

// Touch 触摸缓存条目（更新访问时间）
func (cc *ConfigCache) Touch(key string) bool {
	return cc.Refresh(key)
}
