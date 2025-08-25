// Package cache 提供统一的缓存抽象层
package cache

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"sync"
	"time"
)

// 缓存项
type cacheItem struct {
	Value      interface{}
	Expiration int64
}

// 是否已过期
func (item *cacheItem) isExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	items      map[string]*cacheItem
	mu         sync.RWMutex
	options    Options
	janitor    *janitor
	size       int64
	serializer Serializer
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(opts Options) *MemoryCache {
	cache := &MemoryCache{
		items:   make(map[string]*cacheItem),
		options: opts,
		size:    0,
	}

	// 设置序列化器
	if opts.Serializer == nil {
		cache.serializer = &JSONSerializer{}
	} else {
		cache.serializer = opts.Serializer
	}

	// 启动清理协程
	if opts.DefaultTTL > 0 {
		cache.janitor = newJanitor(cache, opts.DefaultTTL/2)
		cache.janitor.start()
	}

	return cache
}

// Get 获取缓存值
func (c *MemoryCache) Get(ctx context.Context, key string, value interface{}) error {
	// 添加前缀
	key = c.addPrefix(key)

	c.mu.RLock()
	item, found := c.items[key]
	if !found {
		c.mu.RUnlock()
		return ErrCacheMiss
	}

	// 检查是否过期
	if item.isExpired() {
		c.mu.RUnlock()
		// 异步删除过期项
		go c.Delete(ctx, key)
		return ErrCacheMiss
	}

	c.mu.RUnlock()

	// 检查值类型
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrCacheValueInvalid
	}

	// 复制值
	if reflect.TypeOf(item.Value) == reflect.TypeOf(value).Elem() {
		// 类型相同，直接赋值
		reflect.ValueOf(value).Elem().Set(reflect.ValueOf(item.Value))
	} else {
		// 类型不同，尝试通过JSON转换
		data, err := c.serializer.Marshal(item.Value)
		if err != nil {
			return err
		}
		if err := c.serializer.Unmarshal(data, value); err != nil {
			return err
		}
	}

	return nil
}

// Set 设置缓存值
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 添加前缀
	key = c.addPrefix(key)

	// 计算过期时间
	var expiration int64
	if ttl == 0 {
		// 使用默认过期时间
		if c.options.DefaultTTL > 0 {
			expiration = time.Now().Add(c.options.DefaultTTL).UnixNano()
		}
	} else {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查容量
	if c.options.MaxEntries > 0 && len(c.items) >= c.options.MaxEntries {
		// 删除最早过期的项
		c.deleteOldest()
	}

	// 计算大小
	size := int64(0)
	if c.options.MaxSize > 0 {
		data, err := c.serializer.Marshal(value)
		if err != nil {
			return err
		}
		size = int64(len(data))

		// 检查单项大小
		if size > c.options.MaxSize {
			return ErrCacheValueInvalid
		}

		// 检查总大小
		if c.size+size > c.options.MaxSize {
			// 删除项直到有足够空间
			c.deleteUntilSize(c.size + size - c.options.MaxSize)
		}
	}

	// 存储项
	c.items[key] = &cacheItem{
		Value:      value,
		Expiration: expiration,
	}

	// 更新缓存大小
	if c.options.MaxSize > 0 {
		c.size += size
	}

	return nil
}

// Delete 删除缓存
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	// 添加前缀
	key = c.addPrefix(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	if item, found := c.items[key]; found {
		// 更新缓存大小
		if c.options.MaxSize > 0 {
			data, _ := c.serializer.Marshal(item.Value)
			c.size -= int64(len(data))
		}

		delete(c.items, key)
	}

	return nil
}

// Exists 检查缓存是否存在
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	// 添加前缀
	key = c.addPrefix(key)

	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return false, nil
	}

	// 检查是否过期
	if item.isExpired() {
		// 异步删除过期项
		go c.Delete(ctx, key)
		return false, nil
	}

	return true, nil
}

// Clear 清空缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.size = 0

	return nil
}

