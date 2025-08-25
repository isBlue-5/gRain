package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SearchController 搜索控制器
// frame:controller
type SearchController struct {
	searchService *SearchService `inject:""`
}

// SearchService 搜索服务
type SearchService struct{}

// SearchAPI 搜索API（IP限流）
// frame:route(method="GET", path="/api/search")
// frame:rateLimit(limit=10, period="1m", key="ip")
func (c *SearchController) SearchAPI(ctx *gin.Context) {
	// 模拟搜索逻辑
	time.Sleep(100 * time.Millisecond)

	ctx.JSON(http.StatusOK, gin.H{
		"results": []string{"result1", "result2", "result3"},
		"query":   ctx.Query("q"),
	})
}

// UserSearchAPI 用户搜索API（用户限流）
// frame:route(method="GET", path="/api/user/search")
// frame:rateLimit(limit=5, period="1m", key="user")
func (c *SearchController) UserSearchAPI(ctx *gin.Context) {
	// 模拟用户搜索逻辑
	time.Sleep(200 * time.Millisecond)

	ctx.JSON(http.StatusOK, gin.H{
		"results": []string{"user_result1", "user_result2"},
		"user_id": ctx.GetString("user_id"),
		"query":   ctx.Query("q"),
	})
}

// AdminAPI 管理员API（路径限流）
// frame:route(method="GET", path="/api/admin/data")
// frame:rateLimit(limit=2, period="1m", key="path")
func (c *SearchController) AdminAPI(ctx *gin.Context) {
	// 模拟管理员数据获取
	time.Sleep(500 * time.Millisecond)

	ctx.JSON(http.StatusOK, gin.H{
		"admin_data": "sensitive_data",
		"timestamp":  time.Now().Unix(),
	})
}

// CustomKeyAPI 自定义键限流API
// frame:route(method="POST", path="/api/custom")
// frame:rateLimit(limit=3, period="30s", key="header:X-Custom-Key")
func (c *SearchController) CustomKeyAPI(ctx *gin.Context) {
	// 模拟自定义键限流逻辑
	time.Sleep(150 * time.Millisecond)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Custom key rate limited",
		"key":     ctx.GetHeader("X-Custom-Key"),
	})
}

func main() {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin引擎
	r := gin.Default()

	// 创建控制器实例
	searchController := &SearchController{
		searchService: &SearchService{},
	}

	// 注册路由
	r.GET("/api/search", searchController.SearchAPI)
	r.GET("/api/user/search", searchController.UserSearchAPI)
	r.GET("/api/admin/data", searchController.AdminAPI)
	r.POST("/api/custom", searchController.CustomKeyAPI)

	// 添加用户ID中间件（模拟）
	r.Use(func(c *gin.Context) {
		// 模拟从认证中获取用户ID
		c.Set("user_id", "user123")
		c.Next()
	})

	log.Println("限流注解示例服务器启动在 :8080")
	log.Println("测试端点:")
	log.Println("  GET  /api/search      - IP限流 (10次/分钟)")
	log.Println("  GET  /api/user/search - 用户限流 (5次/分钟)")
	log.Println("  GET  /api/admin/data  - 路径限流 (2次/分钟)")
	log.Println("  POST /api/custom      - 自定义键限流 (3次/30秒)")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
