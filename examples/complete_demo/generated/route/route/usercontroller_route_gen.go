package /Users/Zyi/GolandProjects/OpenSourceProject/gin-register-doc/gRain/examples/complete_demo/controllers

import (
	"github.com/gin-gonic/gin"
	
)

// RegisterRoutes 注册路由到给定的路由器
func (c *UserController) RegisterRoutes(router gin.IRouter) {
	
	// 没有路由前缀
	group := router.Group("")
	

	
	// GetUserList 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserList(ctx)
	})
	
	// GetUsers 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUsers(ctx)
	})
	
	// GetUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUser(ctx)
	})
	
	// CreateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateUser(ctx)
	})
	
	// UpdateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateUser(ctx)
	})
	
	// DeleteUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteUser(ctx)
	})
	
	// GetUserStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserStats(ctx)
	})
	
	// GetUserList 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserList(ctx)
	})
	
	// GetUsers 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUsers(ctx)
	})
	
	// GetUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUser(ctx)
	})
	
	// CreateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateUser(ctx)
	})
	
	// UpdateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateUser(ctx)
	})
	
	// DeleteUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteUser(ctx)
	})
	
	// GetUserStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserStats(ctx)
	})
	
}