// GetMulti 批量获取缓存
func (c *MemoryCache) GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, key := range keys {
		// 添加前缀
		prefixedKey := c.addPrefix(key)

		item, found := c.items[prefixedKey]
		if !found {
			continue
		}

		// 检查是否过期
		if item.isExpired() {
			// 异步删除过期项
			go c.Delete(ctx, prefixedKey)
			continue
		}

		// 添加到结果
		result[key] = item.Value
	}

	return result, nil
}

// SetMulti 批量设置缓存
func (c *MemoryCache) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 计算过期时间
	var expiration int64
	if ttl == 0 {
		// 使用默认过期时间
		if c.options.DefaultTTL > 0 {
			expiration = time.Now().Add(c.options.DefaultTTL).UnixNano()
		}
	} else {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查容量
	if c.options.MaxEntries > 0 && len(c.items)+len(items) > c.options.MaxEntries {
		// 删除足够的项以容纳新项
		itemsToDelete := len(c.items) + len(items) - c.options.MaxEntries
		c.deleteOldestN(itemsToDelete)
	}

	// 计算总大小
	totalSize := int64(0)
	if c.options.MaxSize > 0 {
		for _, value := range items {
			data, err := c.serializer.Marshal(value)
			if err != nil {
				return err
			}
			totalSize += int64(len(data))
		}

		// 检查总大小
		if c.size+totalSize > c.options.MaxSize {
			// 删除项直到有足够空间
			c.deleteUntilSize(c.size + totalSize - c.options.MaxSize)
		}
	}

	// 存储项
	for key, value := range items {
		// 添加前缀
		prefixedKey := c.addPrefix(key)

		c.items[prefixedKey] = &cacheItem{
			Value:      value,
			Expiration: expiration,
		}
	}

	// 更新缓存大小
	if c.options.MaxSize > 0 {
		c.size += totalSize
	}

	return nil
}

// DeleteMulti 批量删除缓存
func (c *MemoryCache) DeleteMulti(ctx context.Context, keys []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		// 添加前缀
		prefixedKey := c.addPrefix(key)

		if item, found := c.items[prefixedKey]; found {
			// 更新缓存大小
			if c.options.MaxSize > 0 {
				data, _ := c.serializer.Marshal(item.Value)
				c.size -= int64(len(data))
			}

			delete(c.items, prefixedKey)
		}
	}

	return nil
}

// Incr 自增
func (c *MemoryCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	// 添加前缀
	key = c.addPrefix(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		// 不存在则创建
		c.items[key] = &cacheItem{
			Value:      delta,
			Expiration: 0,
		}
		return delta, nil
	}

	// 检查是否过期
	if item.isExpired() {
		// 删除过期项并创建新项
		delete(c.items, key)
		c.items[key] = &cacheItem{
			Value:      delta,
			Expiration: 0,
		}
		return delta, nil
	}

	// 检查值类型
	switch v := item.Value.(type) {
	case int:
		item.Value = int(int64(v) + delta)
		return int64(item.Value.(int)), nil
	case int8:
		item.Value = int8(int64(v) + delta)
		return int64(item.Value.(int8)), nil
	case int16:
		item.Value = int16(int64(v) + delta)
		return int64(item.Value.(int16)), nil
	case int32:
		item.Value = int32(int64(v) + delta)
		return int64(item.Value.(int32)), nil
	case int64:
		item.Value = v + delta
		return item.Value.(int64), nil
	case uint:
		item.Value = uint(int64(v) + delta)
		return int64(item.Value.(uint)), nil
	case uint8:
		item.Value = uint8(int64(v) + delta)
		return int64(item.Value.(uint8)), nil
	case uint16:
		item.Value = uint16(int64(v) + delta)
		return int64(item.Value.(uint16)), nil
	case uint32:
		item.Value = uint32(int64(v) + delta)
		return int64(item.Value.(uint32)), nil
	case uint64:
		item.Value = uint64(int64(v) + delta)
		return int64(item.Value.(uint64)), nil
	case float32:
		item.Value = float32(float64(v) + float64(delta))
		return int64(item.Value.(float32)), nil
	case float64:
		item.Value = v + float64(delta)
		return int64(item.Value.(float64)), nil
	default:
		return 0, ErrCacheValueInvalid
	}
}

// Decr 自减
func (c *MemoryCache) Decr(ctx context.Context, key string, delta int64) (int64, error) {
	return c.Incr(ctx, key, -delta)
}

