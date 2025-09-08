# 请求限流系统

gRain框架的请求限流系统提供了灵活、可配置的限流功能，支持多种限流算法和存储后端，保护API服务免受过多请求的影响。

## 功能特性

- **多种限流算法**：固定窗口、滑动窗口、令牌桶、漏桶
- **灵活存储后端**：内存存储、Redis存储（支持分布式部署）
- **多粒度限流**：全局、IP、用户、路由级别
- **中间件集成**：与Gin框架无缝集成
- **配置驱动**：支持JSON、YAML、环境变量配置
- **状态监控**：实时查询限流状态和剩余配额

## 核心组件

### 1. 限流器接口

```go
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
```

### 2. 限流算法

#### 固定窗口计数器
- **特点**：简单高效，但存在边界效应
- **适用场景**：对精度要求不高的场景
- **配置示例**：
```go
limiter := ratelimit.NewFixedWindowLimiter(store, 100, time.Minute)
```

#### 滑动窗口计数器
- **特点**：平滑限流，避免边界效应
- **适用场景**：需要更精确限流的场景
- **配置示例**：
```go
limiter := ratelimit.NewSlidingWindowLimiter(store, 100, time.Minute, 10)
```

#### 令牌桶算法
- **特点**：支持突发流量，平滑处理
- **适用场景**：需要处理突发流量的场景
- **配置示例**：
```go
limiter := ratelimit.NewTokenBucketLimiter(store, 100, 10.0) // 100个令牌，每秒10个
```

#### 漏桶算法
- **特点**：固定速率处理，平滑输出
- **适用场景**：需要固定处理速率的场景
- **配置示例**：
```go
limiter := ratelimit.NewLeakyBucketLimiter(store, 100, 10.0) // 容量100，每秒10个
```

### 3. 存储后端

#### 内存存储
```go
store := ratelimit.NewMemoryStore()
// 启动定期清理
store.StartCleanup(time.Minute)
```

#### Redis存储
```go
redisClient := // 你的Redis客户端
store := ratelimit.NewRedisStore(redisClient)
```

### 4. 中间件

#### 全局限流中间件
```go
globalLimiter := ratelimit.NewDefaultRateLimiter()
r.Use(ratelimit.RateLimitMiddleware(globalLimiter, ratelimit.GlobalKeyFunc))
```

#### IP限流中间件
```go
ipLimiter := ratelimit.NewDefaultRateLimiter()
r.Use(ratelimit.RateLimitMiddleware(ipLimiter, ratelimit.IPKeyFunc))
```

#### 路由级限流中间件
```go
rules := []ratelimit.RateLimitRule{
    {
        Path:      "/api/search",
        Method:    "GET",
        Limit:     10,
        Period:    time.Minute,
        Algorithm: "fixed_window",
        Key:       "ip",
    },
}
r.Use(ratelimit.RouteRateLimitMiddleware(limiterFactory, rules))
```

## 配置管理

### 配置结构
```go
type RateLimitingConfig struct {
    Enabled   bool   `json:"enabled" yaml:"enabled"`
    StoreType string `json:"storeType" yaml:"storeType"` // memory, redis
    
    // 全局限流设置
    Global struct {
        Enabled   bool          `json:"enabled" yaml:"enabled"`
        Limit     int           `json:"limit" yaml:"limit"`
        Period    time.Duration `json:"period" yaml:"period"`
        Algorithm string        `json:"algorithm" yaml:"algorithm"`
    } `json:"global" yaml:"global"`
    
    // IP限流设置
    IP struct {
        Enabled   bool          `json:"enabled" yaml:"enabled"`
        Limit     int           `json:"limit" yaml:"limit"`
        Period    time.Duration `json:"period" yaml:"period"`
        Algorithm string        `json:"algorithm" yaml:"algorithm"`
        Whitelist []string      `json:"whitelist" yaml:"whitelist"`
    } `json:"ip" yaml:"ip"`
    
    // 路由限流规则
    Routes []struct {
        Path      string        `json:"path" yaml:"path"`
        Method    string        `json:"method" yaml:"method"`
        Limit     int           `json:"limit" yaml:"limit"`
        Period    time.Duration `json:"period" yaml:"period"`
        Algorithm string        `json:"algorithm" yaml:"algorithm"`
        Key       string        `json:"key" yaml:"key"`
    } `json:"routes" yaml:"routes"`
}
```

### 环境变量配置
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_STORE=memory
RATE_LIMIT_GLOBAL_ENABLED=true
RATE_LIMIT_GLOBAL_LIMIT=1000
RATE_LIMIT_GLOBAL_PERIOD=1m
RATE_LIMIT_GLOBAL_ALGORITHM=token_bucket
RATE_LIMIT_IP_ENABLED=true
RATE_LIMIT_IP_LIMIT=100
RATE_LIMIT_IP_PERIOD=1m
RATE_LIMIT_IP_ALGORITHM=sliding_window
```

## 使用示例

### 基本使用
```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/isBlue-5/gRain/pkg/ratelimit"
)

