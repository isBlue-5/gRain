// Package middleware 提供中间件注册与统一管理能力
package middleware

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// MiddlewareFunc 定义中间件函数类型
// 返回 gin.HandlerFunc 的工厂函数，便于按需生成
type MiddlewareFunc func() gin.HandlerFunc

// Registry 中间件注册中心，支持自定义中间件注册、查找、链式组合
type Registry struct {
	mu          sync.RWMutex
	middlewares map[string]MiddlewareFunc
}

// DefaultRegistry 全局默认注册中心
var DefaultRegistry = NewRegistry()

// NewRegistry 创建新的中间件注册中心
func NewRegistry() *Registry {
	return &Registry{
		middlewares: make(map[string]MiddlewareFunc),
	}
}

// Register 注册一个中间件
// name: 唯一名称，fn: 返回 gin.HandlerFunc 的工厂函数
func (r *Registry) Register(name string, fn MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares[name] = fn
}

// Get 获取已注册的中间件
func (r *Registry) Get(name string) (MiddlewareFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.middlewares[name]
	return fn, ok
}

// Chain 链式组合多个中间件
// names: 中间件名称列表，返回 gin.HandlerFunc 切片
func (r *Registry) Chain(names ...string) []gin.HandlerFunc {
	var chain []gin.HandlerFunc
	for _, name := range names {
		if fn, ok := r.Get(name); ok {
			chain = append(chain, fn())
		}
	}
	return chain
}
