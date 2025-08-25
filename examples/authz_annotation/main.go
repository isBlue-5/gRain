package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
// frame:controller
type UserController struct {
	userService *UserService `inject:""`
}

// UserService 用户服务
type UserService struct{}

// GetUserProfile 获取用户资料（需要用户角色）
// frame:route(method="GET", path="/api/user/profile")
// frame:auth(roles={"user", "admin"})
func (c *UserController) GetUserProfile(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	ctx.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": "testuser",
		"email":    "test@example.com",
		"role":     "user",
	})
}

// UpdateUserProfile 更新用户资料（需要用户权限）
// frame:route(method="PUT", path="/api/user/profile")
// frame:auth(permissions={"user:write"})
func (c *UserController) UpdateUserProfile(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user_id": userID,
	})
}

// GetUserList 获取用户列表（需要管理员角色）
// frame:route(method="GET", path="/api/admin/users")
// frame:auth(roles={"admin"})
func (c *UserController) GetUserList(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"users": []gin.H{
			{"id": "1", "username": "user1", "role": "user"},
			{"id": "2", "username": "user2", "role": "admin"},
		},
	})
}

// DeleteUser 删除用户（需要管理员权限）
// frame:route(method="DELETE", path="/api/admin/users/:id")
// frame:auth(permissions={"admin:delete"})
func (c *UserController) DeleteUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	ctx.JSON(http.StatusOK, gin.H{
		"message":         "User deleted successfully",
		"deleted_user_id": userID,
	})
}

// GetSensitiveData 获取敏感数据（需要表达式权限）
// frame:route(method="GET", path="/api/data/sensitive")
// frame:auth(expression="user.role == 'admin' && user.department == 'IT'")
func (c *UserController) GetSensitiveData(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"sensitive_data": "This is sensitive information",
		"access_level":   "high",
	})
}

// AdminDashboard 管理员仪表板（需要管理员角色和权限）
// frame:route(method="GET", path="/api/admin/dashboard")
// frame:auth(roles={"admin"}, permissions={"admin:read"})
func (c *UserController) AdminDashboard(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"dashboard": gin.H{
			"total_users":   100,
			"active_users":  85,
			"system_status": "healthy",
		},
	})
}

func main() {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin引擎
	r := gin.Default()

	// 创建控制器实例
	userController := &UserController{
		userService: &UserService{},
	}

	// 注册路由
	r.GET("/api/user/profile", userController.GetUserProfile)
	r.PUT("/api/user/profile", userController.UpdateUserProfile)
	r.GET("/api/admin/users", userController.GetUserList)
	r.DELETE("/api/admin/users/:id", userController.DeleteUser)
	r.GET("/api/data/sensitive", userController.GetSensitiveData)
	r.GET("/api/admin/dashboard", userController.AdminDashboard)

	// 添加用户ID中间件（模拟）
	r.Use(func(c *gin.Context) {
		// 模拟从认证中获取用户ID
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "user123" // 默认用户
		}

		// 支持通过请求参数切换用户身份进行测试
		if testUser := c.Query("test_user"); testUser != "" {
			userID = testUser
		}

		c.Set("user_id", userID)
		c.Next()
	})

	log.Println("权限控制注解示例服务器启动在 :8080")
	log.Println("测试端点:")
	log.Println("  GET  /api/user/profile     - 需要用户或管理员角色")
	log.Println("  PUT  /api/user/profile     - 需要用户写入权限")
	log.Println("  GET  /api/admin/users      - 需要管理员角色")
	log.Println("  DELETE /api/admin/users/:id - 需要管理员删除权限")
	log.Println("  GET  /api/data/sensitive   - 需要表达式权限")
	log.Println("  GET  /api/admin/dashboard  - 需要管理员角色和权限")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
