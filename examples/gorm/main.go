// Package gorm 演示GORM适配器的使用示例
package gorm

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/data"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 应用程序入口
func RunExample() {
	// 设置上下文，支持取消信号
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 捕获终止信号
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		log.Println("接收到终止信号，准备退出...")
		cancel()
	}()

	// 初始化GORM
	db, err := initGorm()
	if err != nil {
		log.Fatalf("初始化GORM失败: %v", err)
	}

	// 初始化数据库表结构
	if err := initDatabase(db); err != nil {
		log.Fatalf("初始化数据库表结构失败: %v", err)
	}

	// 创建GORM会话
	session := data.NewGormSession(db)

	// 创建用户仓库
	userRepo := NewGormUserRepository(session)

	// 创建用户服务
	userService := NewUserService(userRepo)

	// 创建用户控制器
	userController := NewUserController(userService)

	// 创建Gin引擎
	router := gin.Default()

	// 注册API路由
	apiGroup := router.Group("/api")

	// 用户API
	usersGroup := apiGroup.Group("/users")
	{
		usersGroup.GET("/", userController.GetAllUsers)
		usersGroup.GET("/:id", userController.GetUserByID)
		usersGroup.POST("/", userController.CreateUser)
		usersGroup.PUT("/:id", userController.UpdateUser)
		usersGroup.DELETE("/:id", userController.DeleteUser)
	}

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 在协程中启动服务器
	go func() {
		log.Println("启动HTTP服务器在 :8080...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待取消信号
	<-ctx.Done()

	// 优雅关闭服务器
	log.Println("正在关闭HTTP服务器...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}

// 初始化GORM
func initGorm() (*gorm.DB, error) {
	// 配置GORM日志
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// 连接数据库
	dsn := "user:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// 初始化数据库表结构
func initDatabase(db *gorm.DB) error {
	// 自动迁移表结构
	return db.AutoMigrate(&User{})
}
