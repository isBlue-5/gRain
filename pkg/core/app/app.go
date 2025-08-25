// Package app 提供应用程序生命周期管理
package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/util/options"
)

// Service 服务接口
// 定义了可以被应用程序管理的服务生命周期
type Service interface {
	// Start 启动服务
	Start() error
	// Shutdown 优雅关闭服务
	Shutdown(ctx context.Context) error
	// Name 返回服务名称
	Name() string
}

// Config 应用程序配置
type Config struct {
	// 应用基本信息
	Name        string
	Version     string
	Description string

	// HTTP服务器配置
	Server struct {
		Host         string
		Port         int
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}

	// 关闭超时配置
	ShutdownTimeout time.Duration

	// 运行模式
	Mode string

	// 是否禁用Banner输出
	DisableBanner bool
}

// Option 应用程序选项函数类型
type Option = options.Option[Config]

// Application 应用程序结构
type Application struct {
	config      *Config
	router      *gin.Engine
	httpServer  *http.Server
	services    []Service
	controllers []interface{} // 存储注册的控制器
	shutdownCh  chan struct{}
	shutdownWg  sync.WaitGroup
	logger      *log.Logger
	initOnce    sync.Once
}

// defaultConfig 返回默认配置
func defaultConfig() *Config {
	cfg := &Config{
		Name:            "gRain-App",
		Version:         "1.0.0",
		Description:     "A Go Web Application",
		Mode:            gin.ReleaseMode,
		ShutdownTimeout: 30 * time.Second,
	}

	// 默认HTTP服务器配置
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 8080
	cfg.Server.ReadTimeout = 10 * time.Second
	cfg.Server.WriteTimeout = 10 * time.Second
	cfg.Server.IdleTimeout = 60 * time.Second

	return cfg
}

// WithConfig 设置应用程序配置
func WithConfig(config *Config) Option {
	return func(c *Config) {
		*c = *config
	}
}

// WithName 设置应用程序名称
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithVersion 设置应用程序版本
func WithVersion(version string) Option {
	return func(c *Config) {
		c.Version = version
	}
}

// WithDescription 设置应用程序描述
func WithDescription(description string) Option {
	return func(c *Config) {
		c.Description = description
	}
}

// WithHost 设置HTTP服务器主机
func WithHost(host string) Option {
	return func(c *Config) {
		c.Server.Host = host
	}
}

// WithPort 设置HTTP服务器端口
func WithPort(port int) Option {
	return func(c *Config) {
		c.Server.Port = port
	}
}

// WithMode 设置运行模式
func WithMode(mode string) Option {
	return func(c *Config) {
		c.Mode = mode
	}
}

// WithShutdownTimeout 设置关闭超时时间
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ShutdownTimeout = timeout
	}
}

// New 创建一个新的应用程序实例
func New(opts ...Option) *Application {
	// 应用默认配置
	cfg := defaultConfig()

	// 应用所有选项
	options.Apply(cfg, opts...)

	// 设置Gin模式
	gin.SetMode(cfg.Mode)

	// 创建应用程序
	app := &Application{
		config:     cfg,
		router:     gin.New(),
		shutdownCh: make(chan struct{}),
		logger:     log.New(os.Stdout, "[gRain] ", log.LstdFlags),
	}

	return app
}

// init 初始化应用程序
func (app *Application) init() {
	app.initOnce.Do(func() {
		// 设置HTTP服务器
		app.httpServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", app.config.Server.Host, app.config.Server.Port),
			Handler:      app.router,
			ReadTimeout:  app.config.Server.ReadTimeout,
			WriteTimeout: app.config.Server.WriteTimeout,
			IdleTimeout:  app.config.Server.IdleTimeout,
		}

		// 打印应用程序Banner
		if !app.config.DisableBanner {
			app.printBanner()
		}
	})
}

// printBanner 打印应用程序Banner
func (app *Application) printBanner() {
	banner := fmt.Sprintf(`
	  ________                .__        
	 /  _____/_______ _____  |__| ____  
	/   \  __\_  __ \\__  \ |  |/    \ 
	\    \_\  \  | \/ / __ \|  |   |  \
	 \______  /__|   (____  /__|___|  /
	        \/            \/        \/ v%s
	
	Application: %s
	%s
	`, app.config.Version, app.config.Name, app.config.Description)
	fmt.Println(banner)
}

