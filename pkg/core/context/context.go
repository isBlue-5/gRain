// Package context 提供框架统一的上下文管理
package context

import (
	"context"
	"time"
)

// key 是上下文中使用的非导出键类型
// 确保不会与其他包的键冲突
type key string

const (
	// 预定义的上下文键
	userIDKey      key = "userId"    // 用户ID
	requestIDKey   key = "requestId" // 请求ID
	traceIDKey     key = "traceId"   // 追踪ID
	startTimeKey   key = "startTime" // 请求开始时间
	sessionIDKey   key = "sessionId" // 会话ID
	clientIPKey    key = "clientIp"  // 客户端IP
	userAgentKey   key = "userAgent" // 用户代理
	pathKey        key = "path"      // 请求路径
	methodKey      key = "method"    // 请求方法
	transactionKey key = "tx"        // 事务对象
)

// AppContext 定义应用上下文接口
// 扩展标准的context.Context，添加应用特定功能
type AppContext interface {
	context.Context

	// 用户标识和身份认证
	SetUserID(userID string)
	GetUserID() string
	SetSessionID(sessionID string)
	GetSessionID() string

	// 请求跟踪
	SetRequestID(requestID string)
	GetRequestID() string
	SetTraceID(traceID string)
	GetTraceID() string
	SetStartTime(time time.Time)
	GetStartTime() time.Time
	GetElapsedTime() time.Duration

	// 请求元数据
	SetClientIP(ip string)
	GetClientIP() string
	SetUserAgent(userAgent string)
	GetUserAgent() string
	SetPath(path string)
	GetPath() string
	SetMethod(method string)
	GetMethod() string

	// 事务管理
	SetTransaction(tx interface{})
	GetTransaction() interface{}

	// 通用键值存储
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool

	// 上下文转换
	ToContext() context.Context
	WithValue(key string, value interface{}) AppContext
	WithCancel() (AppContext, context.CancelFunc)
	WithTimeout(timeout time.Duration) (AppContext, context.CancelFunc)
	WithDeadline(deadline time.Time) (AppContext, context.CancelFunc)
}

// appContext 是AppContext接口的基本实现
type appContext struct {
	ctx context.Context
}

// NewContext 创建新的应用上下文
func NewContext(parent context.Context) AppContext {
	if parent == nil {
		parent = context.Background()
	}
	return &appContext{ctx: parent}
}

// Deadline 实现Context接口
func (c *appContext) Deadline() (time.Time, bool) {
	return c.ctx.Deadline()
}

// Done 实现Context接口
func (c *appContext) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err 实现Context接口
func (c *appContext) Err() error {
	return c.ctx.Err()
}

// Value 实现Context接口
func (c *appContext) Value(k interface{}) interface{} {
	return c.ctx.Value(k)
}

// SetUserID 设置用户ID
func (c *appContext) SetUserID(userID string) {
	c.ctx = context.WithValue(c.ctx, userIDKey, userID)
}

// GetUserID 获取用户ID
func (c *appContext) GetUserID() string {
	if v := c.ctx.Value(userIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetSessionID 设置会话ID
func (c *appContext) SetSessionID(sessionID string) {
	c.ctx = context.WithValue(c.ctx, sessionIDKey, sessionID)
}

// GetSessionID 获取会话ID
func (c *appContext) GetSessionID() string {
	if v := c.ctx.Value(sessionIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetRequestID 设置请求ID
func (c *appContext) SetRequestID(requestID string) {
	c.ctx = context.WithValue(c.ctx, requestIDKey, requestID)
}

// GetRequestID 获取请求ID
func (c *appContext) GetRequestID() string {
	if v := c.ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetTraceID 设置追踪ID
func (c *appContext) SetTraceID(traceID string) {
	c.ctx = context.WithValue(c.ctx, traceIDKey, traceID)
}

// GetTraceID 获取追踪ID
func (c *appContext) GetTraceID() string {
	if v := c.ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetStartTime 设置请求开始时间
func (c *appContext) SetStartTime(time time.Time) {
	c.ctx = context.WithValue(c.ctx, startTimeKey, time)
}

// GetStartTime 获取请求开始时间
func (c *appContext) GetStartTime() time.Time {
	if v := c.ctx.Value(startTimeKey); v != nil {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

// GetElapsedTime 获取从开始到现在经过的时间
func (c *appContext) GetElapsedTime() time.Duration {
	startTime := c.GetStartTime()
	if startTime.IsZero() {
		return 0
	}
	return time.Since(startTime)
}

// SetClientIP 设置客户端IP
func (c *appContext) SetClientIP(ip string) {
	c.ctx = context.WithValue(c.ctx, clientIPKey, ip)
}

// GetClientIP 获取客户端IP
func (c *appContext) GetClientIP() string {
	if v := c.ctx.Value(clientIPKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetUserAgent 设置用户代理
func (c *appContext) SetUserAgent(userAgent string) {
	c.ctx = context.WithValue(c.ctx, userAgentKey, userAgent)
}

// GetUserAgent 获取用户代理
func (c *appContext) GetUserAgent() string {
	if v := c.ctx.Value(userAgentKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetPath 设置请求路径
func (c *appContext) SetPath(path string) {
	c.ctx = context.WithValue(c.ctx, pathKey, path)
}

// GetPath 获取请求路径
func (c *appContext) GetPath() string {
	if v := c.ctx.Value(pathKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetMethod 设置请求方法
func (c *appContext) SetMethod(method string) {
	c.ctx = context.WithValue(c.ctx, methodKey, method)
}

// GetMethod 获取请求方法
func (c *appContext) GetMethod() string {
	if v := c.ctx.Value(methodKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetTransaction 设置事务对象
func (c *appContext) SetTransaction(tx interface{}) {
	c.ctx = context.WithValue(c.ctx, transactionKey, tx)
}

// GetTransaction 获取事务对象
func (c *appContext) GetTransaction() interface{} {
	return c.ctx.Value(transactionKey)
}

// Set 设置自定义键值
func (c *appContext) Set(key string, value interface{}) {
	c.ctx = context.WithValue(c.ctx, key, value)
}

// Get 获取自定义键值
func (c *appContext) Get(key string) (interface{}, bool) {
	v := c.ctx.Value(key)
	return v, v != nil
}

// GetString 获取字符串类型的值
func (c *appContext) GetString(key string) string {
	if v, ok := c.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt 获取整数类型的值
func (c *appContext) GetInt(key string) int {
	if v, ok := c.Get(key); ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

// GetBool 获取布尔类型的值
func (c *appContext) GetBool(key string) bool {
	if v, ok := c.Get(key); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// ToContext 转换为标准context.Context
func (c *appContext) ToContext() context.Context {
	return c.ctx
}

// WithValue 创建包含新值的上下文副本
func (c *appContext) WithValue(key string, value interface{}) AppContext {
	return &appContext{
		ctx: context.WithValue(c.ctx, key, value),
	}
}

// WithCancel 创建可取消的上下文
func (c *appContext) WithCancel() (AppContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &appContext{ctx: ctx}, cancel
}

// WithTimeout 创建带超时的上下文
func (c *appContext) WithTimeout(timeout time.Duration) (AppContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	return &appContext{ctx: ctx}, cancel
}

// WithDeadline 创建带截止时间的上下文
func (c *appContext) WithDeadline(deadline time.Time) (AppContext, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(c.ctx, deadline)
	return &appContext{ctx: ctx}, cancel
}
