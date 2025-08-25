package /Users/Zyi/GolandProjects/OpenSourceProject/gin-register-doc/gRain/examples/complete_demo/controllers

import (
	"github.com/gin-gonic/gin"
	
)

// RegisterRoutes 注册路由到给定的路由器
func (c *DemoController) RegisterRoutes(router gin.IRouter) {
	
	// 没有路由前缀
	group := router.Group("")
	

	
	// GetUserById 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserById(ctx)
	})
	
	// GetUserById 路由
	group.GET("", func(ctx *gin.Context) {
		
		
		
		
		c.GetUserById(ctx)
	})
	
}
