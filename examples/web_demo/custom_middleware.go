// Package main 演示如何自定义并注册中间件
package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/web/middleware"
)

// TimingMiddleware 统计请求耗时的中间件
// @return gin.HandlerFunc
func TimingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		log.Printf("[Timing] %s %s 耗时: %v", c.Request.Method, c.Request.URL.Path, duration)
	}
}

func init() {
	// 注册自定义中间件到全局注册中心
	middleware.DefaultRegistry.Register("timing", func() gin.HandlerFunc { return TimingMiddleware() })
}
