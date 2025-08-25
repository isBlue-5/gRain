package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/core/app"
	"github.com/isBlue-5/grain/pkg/web/middleware"
	"github.com/isBlue-5/grain/pkg/web/render"
)

func main() {
	// 创建应用程序
	application := app.New(
		app.WithName("WebDemo"),
		app.WithDescription("Web演示应用"),
	)

	// 创建路由器
	router := gin.Default()
	// 通过注册中心链式组合自定义和内置中间件
	router.Use(middleware.DefaultRegistry.Chain("timing")...)

	// 注册API路由
	apiGroup := router.Group("/api")

	// 注册用户控制器
	userController := NewUserController()
	userController.RegisterRoutes(apiGroup.Group("/users"))

	// 注册产品控制器
	productController := NewProductController()
	productController.RegisterRoutes(apiGroup.Group("/products"))

	// 启动HTTP服务
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 启动HTTP服务
	go func() {
		log.Println("HTTP服务启动在 :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP服务启动失败: %v\n", err)
		}
	}()

	// 启动应用（这将处理信号和优雅关闭）
	if err := application.Run(); err != nil {
		log.Fatalf("应用运行失败: %v", err)
	}
}

// User 用户结构
type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"gte=0,lt=120"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"gte=0,lt=120"`
}

// UserController 用户控制器
// frame:controller(path="/users")
type UserController struct {
	users  []User
	nextID uint
}

// NewUserController 创建用户控制器
func NewUserController() *UserController {
	return &UserController{
		users: []User{
			{ID: 1, Username: "admin", Email: "admin@example.com", Age: 30},
			{ID: 2, Username: "user", Email: "user@example.com", Age: 25},
		},
		nextID: 3,
	}
}

// RegisterRoutes 注册路由
func (c *UserController) RegisterRoutes(router gin.IRouter) {
	router.GET("", c.GetUsers)
	router.GET("/:id", c.GetUser)
	router.POST("", c.CreateUser)
	router.PUT("/:id", c.UpdateUser)
	router.DELETE("/:id", c.DeleteUser)
}

// GetUsers 获取所有用户
// frame:route(method="GET", path="")
// frame:log(level="info", message="获取所有用户")
func (c *UserController) GetUsers(ctx *gin.Context) {
	render.Success(ctx, c.users)
}

// GetUser 获取指定用户
// frame:route(method="GET", path="/:id")
// frame:log(level="info", includeParams=true)
func (c *UserController) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")
	for _, user := range c.users {
		if fmt.Sprint(user.ID) == id {
			render.Success(ctx, user)
			return
		}
	}
	render.Error(ctx, http.StatusNotFound, fmt.Errorf("用户不存在: %s", id))
}

// CreateUser 创建用户
// frame:route(method="POST", path="")
// frame:bind(source="json", model="CreateUserRequest")
// frame:validate
// frame:log(level="info", includeParams=true, includeResult=true)
func (c *UserController) CreateUser(ctx *gin.Context) {
	var req CreateUserRequest

	// 绑定和验证请求（由注解生成的代码处理）
	if err := ctx.ShouldBindJSON(&req); err != nil {
		render.Error(ctx, http.StatusBadRequest, err)
		return
	}

	// 创建新用户
	user := User{
		ID:       c.nextID,
		Username: req.Username,
		Email:    req.Email,
		Age:      req.Age,
	}

	// 添加到列表
	c.users = append(c.users, user)
	c.nextID++

	render.Success(ctx, user)
}

// UpdateUser 更新用户
// frame:route(method="PUT", path="/:id")
// frame:bind(source="json", model="User")
// frame:validate
// frame:log(level="info", includeParams=true)
func (c *UserController) UpdateUser(ctx *gin.Context) {
	id := ctx.Param("id")
	var req User

	// 绑定和验证请求
	if err := ctx.ShouldBindJSON(&req); err != nil {
		render.Error(ctx, http.StatusBadRequest, err)
		return
	}

	// 查找用户
	for i, user := range c.users {
		if fmt.Sprint(user.ID) == id {
			// 更新用户信息
			c.users[i].Username = req.Username
			c.users[i].Email = req.Email
			c.users[i].Age = req.Age

			render.Success(ctx, c.users[i])
			return
		}
	}

	render.Error(ctx, http.StatusNotFound, fmt.Errorf("用户不存在: %s", id))
}

// DeleteUser 删除用户
// frame:route(method="DELETE", path="/:id")
// frame:log(level="info")
func (c *UserController) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")

	// 查找用户
	for i, user := range c.users {
		if fmt.Sprint(user.ID) == id {
			// 删除用户
			c.users = append(c.users[:i], c.users[i+1:]...)
			render.SuccessWithMessage(ctx, "删除成功", nil)
			return
		}
	}

	render.Error(ctx, http.StatusNotFound, fmt.Errorf("用户不存在: %s", id))
}

// Product 产品结构
type Product struct {
	ID    uint    `json:"id"`
	Name  string  `json:"name" validate:"required"`
	Price float64 `json:"price" validate:"required,gt=0"`
}

// ProductController 产品控制器
// frame:controller(path="/products")
type ProductController struct {
	products []Product
	nextID   uint
}

// NewProductController 创建产品控制器
func NewProductController() *ProductController {
	return &ProductController{
		products: []Product{
			{ID: 1, Name: "产品1", Price: 99.99},
			{ID: 2, Name: "产品2", Price: 199.99},
		},
		nextID: 3,
	}
}

// RegisterRoutes 注册路由
func (c *ProductController) RegisterRoutes(router gin.IRouter) {
	router.GET("", c.GetProducts)
	router.GET("/:id", c.GetProduct)
}

// GetProducts 获取所有产品
// frame:route(method="GET", path="")
// frame:log(level="info")
func (c *ProductController) GetProducts(ctx *gin.Context) {
	render.Success(ctx, c.products)
}

// GetProduct 获取指定产品
// frame:route(method="GET", path="/:id")
// frame:log(level="info")
func (c *ProductController) GetProduct(ctx *gin.Context) {
	id := ctx.Param("id")
	for _, product := range c.products {
		if fmt.Sprint(product.ID) == id {
			render.Success(ctx, product)
			return
		}
	}
	render.Error(ctx, http.StatusNotFound, fmt.Errorf("产品不存在: %s", id))
}
