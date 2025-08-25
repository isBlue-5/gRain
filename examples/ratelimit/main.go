package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/ratelimit"
)

func main() {
	// 创建限流配置
	config := ratelimit.DefaultRateLimitingConfig()

	// 创建存储
	store := ratelimit.NewMemoryStore()

	// 创建全局限流器
	globalLimiter, err := ratelimit.NewRateLimiterFromConfig(config)
	if err != nil {
		log.Fatalf("Failed to create global rate limiter: %v", err)
	}

	// 创建IP限流器
	ipLimiter, err := ratelimit.NewIPRateLimiter(config)
	if err != nil {
		log.Fatalf("Failed to create IP rate limiter: %v", err)
	}

	// 创建路由限流器（用于演示）
	_, err = ratelimit.NewRouteRateLimiters(config)
	if err != nil {
		log.Fatalf("Failed to create route rate limiters: %v", err)
	}

	// 创建路由
	r := gin.Default()

	// 应用全局限流中间件
	if config.Global.Enabled {
		r.Use(ratelimit.RateLimitMiddleware(globalLimiter, ratelimit.GlobalKeyFunc))
		log.Println("Global rate limiting enabled")
	}

	// 应用IP限流中间件
	if config.IP.Enabled {
		r.Use(ratelimit.RateLimitMiddleware(ipLimiter, ratelimit.IPKeyFunc))
		log.Println("IP rate limiting enabled")
	}

	// 公开接口（无特殊限流）
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Rate Limiting Demo",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 搜索接口（固定窗口限流）
	r.GET("/api/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
			return
		}

		// 模拟搜索处理时间
		time.Sleep(100 * time.Millisecond)

		c.JSON(http.StatusOK, gin.H{
			"query":   query,
			"results": []string{"result1", "result2", "result3"},
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 用户列表接口（滑动窗口限流）
	r.GET("/api/users", func(c *gin.Context) {
		// 模拟用户数据
		users := []gin.H{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
		}

		c.JSON(http.StatusOK, gin.H{
			"users": users,
			"time":  time.Now().Format(time.RFC3339),
		})
	})

	// 用户详情接口（令牌桶限流）
	r.GET("/api/users/:id", func(c *gin.Context) {
		id := c.Param("id")

		// 模拟用户详情
		user := gin.H{
			"id":    id,
			"name":  "User " + id,
			"email": "user" + id + "@example.com",
			"time":  time.Now().Format(time.RFC3339),
		}

		c.JSON(http.StatusOK, user)
	})

	// 创建用户接口（漏桶限流）
	r.POST("/api/users", func(c *gin.Context) {
		var user struct {
			Name  string `json:"name" binding:"required"`
			Email string `json:"email" binding:"required,email"`
		}

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 模拟创建用户
		time.Sleep(200 * time.Millisecond)

		c.JSON(http.StatusCreated, gin.H{
			"id":    123,
			"name":  user.Name,
			"email": user.Email,
			"time":  time.Now().Format(time.RFC3339),
		})
	})

	// 限流状态查询接口
	r.GET("/api/rate-limit/status", func(c *gin.Context) {
		ip := c.ClientIP()

		status := gin.H{
			"ip": gin.H{
				"limit":     ipLimiter.GetLimit(),
				"remaining": ipLimiter.GetQuota(ip),
				"period":    ipLimiter.GetPeriod().String(),
			},
			"global": gin.H{
				"limit":     globalLimiter.GetLimit(),
				"remaining": globalLimiter.GetQuota("global"),
				"period":    globalLimiter.GetPeriod().String(),
			},
			"time": time.Now().Format(time.RFC3339),
		}

		c.JSON(http.StatusOK, status)
	})

	// 限流重置接口（仅用于演示）
	r.POST("/api/rate-limit/reset", func(c *gin.Context) {
		ip := c.ClientIP()

		// 重置IP限流
		ipLimiter.Reset(ip)

		// 重置全局限流
		globalLimiter.Reset("global")

		c.JSON(http.StatusOK, gin.H{
			"message": "Rate limit reset successfully",
			"ip":      ip,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 演示不同限流算法的接口
	r.GET("/api/demo/fixed-window", func(c *gin.Context) {
		// 使用固定窗口限流器
		limiter := ratelimit.NewFixedWindowLimiter(store, 5, 10*time.Second)
		key := "demo:fixed-window:" + c.ClientIP()

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     "Fixed window rate limit exceeded",
				"limit":     limiter.GetLimit(),
				"remaining": limiter.GetQuota(key),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Fixed window request allowed",
			"remaining": limiter.GetQuota(key),
			"time":      time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/api/demo/sliding-window", func(c *gin.Context) {
		// 使用滑动窗口限流器
		limiter := ratelimit.NewSlidingWindowLimiter(store, 5, 10*time.Second, 5)
		key := "demo:sliding-window:" + c.ClientIP()

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     "Sliding window rate limit exceeded",
				"limit":     limiter.GetLimit(),
				"remaining": limiter.GetQuota(key),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Sliding window request allowed",
			"remaining": limiter.GetQuota(key),
			"time":      time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/api/demo/token-bucket", func(c *gin.Context) {
		// 使用令牌桶限流器
		limiter := ratelimit.NewTokenBucketLimiter(store, 10, 1.0) // 10个令牌，每秒1个
		key := "demo:token-bucket:" + c.ClientIP()

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     "Token bucket rate limit exceeded",
				"limit":     limiter.GetLimit(),
				"remaining": limiter.GetQuota(key),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Token bucket request allowed",
			"remaining": limiter.GetQuota(key),
			"time":      time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/api/demo/leaky-bucket", func(c *gin.Context) {
		// 使用漏桶限流器
		limiter := ratelimit.NewLeakyBucketLimiter(store, 10, 1.0) // 容量10，每秒1个
		key := "demo:leaky-bucket:" + c.ClientIP()

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     "Leaky bucket rate limit exceeded",
				"limit":     limiter.GetLimit(),
				"remaining": limiter.GetQuota(key),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Leaky bucket request allowed",
			"remaining": limiter.GetQuota(key),
			"time":      time.Now().Format(time.RFC3339),
		})
	})

	log.Println("Rate limiting server starting on :8080")
	log.Println("Available endpoints:")
	log.Println("  GET  /                    - Welcome page")
	log.Println("  GET  /api/search?q=test   - Search (fixed window)")
	log.Println("  GET  /api/users           - User list (sliding window)")
	log.Println("  GET  /api/users/:id       - User detail (token bucket)")
	log.Println("  POST /api/users           - Create user (leaky bucket)")
	log.Println("  GET  /api/rate-limit/status - Rate limit status")
	log.Println("  POST /api/rate-limit/reset  - Reset rate limits")
	log.Println("  GET  /api/demo/*          - Algorithm demos")

	r.Run(":8080")
}
