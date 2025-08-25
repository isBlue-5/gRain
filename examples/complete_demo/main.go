//go:generate ginframe-gen --input=. --output=./generated --prefix=frame:

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"complete_demo/config"
	"complete_demo/database"
	"complete_demo/generated" // 导入gRain生成的代码

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	dbManager, err := database.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 自动迁移数据库
	if err := dbManager.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 填充测试数据
	if err := dbManager.SeedData(); err != nil {
		log.Printf("Warning: 填充测试数据失败: %v", err)
	}

	// 设置Gin模式
	if cfg.Log.Level == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin路由器
	router := gin.Default()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 健康检查路由
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 数据库健康检查
	router.GET("/health/db", func(c *gin.Context) {
		if err := dbManager.HealthCheck(); err != nil {
			c.JSON(500, gin.H{"status": "error", "message": "数据库连接失败"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 🎯 gRain框架的魔法时刻！
	// 只需要调用这一个函数，所有注解的路由自动注册
	// 包括：
	// - 用户控制器路由 (/api/users/*)
	// - 产品控制器路由 (/api/products/*)
	// - 认证相关路由 (/api/auth/*)
	// - 统计相关路由 (/api/stats/*)
	log.Println("🚀 启动gRain框架自动路由注册...")
	generated.RegisterAllgRainRoutes(router, cfg, dbManager)
	log.Println("✅ gRain框架自动路由注册完成！")

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 启动服务器
	go func() {
		log.Printf("启动服务器，监听端口: %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}

	log.Println("服务器已关闭")

	// 关闭数据库连接
	if err := dbManager.Close(); err != nil {
		log.Printf("关闭数据库连接失败: %v", err)
	}
}

// 🎉 gRain框架的优势总结：
//
// 1. **真正的自动化**：
//    - 不需要手动注册每个路由
//    - 不需要手动创建路由组
//    - 不需要手动指定HTTP方法
//
// 2. **注解驱动**：
//    - // frame:controller(path="/api/users")
//    - // frame:route(method="GET", path="/")
//    - 注解就是配置，配置就是代码
//
// 3. **依赖注入**：
//    - 自动创建仓库层
//    - 自动创建服务层
//    - 自动创建控制器层
//
// 4. **零配置启动**：
//    - 只需要调用 generated.RegisterAllgRainRoutes(router, cfg, db)
//    - 所有路由自动注册
//    - 所有依赖自动注入
//
// 5. **生产级特性**：
//    - 自动参数验证
//    - 自动权限控制
//    - 自动限流控制
//    - 自动事务管理
//
// 这就是gRain框架的魔力：让开发者专注于业务逻辑，框架处理所有样板代码！