// Router 返回Gin路由器实例
func (app *Application) Router() *gin.Engine {
	return app.router
}

// RegisterService 注册服务
func (app *Application) RegisterService(service Service) {
	app.services = append(app.services, service)
}

// RegisterController 注册控制器
// 这里提供一个占位实现，在实际代码生成系统中会被扩展
func (app *Application) RegisterController(controller interface{}) {
	if controller == nil {
		app.logger.Printf("WARN: Attempted to register nil controller")
		return
	}

	// 获取控制器的类型信息
	controllerType := reflect.TypeOf(controller)
	controllerName := controllerType.String()

	// 检查控制器是否已经注册
	if app.isControllerRegistered(controllerName) {
		app.logger.Printf("WARN: Controller %s is already registered", controllerName)
		return
	}

	// 注册控制器到应用
	app.controllers = append(app.controllers, controller)
	app.logger.Printf("INFO: Controller registered successfully: %s", controllerName)

	// 如果控制器实现了Service接口，也注册为服务
	if service, ok := controller.(Service); ok {
		app.RegisterService(service)
	}
}

// isControllerRegistered 检查控制器是否已注册
func (app *Application) isControllerRegistered(controllerName string) bool {
	for _, c := range app.controllers {
		if reflect.TypeOf(c).String() == controllerName {
			return true
		}
	}
	return false
}

// Run 运行应用程序
func (app *Application) Run() error {
	// 初始化应用程序
	app.init()

	// 设置信号处理
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// 启动所有服务
	for _, svc := range app.services {
		app.shutdownWg.Add(1)
		go func(s Service) {
			defer app.shutdownWg.Done()

			app.logger.Printf("Starting service: %s", s.Name())
			if err := s.Start(); err != nil {
				app.logger.Printf("Failed to start service %s: %v", s.Name(), err)
			}
		}(svc)
	}

	// 启动HTTP服务器
	app.shutdownWg.Add(1)
	go func() {
		defer app.shutdownWg.Done()

		app.logger.Printf("Starting HTTP server at %s", app.httpServer.Addr)
		if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.logger.Printf("HTTP server error: %v", err)
		}
	}()

	// 等待关闭信号
	select {
	case sig := <-signalCh:
		app.logger.Printf("Received shutdown signal: %v", sig)
	case <-app.shutdownCh:
		app.logger.Printf("Shutdown requested")
	}

	// 开始优雅关闭流程
	return app.Shutdown()
}

// Shutdown 优雅关闭应用程序
func (app *Application) Shutdown() error {
	app.logger.Printf("Initiating graceful shutdown")

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), app.config.ShutdownTimeout)
	defer cancel()

	// 首先关闭HTTP服务器，停止接受新请求
	app.logger.Printf("Shutting down HTTP server")
	if err := app.httpServer.Shutdown(ctx); err != nil {
		app.logger.Printf("HTTP server shutdown error: %v", err)
	}

	// 按相反顺序关闭服务
	for i := len(app.services) - 1; i >= 0; i-- {
		svc := app.services[i]
		app.logger.Printf("Shutting down service: %s", svc.Name())

		if err := svc.Shutdown(ctx); err != nil {
			app.logger.Printf("Service %s shutdown error: %v", svc.Name(), err)
		}
	}

	// 等待所有goroutine完成
	app.logger.Printf("Waiting for all operations to complete")

	// 使用带超时的等待，避免永久阻塞
	waitCh := make(chan struct{})
	go func() {
		app.shutdownWg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		app.logger.Printf("All operations completed successfully")
	case <-ctx.Done():
		app.logger.Printf("Shutdown timed out, some operations may not have completed")
	}

	app.logger.Printf("Graceful shutdown completed")
	return nil
}

// Close 请求应用程序关闭
func (app *Application) Close() {
	close(app.shutdownCh)
}

// Address 返回应用程序的监听地址
func (app *Application) Address() string {
	return app.httpServer.Addr
}
