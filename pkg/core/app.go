package core

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// App gRain应用实例
type App struct {
	config         *Config
	router         *gin.Engine
	autoGenerator  *AutoGenerator
	swaggerServer  *SwaggerServer
	server         *http.Server
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// Config 应用配置
type Config struct {
	Host            string
	Port            string
	Mode            string
	Debug           bool
	ShutdownTimeout time.Duration
	AutoGenerate    bool // 是否自动生成代码
}

// Option 应用选项
type Option func(*Config)

// WithHost 设置主机地址
func WithHost(host string) Option {
	return func(c *Config) {
		c.Host = host
	}
}

// WithPort 设置端口
func WithPort(port string) Option {
	return func(c *Config) {
		c.Port = port
	}
}

// WithMode 设置运行模式
func WithMode(mode string) Option {
	return func(c *Config) {
		c.Mode = mode
	}
}

// WithDebug 设置调试模式
func WithDebug(debug bool) Option {
	return func(c *Config) {
		c.Debug = debug
	}
}

// WithShutdownTimeout 设置关闭超时
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ShutdownTimeout = timeout
	}
}

// WithAutoGenerate 设置是否自动生成代码
func WithAutoGenerate(autoGenerate bool) Option {
	return func(c *Config) {
		c.AutoGenerate = autoGenerate
	}
}

// New 创建新的应用实例
func New(options ...Option) *App {
	// 默认配置
	config := &Config{
		Host:            "0.0.0.0",
		Port:            "8080",
		Mode:            "release",
		Debug:           false,
		ShutdownTimeout: 30 * time.Second,
		AutoGenerate:    true, // 默认启用自动生成
	}

	// 应用选项
	for _, option := range options {
		option(config)
	}

	// 设置Gin模式
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin路由器
	router := gin.Default()

	// 创建应用实例
	app := &App{
		config: config,
		router: router,
	}

	// 创建关闭上下文
	app.shutdownCtx, app.shutdownCancel = context.WithCancel(context.Background())

	return app
}

// Initialize 初始化应用
func (app *App) Initialize() error {
	log.Println("🚀 初始化gRain应用...")

	// 如果启用自动生成，则执行代码生成
	if app.config.AutoGenerate {
		if err := app.autoGenerateCode(); err != nil {
			return fmt.Errorf("自动代码生成失败: %w", err)
		}
	}

	// 注册基础路由
	app.registerBaseRoutes()

	// 注册Swagger路由
	if app.swaggerServer != nil {
		app.swaggerServer.RegisterRoutes(app.router)
	}

	log.Println("✅ gRain应用初始化完成")
	return nil
}

// autoGenerateCode 自动生成代码
func (app *App) autoGenerateCode() error {
	log.Println("🔧 开始自动代码生成...")

	// 获取工作目录
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	// 创建自动生成器
	app.autoGenerator = NewAutoGenerator(workDir)

	// 执行自动生成
	if err := app.autoGenerator.AutoGenerate(); err != nil {
		return fmt.Errorf("自动生成失败: %w", err)
	}

	// 获取Swagger服务器
	app.swaggerServer = app.autoGenerator.GetSwaggerServer()

	log.Println("✅ 自动代码生成完成")
	return nil
}

// registerBaseRoutes 注册基础路由
func (app *App) registerBaseRoutes() {
	// 健康检查
	app.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"framework": "gRain",
			"version":   "1.0.0",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 框架信息
	app.router.GET("/info", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":        "gRain Framework",
			"description": "Go企业级Web框架 - 约定大于配置，性能不受影响",
			"version":     "1.0.0",
			"go_version":  "1.21+",
			"features": []string{
				"注解驱动开发",
				"零反射依赖注入",
				"自动化路由注册",
				"智能Swagger文档",
				"增强上下文管理",
				"类型安全配置",
			},
		})
	})
}

// AddRoute 添加路由
func (app *App) AddRoute(method, path string, handler gin.HandlerFunc) {
	switch method {
	case "GET":
		app.router.GET(path, handler)
	case "POST":
		app.router.POST(path, handler)
	case "PUT":
		app.router.PUT(path, handler)
	case "DELETE":
		app.router.DELETE(path, handler)
	case "PATCH":
		app.router.PATCH(path, handler)
	default:
		log.Printf("⚠️  不支持的路由方法: %s", method)
	}
}

// AddMiddleware 添加中间件
func (app *App) AddMiddleware(middleware gin.HandlerFunc) {
	app.router.Use(middleware)
}

// GetRouter 获取Gin路由器
func (app *App) GetRouter() *gin.Engine {
	return app.router
}

// Run 运行应用
func (app *App) Run() error {
	// 初始化应用
	if err := app.Initialize(); err != nil {
		return err
	}

	// 创建HTTP服务器
	app.server = &http.Server{
		Addr:         app.config.Host + ":" + app.config.Port,
		Handler:      app.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器
	go func() {
		log.Printf("🌐 启动HTTP服务器: %s", app.server.Addr)
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🔄 正在关闭服务器...")

	// 优雅关闭
	return app.Shutdown()
}

// Shutdown 优雅关闭应用
func (app *App) Shutdown() error {
	// 设置关闭超时
	ctx, cancel := context.WithTimeout(app.shutdownCtx, app.config.ShutdownTimeout)
	defer cancel()

	// 关闭HTTP服务器
	if app.server != nil {
		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("关闭HTTP服务器时出错: %v", err)
		}
	}

	// 取消关闭上下文
	app.shutdownCancel()

	log.Println("✅ 服务器已关闭")
	return nil
}

// GetConfig 获取应用配置
func (app *App) GetConfig() *Config {
	return app.config
}

// GetAutoGenerator 获取自动生成器
func (app *App) GetAutoGenerator() *AutoGenerator {
	return app.autoGenerator
}

// GetSwaggerServer 获取Swagger服务器
func (app *App) GetSwaggerServer() *SwaggerServer {
	return app.swaggerServer
}
