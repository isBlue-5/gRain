package /Users/Zyi/GolandProjects/OpenSourceProject/gin-register-doc/gRain/examples/complete_demo/controllers

import (
	"github.com/gin-gonic/gin"
	
)

// RegisterRoutes 注册路由到给定的路由器
func (c *SimpleAutoBindingController) RegisterRoutes(router gin.IRouter) {
	
	// 没有路由前缀
	group := router.Group("")
	

	
	// CreateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateUser(ctx)
	})
	
	// UpdateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateUser(ctx)
	})
	
	// GetUsers 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUsers(ctx)
	})
	
	// GetUserByID 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserByID(ctx)
	})
	
	// DeleteUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteUser(ctx)
	})
	
	// CreateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateProduct(ctx)
	})
	
	// UpdateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateProduct(ctx)
	})
	
	// GetProducts 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProducts(ctx)
	})
	
	// GetProductByID 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductByID(ctx)
	})
	
	// DeleteProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteProduct(ctx)
	})
	
	// GetUserStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserStats(ctx)
	})
	
	// GetProductStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductStats(ctx)
	})
	
	// CreateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateUser(ctx)
	})
	
	// UpdateUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateUser(ctx)
	})
	
	// GetUsers 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUsers(ctx)
	})
	
	// GetUserByID 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserByID(ctx)
	})
	
	// DeleteUser 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteUser(ctx)
	})
	
	// CreateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateProduct(ctx)
	})
	
	// UpdateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateProduct(ctx)
	})
	
	// GetProducts 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProducts(ctx)
	})
	
	// GetProductByID 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductByID(ctx)
	})
	
	// DeleteProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteProduct(ctx)
	})
	
	// GetUserStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserStats(ctx)
	})
	
	// GetProductStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductStats(ctx)
	})
	
}
