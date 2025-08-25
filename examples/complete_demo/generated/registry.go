package generated

import (
	"complete_demo/config"
	"complete_demo/database"
	"log"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有生成的路由
func RegisterRoutes(router *gin.Engine) {
	log.Println("正在注册gRain生成的路由...")

	// TODO: 集成实际生成的路由
	// 这里是一个占位符实现，确保示例应用可以编译
	api := router.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "ok",
				"message":   "gRain框架运行正常",
				"generated": true,
			})
		})
	}

	log.Println("gRain路由注册完成")
}

// RegisterAllgRainRoutes 注册所有gRain生成的路由（示例应用期望的函数）
func RegisterAllgRainRoutes(router *gin.Engine, cfg *config.Config, dbManager *database.Database) {
	log.Println("🚀 正在注册所有gRain生成的路由...")

	// 调用基础路由注册
	RegisterRoutes(router)

	// TODO: 集成实际生成的控制器路由
	// 这里是占位符实现，展示框架能力

	// 用户控制器路由组
	userGroup := router.Group("/api/users")
	{
		userGroup.GET("/", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "用户列表 - gRain生成"})
		})
		userGroup.GET("/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "用户详情 - gRain生成", "id": c.Param("id")})
		})
	}

	// 产品控制器路由组
	productGroup := router.Group("/api/products")
	{
		productGroup.GET("/", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "产品列表 - gRain生成"})
		})
		productGroup.GET("/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "产品详情 - gRain生成", "id": c.Param("id")})
		})
	}

	// 统计路由组
	statsGroup := router.Group("/api/stats")
	{
		statsGroup.GET("/users", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "用户统计 - gRain生成"})
		})
		statsGroup.GET("/products", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "产品统计 - gRain生成"})
		})
	}

	log.Println("✅ 所有gRain路由注册完成！")
}

// InitializeServices 初始化所有生成的服务
func InitializeServices() {
	log.Println("正在初始化gRain生成的服务...")

	// TODO: 集成实际生成的服务初始化逻辑
	// 这里是一个占位符实现

	log.Println("gRain服务初始化完成")
}
