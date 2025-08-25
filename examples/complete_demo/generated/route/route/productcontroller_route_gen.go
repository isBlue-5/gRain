package /Users/Zyi/GolandProjects/OpenSourceProject/gin-register-doc/gRain/examples/complete_demo/controllers

import (
	"github.com/gin-gonic/gin"
	
)

// RegisterRoutes 注册路由到给定的路由器
func (c *ProductController) RegisterRoutes(router gin.IRouter) {
	
	// 没有路由前缀
	group := router.Group("")
	

	
	// GetProducts 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProducts(ctx)
	})
	
	// GetProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProduct(ctx)
	})
	
	// CreateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateProduct(ctx)
	})
	
	// UpdateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateProduct(ctx)
	})
	
	// DeleteProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteProduct(ctx)
	})
	
	// GetProductCategories 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductCategories(ctx)
	})
	
	// GetProductBrands 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductBrands(ctx)
	})
	
	// GetProductStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductStats(ctx)
	})
	
	// GetProducts 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProducts(ctx)
	})
	
	// GetProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProduct(ctx)
	})
	
	// CreateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.CreateProduct(ctx)
	})
	
	// UpdateProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.UpdateProduct(ctx)
	})
	
	// DeleteProduct 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.DeleteProduct(ctx)
	})
	
	// GetProductCategories 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductCategories(ctx)
	})
	
	// GetProductBrands 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductBrands(ctx)
	})
	
	// GetProductStats 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetProductStats(ctx)
	})
	
}
