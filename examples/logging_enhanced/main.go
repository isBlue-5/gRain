package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UserService 用户服务
type UserService struct{}

// GetUserByID 根据ID获取用户（基本日志）
// frame:log(level="info", message="获取用户信息", includeParams=true, includeResult=true, includeTime=true)
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*User, error) {
	// 模拟数据库查询
	time.Sleep(100 * time.Millisecond)

	user := &User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}

	return user, nil
}

// CreateUser 创建用户（自定义字段日志）
// frame:log(level="info", message="创建新用户", fields={"operation=create", "service=user"}, includeParams=true)
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	// 模拟创建用户
	time.Sleep(200 * time.Millisecond)

	// 模拟成功
	return nil
}

// UpdateUser 更新用户（动态日志级别）
// frame:log(dynamicLevel="getLogLevel", message="更新用户信息", includeParams=true, includeResult=true)
func (s *UserService) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error {
	// 模拟更新用户
	time.Sleep(150 * time.Millisecond)

	// 根据更新内容决定日志级别
	if len(updates) > 5 {
		// 大量更新，使用警告级别
		return nil
	}

	return nil
}

// DeleteUser 删除用户（条件日志）
// frame:log(level="warn", message="删除用户", condition="isSensitiveOperation", includeParams=true)
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	// 模拟删除用户
	time.Sleep(100 * time.Millisecond)

	return nil
}

// BatchProcessUsers 批量处理用户（采样日志）
// frame:log(level="info", message="批量处理用户", sampling=0.1, includeParams=true)
func (s *UserService) BatchProcessUsers(ctx context.Context, userIDs []string) error {
	// 模拟批量处理
	time.Sleep(500 * time.Millisecond)

	return nil
}

// GetUserStats 获取用户统计（MDC日志）
// frame:log(level="debug", message="获取用户统计", mdc={"request_id", "user_id"}, includeResult=true)
func (s *UserService) GetUserStats(ctx context.Context) (*UserStats, error) {
	// 模拟获取统计信息
	time.Sleep(50 * time.Millisecond)

	stats := &UserStats{
		TotalUsers:  1000,
		ActiveUsers: 850,
		NewUsers:    50,
		LastUpdated: time.Now(),
	}

	return stats, nil
}

// User 用户模型
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// UserStats 用户统计
type UserStats struct {
	TotalUsers  int       `json:"total_users"`
	ActiveUsers int       `json:"active_users"`
	NewUsers    int       `json:"new_users"`
	LastUpdated time.Time `json:"last_updated"`
}

// UserController 用户控制器
type UserController struct {
	userService *UserService
}

// NewUserController 创建用户控制器
func NewUserController() *UserController {
	return &UserController{
		userService: &UserService{},
	}
}

// GetUser 获取用户API
func (c *UserController) GetUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	user, err := c.userService.GetUserByID(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// CreateUser 创建用户API
func (c *UserController) CreateUser(ctx *gin.Context) {
	var user User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.userService.CreateUser(ctx, &user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// UpdateUser 更新用户API
func (c *UserController) UpdateUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	var updates map[string]interface{}
	if err := ctx.ShouldBindJSON(&updates); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.userService.UpdateUser(ctx, userID, updates)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser 删除用户API
func (c *UserController) DeleteUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	err := c.userService.DeleteUser(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// GetUserStats 获取用户统计API
func (c *UserController) GetUserStats(ctx *gin.Context) {
	stats, err := c.userService.GetUserStats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

// 辅助函数
func getLogLevel(updates map[string]interface{}) string {
	if len(updates) > 5 {
		return "warn"
	}
	return "info"
}

func isSensitiveOperation(userID string) bool {
	// 检查是否为敏感操作
	return userID == "admin" || userID == "root"
}

func main() {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin引擎
	r := gin.Default()

	// 创建控制器
	userController := NewUserController()

	// 注册路由
	r.GET("/api/users/:id", userController.GetUser)
	r.POST("/api/users", userController.CreateUser)
	r.PUT("/api/users/:id", userController.UpdateUser)
	r.DELETE("/api/users/:id", userController.DeleteUser)
	r.GET("/api/users/stats", userController.GetUserStats)

	// 添加请求ID中间件
	r.Use(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = "req-" + time.Now().Format("20060102150405")
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	})

	log.Println("增强日志注解示例服务器启动在 :8080")
	log.Println("测试端点:")
	log.Println("  GET  /api/users/:id      - 基本日志")
	log.Println("  POST /api/users          - 自定义字段日志")
	log.Println("  PUT  /api/users/:id      - 动态日志级别")
	log.Println("  DELETE /api/users/:id    - 条件日志")
	log.Println("  GET  /api/users/stats    - MDC日志")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
