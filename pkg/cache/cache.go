// Package cache 提供统一的缓存抽象层
package cache

import (
	"context"
	"errors"
	"time"
)

// 常见错误定义
var (
	// ErrCacheMiss 缓存未命中
	ErrCacheMiss = errors.New("缓存未命中")

	// ErrCacheKeyNotFound 缓存键不存在
	ErrCacheKeyNotFound = errors.New("缓存键不存在")

	// ErrCacheKeyInvalid 缓存键无效
	ErrCacheKeyInvalid = errors.New("缓存键无效")

	// ErrCacheValueInvalid 缓存值无效
	ErrCacheValueInvalid = errors.New("缓存值无效")

	// ErrCacheFull 缓存已满
	ErrCacheFull = errors.New("缓存已满")
)

// Cache 定义缓存接口
type Cache interface {
	// Get 获取缓存值
	// key: 缓存键
	// value: 缓存值（指针类型，用于接收结果）
	// 返回: 错误信息，如果键不存在则返回ErrCacheMiss
	Get(ctx context.Context, key string, value interface{}) error

	// Set 设置缓存值
	// key: 缓存键
	// value: 缓存值
	// ttl: 过期时间，如果为0则永不过期
	// 返回: 错误信息
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存
	// key: 缓存键
	// 返回: 错误信息
	Delete(ctx context.Context, key string) error

	// Exists 检查缓存是否存在
	// key: 缓存键
	// 返回: 是否存在，错误信息
	Exists(ctx context.Context, key string) (bool, error)

	// Clear 清空缓存
	// 返回: 错误信息
	Clear(ctx context.Context) error

	// GetMulti 批量获取缓存
	// keys: 缓存键列表
	// 返回: 缓存值映射，错误信息
	GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error)

	// SetMulti 批量设置缓存
	// items: 缓存键值对
	// ttl: 过期时间
	// 返回: 错误信息
	SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

	// DeleteMulti 批量删除缓存
	// keys: 缓存键列表
	// 返回: 错误信息
	DeleteMulti(ctx context.Context, keys []string) error

	// Incr 自增
	// key: 缓存键
	// delta: 增量
	// 返回: 增加后的值，错误信息
	Incr(ctx context.Context, key string, delta int64) (int64, error)

	// Decr 自减
	// key: 缓存键
	// delta: 减量
	// 返回: 减少后的值，错误信息
	Decr(ctx context.Context, key string, delta int64) (int64, error)
}

// Options 缓存选项
type Options struct {
	// Prefix 缓存键前缀
	Prefix string

	// DefaultTTL 默认过期时间
	DefaultTTL time.Duration

	// MaxEntries 最大条目数
	MaxEntries int

	// MaxSize 最大缓存大小（字节）
	MaxSize int64

	// Serializer 序列化器
	Serializer Serializer
}

// Option 缓存选项函数
type Option func(*Options)

// WithPrefix 设置缓存键前缀
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

// WithDefaultTTL 设置默认过期时间
func WithDefaultTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultTTL = ttl
	}
}

// WithMaxEntries 设置最大条目数
func WithMaxEntries(maxEntries int) Option {
	return func(o *Options) {
		o.MaxEntries = maxEntries
	}
}

// WithMaxSize 设置最大缓存大小
func WithMaxSize(maxSize int64) Option {
	return func(o *Options) {
		o.MaxSize = maxSize
	}
}

// WithSerializer 设置序列化器
func WithSerializer(serializer Serializer) Option {
	return func(o *Options) {
		o.Serializer = serializer
	}
}

// Serializer 序列化器接口
type Serializer interface {
	// Marshal 序列化
	Marshal(v interface{}) ([]byte, error)

	// Unmarshal 反序列化
	Unmarshal(data []byte, v interface{}) error
}

// DefaultOptions 默认缓存选项
var DefaultOptions = Options{
	Prefix:     "",
	DefaultTTL: 5 * time.Minute,
	MaxEntries: 10000,
	MaxSize:    100 * 1024 * 1024, // 100MB
}

// CacheProvider 缓存提供者接口
type CacheProvider interface {
	// GetCache 获取缓存实例
	GetCache(name string) Cache

	// CreateCache 创建缓存实例
	CreateCache(name string, opts ...Option) (Cache, error)

	// RemoveCache 移除缓存实例
	RemoveCache(name string) error

	// HasCache 检查缓存实例是否存在
	HasCache(name string) bool

	// GetCacheNames 获取所有缓存实例名称
	GetCacheNames() []string
}

// CacheManager 缓存管理器，默认的缓存提供者实现
type CacheManager struct {
	caches map[string]Cache
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	return &CacheManager{
		caches: make(map[string]Cache),
	}
}

// GetCache 获取缓存实例
func (m *CacheManager) GetCache(name string) Cache {
	return m.caches[name]
}

// CreateCache 创建缓存实例
func (m *CacheManager) CreateCache(name string, opts ...Option) (Cache, error) {
	if m.HasCache(name) {
		return m.caches[name], nil
	}

	// 应用选项
	options := DefaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	// 创建内存缓存
	cache := NewMemoryCache(options)
	m.caches[name] = cache

	return cache, nil
}

// RemoveCache 移除缓存实例
func (m *CacheManager) RemoveCache(name string) error {
	if !m.HasCache(name) {
		return ErrCacheKeyNotFound
	}

	delete(m.caches, name)
	return nil
}

// HasCache 检查缓存实例是否存在
func (m *CacheManager) HasCache(name string) bool {
	_, exists := m.caches[name]
	return exists
}

// GetCacheNames 获取所有缓存实例名称
func (m *CacheManager) GetCacheNames() []string {
	names := make([]string, 0, len(m.caches))
	for name := range m.caches {
		names = append(names, name)
	}
	return names
}

// DefaultCacheManager 默认缓存管理器
var DefaultCacheManager = NewCacheManager()
