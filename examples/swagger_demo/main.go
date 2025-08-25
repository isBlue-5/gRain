package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/annotation/processor"
	"github.com/grain-framework/grain/pkg/annotation/registry"
)

// User 用户模型
type User struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" binding:"required" example:"johndoe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Role     string `json:"role" binding:"oneof=USER ADMIN" example:"USER"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
	Role     string `json:"role" binding:"oneof=USER ADMIN" example:"USER"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username string `json:"username" binding:"min=3,max=50" example:"johndoe"`
	Email    string `json:"email" binding:"email" example:"john@example.com"`
	Role     string `json:"role" binding:"oneof=USER ADMIN" example:"USER"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" example:"johndoe"`
	Email    string `json:"email" example:"john@example.com"`
	Role     string `json:"role" example:"USER"`
}

// UserController 用户控制器
// frame:controller(path="/api/users")
type UserController struct{}

// GetUsers 获取用户列表
// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:response(200, []UserResponse, "成功获取用户列表")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) GetUsers(ctx *gin.Context) {
	users := []UserResponse{
		{ID: 1, Username: "johndoe", Email: "john@example.com", Role: "USER"},
		{ID: 2, Username: "janedoe", Email: "jane@example.com", Role: "ADMIN"},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}

// GetUser 获取单个用户
// frame:route(method="GET", path="/:id")
// frame:summary(获取单个用户)
// frame:response(200, UserResponse, "成功获取用户信息")
// frame:response(404, gin.H, "用户不存在")
func (c *UserController) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")

	// 模拟用户数据
	user := UserResponse{
		ID:       1,
		Username: "johndoe",
		Email:    "john@example.com",
		Role:     "USER",
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// CreateUser 创建用户
// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:bind(source="json", model="CreateUserRequest")
// frame:response(201, UserResponse, "用户创建成功")
// frame:response(400, gin.H, "请求参数错误")
func (c *UserController) CreateUser(ctx *gin.Context) {
	var req CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 模拟创建用户
	user := UserResponse{
		ID:       3,
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
	})
}

// UpdateUser 更新用户
// frame:route(method="PUT", path="/:id")
// frame:summary(更新用户信息)
// frame:bind(source="json", model="UpdateUserRequest")
// frame:response(200, UserResponse, "用户更新成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(404, gin.H, "用户不存在")
func (c *UserController) UpdateUser(ctx *gin.Context) {
	id := ctx.Param("id")
	var req UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 模拟更新用户
	user := UserResponse{
		ID:       1,
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// DeleteUser 删除用户
// frame:route(method="DELETE", path="/:id")
// frame:summary(删除用户)
// frame:response(200, gin.H, "用户删除成功")
// frame:response(404, gin.H, "用户不存在")
func (c *UserController) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户删除成功",
		"id":      id,
	})
}

// Product 产品模型
type Product struct {
	ID          uint    `json:"id" example:"1"`
	Name        string  `json:"name" binding:"required" example:"iPhone 15"`
	Description string  `json:"description" example:"最新款iPhone"`
	Price       float64 `json:"price" binding:"required,gt=0" example:"999.99"`
	Category    string  `json:"category" binding:"required" example:"Electronics"`
	Stock       int     `json:"stock" binding:"gte=0" example:"100"`
}

// CreateProductRequest 创建产品请求
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required,min=2,max=100" example:"iPhone 15"`
	Description string  `json:"description" example:"最新款iPhone"`
	Price       float64 `json:"price" binding:"required,gt=0" example:"999.99"`
	Category    string  `json:"category" binding:"required" example:"Electronics"`
	Stock       int     `json:"stock" binding:"gte=0" example:"100"`
}

// ProductController 产品控制器
// frame:controller(path="/api/products")
type ProductController struct{}

// GetProducts 获取产品列表
// frame:route(method="GET", path="/")
// frame:summary(获取产品列表)
// frame:response(200, []Product, "成功获取产品列表")
func (c *ProductController) GetProducts(ctx *gin.Context) {
	products := []Product{
		{ID: 1, Name: "iPhone 15", Description: "最新款iPhone", Price: 999.99, Category: "Electronics", Stock: 100},
		{ID: 2, Name: "MacBook Pro", Description: "专业级笔记本", Price: 1999.99, Category: "Electronics", Stock: 50},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
	})
}

// GetProduct 获取单个产品
// frame:route(method="GET", path="/:id")
// frame:summary(获取单个产品)
// frame:response(200, Product, "成功获取产品信息")
// frame:response(404, gin.H, "产品不存在")
func (c *ProductController) GetProduct(ctx *gin.Context) {
	id := ctx.Param("id")

	product := Product{
		ID:          1,
		Name:        "iPhone 15",
		Description: "最新款iPhone",
		Price:       999.99,
		Category:    "Electronics",
		Stock:       100,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    product,
	})
}

// CreateProduct 创建产品
// frame:route(method="POST", path="/")
// frame:summary(创建新产品)
// frame:bind(source="json", model="CreateProductRequest")
// frame:response(201, Product, "产品创建成功")
// frame:response(400, gin.H, "请求参数错误")
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	var req CreateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	product := Product{
		ID:          3,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Stock:       req.Stock,
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    product,
	})
}

func main() {
	// 创建Gin引擎
	r := gin.Default()

	// 创建注解注册表
	registry := registry.NewRegistry()

	// 创建Swagger生成器
	swaggerGenerator := processor.NewSwaggerGenerator(registry, ".")

	// 创建Swagger服务器
	swaggerServer := processor.NewSwaggerServer(swaggerGenerator)

	// 注册Swagger路由
	swaggerServer.RegisterRoutes(r)

	// 注册API路由
	userController := &UserController{}
	productController := &ProductController{}

	// 用户API
	users := r.Group("/api/users")
	{
		users.GET("/", userController.GetUsers)
		users.GET("/:id", userController.GetUser)
		users.POST("/", userController.CreateUser)
		users.PUT("/:id", userController.UpdateUser)
		users.DELETE("/:id", userController.DeleteUser)
	}

	// 产品API
	products := r.Group("/api/products")
	{
		products.GET("/", productController.GetProducts)
		products.GET("/:id", productController.GetProduct)
		products.POST("/", productController.CreateProduct)
	}

	// 启动服务器
	log.Println("Server starting on :8080")
	log.Println("Swagger UI available at: http://localhost:8080/swagger")
	log.Println("Swagger JSON available at: http://localhost:8080/swagger/swagger.json")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