// 添加前缀
func (c *MemoryCache) addPrefix(key string) string {
	if c.options.Prefix == "" {
		return key
	}
	return c.options.Prefix + ":" + key
}

// 删除最早过期的项
func (c *MemoryCache) deleteOldest() {
	var oldestKey string
	var oldestExpiration int64 = math.MaxInt64

	// 查找最早过期的项
	for key, item := range c.items {
		if item.Expiration > 0 && item.Expiration < oldestExpiration {
			oldestKey = key
			oldestExpiration = item.Expiration
		}
	}

	// 如果没有设置过期时间的项，删除任意一个
	if oldestKey == "" {
		for key := range c.items {
			oldestKey = key
			break
		}
	}

	// 删除项
	if oldestKey != "" {
		if c.options.MaxSize > 0 {
			data, _ := c.serializer.Marshal(c.items[oldestKey].Value)
			c.size -= int64(len(data))
		}
		delete(c.items, oldestKey)
	}
}

// 删除N个最早过期的项
func (c *MemoryCache) deleteOldestN(n int) {
	// 收集所有项的过期时间
	type keyExpiration struct {
		key        string
		expiration int64
	}

	expirations := make([]keyExpiration, 0, len(c.items))
	for key, item := range c.items {
		expirations = append(expirations, keyExpiration{
			key:        key,
			expiration: item.Expiration,
		})
	}

	// 按过期时间排序
	sort.Slice(expirations, func(i, j int) bool {
		// 未设置过期时间的项放在最后
		if expirations[i].expiration == 0 {
			return false
		}
		if expirations[j].expiration == 0 {
			return true
		}
		return expirations[i].expiration < expirations[j].expiration
	})

	// 删除前N个
	for i := 0; i < n && i < len(expirations); i++ {
		key := expirations[i].key
		if c.options.MaxSize > 0 {
			data, _ := c.serializer.Marshal(c.items[key].Value)
			c.size -= int64(len(data))
		}
		delete(c.items, key)
	}
}

// 删除项直到释放指定大小
func (c *MemoryCache) deleteUntilSize(sizeToFree int64) {
	// 收集所有项的大小和过期时间
	type itemInfo struct {
		key        string
		size       int64
		expiration int64
	}

	items := make([]itemInfo, 0, len(c.items))
	for key, item := range c.items {
		data, _ := c.serializer.Marshal(item.Value)
		items = append(items, itemInfo{
			key:        key,
			size:       int64(len(data)),
			expiration: item.Expiration,
		})
	}

	// 按过期时间排序
	sort.Slice(items, func(i, j int) bool {
		// 未设置过期时间的项放在最后
		if items[i].expiration == 0 {
			return false
		}
		if items[j].expiration == 0 {
			return true
		}
		return items[i].expiration < items[j].expiration
	})

	// 删除项直到释放足够空间
	freedSize := int64(0)
	for _, item := range items {
		if freedSize >= sizeToFree {
			break
		}
		delete(c.items, item.key)
		freedSize += item.size
		c.size -= item.size
	}
}

// 清理过期项
func (c *MemoryCache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	for key, item := range c.items {
		if item.Expiration > 0 && item.Expiration < now {
			if c.options.MaxSize > 0 {
				data, _ := c.serializer.Marshal(item.Value)
				c.size -= int64(len(data))
			}
			delete(c.items, key)
		}
	}
}

// janitor 清理器
type janitor struct {
	cache    *MemoryCache
	interval time.Duration
	stopChan chan bool
}

// 创建清理器
func newJanitor(cache *MemoryCache, interval time.Duration) *janitor {
	return &janitor{
		cache:    cache,
		interval: interval,
		stopChan: make(chan bool),
	}
}

// 启动清理器
func (j *janitor) start() {
	go func() {
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				j.cache.deleteExpired()
			case <-j.stopChan:
				return
			}
		}
	}()
}

// 停止清理器
func (j *janitor) stop() {
	j.stopChan <- true
}

// JSONSerializer JSON序列化器
type JSONSerializer struct{}

// Marshal 序列化
func (s *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 反序列化
func (s *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
