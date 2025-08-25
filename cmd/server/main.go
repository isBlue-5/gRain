// Package main 应用程序入口点
package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/core/app"
	"github.com/isBlue-5/grain/pkg/core/config"
	"github.com/isBlue-5/grain/pkg/core/errors"
)

// AppConfig 应用程序配置
type AppConfig struct {
	Server struct {
		Host string `json:"host" yaml:"host" env:"SERVER_HOST" default:"0.0.0.0"`
		Port int    `json:"port" yaml:"port" env:"SERVER_PORT" default:"8080"`
	} `json:"server" yaml:"server"`

	Log struct {
		Level  string `json:"level" yaml:"level" env:"LOG_LEVEL" default:"info"`
		Format string `json:"format" yaml:"format" env:"LOG_FORMAT" default:"json"`
	} `json:"log" yaml:"log"`

	Features struct {
		EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics" env:"ENABLE_METRICS" default:"false"`
		EnableTracing bool `json:"enableTracing" yaml:"enableTracing" env:"ENABLE_TRACING" default:"false"`
	} `json:"features" yaml:"features"`
}

func main() {
	// 创建配置加载器
	cfg := &AppConfig{}
	loader := config.NewConfigLoader(
		&config.EnvConfigSource{Prefix: "GRAIN"},
	)

	// 加载配置
	if err := loader.Load(cfg); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 创建应用程序
	application := app.New(
		app.WithName("gRain-Demo"),
		app.WithVersion("0.1.0"),
		app.WithDescription("gRain Framework Demo Application"),
		app.WithHost(cfg.Server.Host),
		app.WithPort(cfg.Server.Port),
		app.WithMode(gin.DebugMode), // 开发模式
		app.WithShutdownTimeout(10*time.Second),
	)

	// 获取路由器并注册路由
	router := application.Router()

	// 注册中间件
	router.Use(gin.Logger(), gin.Recovery())

	// 路由组
	apiGroup := router.Group("/api")
	{
		// 健康检查
		apiGroup.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
				"time":   time.Now().Format(time.RFC3339),
			})
		})

		// 示例错误处理
		apiGroup.GET("/error", func(c *gin.Context) {
			// 创建应用错误
			err := errors.NewError(errors.CodeInvalidArgument, "示例错误")
			err.WithStatus(errors.StatusBadRequest).
				AddDetail("timestamp", time.Now().Unix()).
				AddDetail("path", c.Request.URL.Path)

			// 返回错误响应
			c.JSON(err.Status(), err.ToMap())
		})

		// 模拟不同错误类型
		errorsGroup := apiGroup.Group("/errors")
		{
			// 未找到资源
			errorsGroup.GET("/not-found", func(c *gin.Context) {
				c.JSON(errors.StatusNotFound, errors.ErrNotFound.ToMap())
			})

			// 未授权
			errorsGroup.GET("/unauthorized", func(c *gin.Context) {
				c.JSON(errors.StatusUnauthorized, errors.ErrUnauthorized.ToMap())
			})

			// 权限不足
			errorsGroup.GET("/forbidden", func(c *gin.Context) {
				c.JSON(errors.StatusForbidden, errors.ErrForbidden.ToMap())
			})

			// 服务器内部错误
			errorsGroup.GET("/internal", func(c *gin.Context) {
				c.JSON(errors.StatusInternalServerError, errors.ErrInternal.ToMap())
			})
		}

		// 配置信息
		apiGroup.GET("/config", func(c *gin.Context) {
			// 注意：在实际应用中，不应该暴露完整配置
			c.JSON(200, gin.H{
				"server": gin.H{
					"host": cfg.Server.Host,
					"port": cfg.Server.Port,
				},
				"log": gin.H{
					"level":  cfg.Log.Level,
					"format": cfg.Log.Format,
				},
				"features": gin.H{
					"enableMetrics": cfg.Features.EnableMetrics,
					"enableTracing": cfg.Features.EnableTracing,
				},
			})
		})

		// 环境信息
		apiGroup.GET("/env", func(c *gin.Context) {
			hostname, _ := os.Hostname()
			c.JSON(200, gin.H{
				"hostname": hostname,
				"os":       os.Getenv("GOOS"),
				"arch":     os.Getenv("GOARCH"),
				"version":  "0.1.0",
				"uptime":   time.Since(startTime).String(),
			})
		})
	}

	// 运行应用程序
	log.Printf("Starting gRain Demo Application...")
	if err := application.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

// 记录应用启动时间
var startTime = time.Now()