func main() {
    r := gin.Default()
    
    // 创建限流器
    store := ratelimit.NewMemoryStore()
    limiter := ratelimit.NewFixedWindowLimiter(store, 60, time.Minute)
    
    // 应用限流中间件
    r.Use(ratelimit.RateLimitMiddleware(limiter, ratelimit.IPKeyFunc))
    
    // 路由
    r.GET("/api/test", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "success"})
    })
    
    r.Run(":8080")
}
```

### 配置驱动使用
```go
func main() {
    // 加载配置
    config := ratelimit.DefaultRateLimitingConfig()
    
    // 创建限流器
    globalLimiter, _ := ratelimit.NewRateLimiterFromConfig(config)
    ipLimiter, _ := ratelimit.NewIPRateLimiter(config)
    
    r := gin.Default()
    
    // 应用限流中间件
    if config.Global.Enabled {
        r.Use(ratelimit.RateLimitMiddleware(globalLimiter, ratelimit.GlobalKeyFunc))
    }
    
    if config.IP.Enabled {
        r.Use(ratelimit.RateLimitMiddleware(ipLimiter, ratelimit.IPKeyFunc))
    }
    
    r.Run(":8080")
}
```

### 自定义键生成
```go
// 基于用户的限流
userKeyFunc := func(c *gin.Context) string {
    if userID, exists := c.Get("user_id"); exists {
        return "user:" + userID.(string)
    }
    return "anonymous"
}

r.Use(ratelimit.RateLimitMiddleware(limiter, userKeyFunc))

// 组合键生成
combinedKeyFunc := ratelimit.CombinedKeyFunc(
    ratelimit.IPKeyFunc,
    ratelimit.PathKeyFunc,
)

r.Use(ratelimit.RateLimitMiddleware(limiter, combinedKeyFunc))
```

## 高级功能

### 1. 动态限流
```go
// 根据系统负载动态调整限流阈值
type DynamicLimiter struct {
    baseLimiter RateLimiter
    metrics     MetricsProvider
}

func (l *DynamicLimiter) Allow(key string) bool {
    cpuUsage := l.metrics.GetCPUUsage()
    
    // 系统负载高时降低限流阈值
    if cpuUsage > 80 {
        // 使用更严格的限流
        return false
    }
    
    return l.baseLimiter.Allow(key)
}
```

### 2. 白名单支持
```go
func whitelistKeyFunc(whitelist []string) func(*gin.Context) string {
    return func(c *gin.Context) string {
        ip := c.ClientIP()
        
        // 检查白名单
        for _, whiteIP := range whitelist {
            if ip == whiteIP {
                return "whitelist"
            }
        }
        
        return ip
    }
}
```

### 3. 限流状态监控
```go
// 查询限流状态
func getRateLimitStatus(c *gin.Context) {
    ip := c.ClientIP()
    
    status := gin.H{
        "ip": gin.H{
            "limit":     ipLimiter.GetLimit(),
            "remaining": ipLimiter.GetQuota(ip),
            "period":    ipLimiter.GetPeriod().String(),
        },
    }
    
    c.JSON(200, status)
}
```

## 最佳实践

### 1. 算法选择
- **固定窗口**：简单场景，对精度要求不高
- **滑动窗口**：需要平滑限流的场景
- **令牌桶**：需要处理突发流量的场景
- **漏桶**：需要固定处理速率的场景

### 2. 存储选择
- **内存存储**：单实例应用，性能要求高
- **Redis存储**：分布式应用，需要共享限流状态

### 3. 配置建议
- 根据API的QPS和响应时间设置合理的限流阈值
- 为不同类型的API设置不同的限流策略
- 监控限流效果，及时调整配置

### 4. 监控和告警
- 监控限流触发频率
- 设置限流告警阈值
- 记录限流日志用于分析

## 扩展开发

### 1. 自定义限流算法
```go
type CustomLimiter struct {
    store RateLimiterStore
    // 自定义字段
}

func (l *CustomLimiter) Allow(key string) bool {
    // 实现自定义限流逻辑
    return true
}

// 实现其他接口方法...
```

### 2. 自定义存储后端
```go
type CustomStore struct {
    // 自定义存储实现
}

func (s *CustomStore) Incr(key string, ttl time.Duration) (int, error) {
    // 实现自定义存储逻辑
    return 0, nil
}

// 实现其他接口方法...
```

### 3. 集成监控系统
```go
type MonitoredLimiter struct {
    limiter RateLimiter
    metrics MetricsCollector
}

func (l *MonitoredLimiter) Allow(key string) bool {
    allowed := l.limiter.Allow(key)
    
    // 记录指标
    l.metrics.RecordRateLimit(key, allowed)
    
    return allowed
}
```

## 故障排除

### 常见问题

1. **限流不生效**
   - 检查配置是否正确加载
   - 确认中间件顺序是否正确
   - 验证键生成函数是否返回预期值

2. **性能问题**
   - 使用内存存储替代Redis存储
   - 减少限流检查频率
   - 优化键生成逻辑

3. **分布式限流不一致**
   - 确保使用Redis存储
   - 检查Redis连接和配置
   - 验证时钟同步

### 调试技巧

1. **启用详细日志**
```go
// 在限流中间件中添加日志
log.Printf("Rate limit check: key=%s, allowed=%v", key, allowed)
```

2. **监控限流状态**
```go
// 定期输出限流统计
go func() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        log.Printf("Rate limit stats: %+v", getStats())
    }
}()
```

3. **压力测试**
```bash
# 使用ab进行压力测试
ab -n 1000 -c 10 http://localhost:8080/api/test
```

## 总结

gRain的请求限流系统提供了完整的限流解决方案，支持多种算法和存储后端，可以满足不同场景的需求。通过合理的配置和使用，可以有效保护API服务，提升系统稳定性。 