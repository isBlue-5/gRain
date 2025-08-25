// 本文件展示了gRain框架的注解系统和代码生成用法
//go:generate ginframe-gen -output ./generated -verbose ./...

package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/examples/annotation/controller"
	"github.com/grain-framework/grain/examples/annotation/service"
)

func main() {
	log.Println("启动示例应用程序...")

	// 创建Gin引擎
	router := gin.Default()

	// 1. 在实际应用中，这些对象会由生成的代码自动创建和注入
	// 这里手动创建它们用于示例
	repo := service.NewInMemoryUserRepository()
	userService := &service.UserServiceImpl{Repository: repo}
	userController := &controller.UserController{UserService: userService}

	// 2. 在实际应用中，这里会使用生成的路由注册代码
	// 我们这里模拟生成的代码的行为
	api := router.Group("/api")
	users := api.Group("/users")

	// 注册路由
	users.GET("/:id", userController.GetUser)
	users.POST("/", userController.CreateUser)
	users.GET("/", userController.ListUsers)

	log.Println("路由已注册:")
	log.Println("- GET  /api/users/:id")
	log.Println("- POST /api/users")
	log.Println("- GET  /api/users")

	// 提示用户如何测试API
	log.Println("\n您可以使用以下命令测试API:")
	log.Println("curl -X GET http://localhost:8080/api/users/1")
	log.Println("curl -X GET http://localhost:8080/api/users")
	log.Println("curl -X POST http://localhost:8080/api/users -H \"Content-Type: application/json\" -d '{\"name\":\"赵六\",\"age\":35}'")

	// 3. 启动服务器
	log.Println("\n服务启动在 http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}

/* 注解系统使用示例说明

1. 注解类型
   - controller: 标记控制器类型
   - route: 标记HTTP路由
   - service: 标记服务类型
   - repository: 标记存储库类型
   - inject: 标记依赖注入
   - log: 标记日志记录
   - transaction: 标记事务管理
   - validation: 标记参数验证
   - cache: 标记方法缓存

2. 代码生成内容
   - 生成依赖注入代码: ./generated/wire_gen.go
   - 生成控制器路由注册代码: ./generated/user_controller_route_gen.go

3. 使用步骤
   a. 添加注解到类型和方法
   b. 运行 `go generate ./...`
   c. 使用生成的代码初始化应用
*/
