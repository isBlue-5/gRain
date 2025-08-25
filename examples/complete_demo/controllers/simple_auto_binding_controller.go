package controllers

import (
	"github.com/gin-gonic/gin"
)

// SimpleAutoBindingController 简化版自动参数绑定控制器示例
// frame:controller(path="/api/v1/simple-auto-binding")
type SimpleAutoBindingController struct{}

// NewSimpleAutoBindingController 创建简化版自动参数绑定控制器
func NewSimpleAutoBindingController() *SimpleAutoBindingController {
	return &SimpleAutoBindingController{}
}

// 请求参数结构体定义

// UserCreateRequest 用户创建请求
type UserCreateRequest struct {
	Username string `json:"username" binding:"required" validate:"required,min=3,max=20"`
	Email    string `json:"email" binding:"required" validate:"required,email"`
	Password string `json:"password" binding:"required" validate:"required,min=6"`
	Role     string `json:"role" binding:"required" validate:"required,oneof=ADMIN USER"`
}

// UserUpdateRequest 用户更新请求
type UserUpdateRequest struct {
	Username string `json:"username" validate:"omitempty,min=3,max=20"`
	Email    string `json:"email" validate:"omitempty,email"`
	Role     string `json:"role" validate:"omitempty,oneof=ADMIN USER"`
	Status   string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UserQueryRequest 用户查询请求
type UserQueryRequest struct {
	Role     string `query:"role" validate:"omitempty,oneof=ADMIN USER"`
	Status   string `query:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Search   string `query:"search" validate:"omitempty,max=100"`
	Page     int    `query:"page" validate:"omitempty,min=1"`
	PageSize int    `query:"page_size" validate:"omitempty,min=1,max=100"`
}

// ProductCreateRequest 产品创建请求
type ProductCreateRequest struct {
	Name        string  `json:"name" binding:"required" validate:"required,min=1,max=100"`
	Description string  `json:"description" validate:"omitempty,max=500"`
	Price       float64 `json:"price" binding:"required" validate:"required,min=0"`
	Stock       int     `json:"stock" binding:"required" validate:"required,min=0"`
	Category    string  `json:"category" binding:"required" validate:"required"`
	Brand       string  `json:"brand" binding:"required" validate:"required"`
	SKU         string  `json:"sku" binding:"required" validate:"required"`
}

// ProductQueryRequest 产品查询请求
type ProductQueryRequest struct {
	Category string  `query:"category" validate:"omitempty"`
	Brand    string  `query:"brand" validate:"omitempty"`
	MinPrice float64 `query:"min_price" validate:"omitempty,min=0"`
	MaxPrice float64 `query:"max_price" validate:"omitempty,min=0"`
	InStock  bool    `query:"in_stock" validate:"omitempty"`
	Search   string  `query:"search" validate:"omitempty,max=100"`
	Page     int     `query:"page" validate:"omitempty,min=1"`
	PageSize int     `query:"page_size" validate:"omitempty,min=1,max=100"`
}

// 用户相关方法

// CreateUser 创建用户
// frame:route(method="POST", path="/users")
// frame:auth(roles={"ADMIN"})
// frame:validate(rules={"username": "required|min:3|max:20", "email": "required|email", "password": "required|min:6"})
// frame:response(201, gin.H, "用户创建成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(403, gin.H, "权限不足")
func (c *SimpleAutoBindingController) CreateUser(ctx *gin.Context, req UserCreateRequest) (gin.H, error) {
	// gRain框架自动绑定请求数据到req结构体
	// 自动验证参数
	// 自动权限检查

	// 模拟创建用户
	user := gin.H{
		"id":       123,
		"username": req.Username,
		"email":    req.Email,
		"role":     req.Role,
		"status":   "ACTIVE",
	}

	return gin.H{
		"success": true,
		"message": "用户创建成功",
		"data":    user,
	}, nil
}

// UpdateUser 更新用户
// frame:route(method="PUT", path="/users/:id")
// frame:auth(roles={"ADMIN", "USER"})
// frame:validate(rules={"id": "required|int"})
// frame:response(200, gin.H, "用户更新成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(404, gin.H, "用户不存在")
func (c *SimpleAutoBindingController) UpdateUser(ctx *gin.Context, id uint, req UserUpdateRequest) (gin.H, error) {
	// gRain框架自动绑定URI参数id和请求体到req结构体
	// 自动验证参数
	// 自动权限检查

	// 模拟更新用户
	user := gin.H{
		"id":       id,
		"username": req.Username,
		"email":    req.Email,
		"role":     req.Role,
		"status":   req.Status,
	}

	return gin.H{
		"success": true,
		"message": "用户更新成功",
		"data":    user,
	}, nil
}

// GetUsers 获取用户列表
// frame:route(method="GET", path="/users")
// frame:auth(roles={"ADMIN", "USER"})
// frame:response(200, gin.H, "成功获取用户列表")
// frame:response(500, gin.H, "服务器内部错误")
func (c *SimpleAutoBindingController) GetUsers(ctx *gin.Context, req UserQueryRequest) (gin.H, error) {
	// gRain框架自动绑定查询参数到req结构体
	// 自动验证参数
	// 自动权限检查

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 模拟用户列表
	users := []gin.H{
		{"id": 1, "username": "admin", "email": "admin@example.com", "role": "ADMIN", "status": "ACTIVE"},
		{"id": 2, "username": "user1", "email": "user1@example.com", "role": "USER", "status": "ACTIVE"},
		{"id": 3, "username": "user2", "email": "user2@example.com", "role": "USER", "status": "INACTIVE"},
	}

	// 应用查询条件
	filteredUsers := make([]gin.H, 0)
	for _, user := range users {
		if req.Role != "" && user["role"] != req.Role {
			continue
		}
		if req.Status != "" && user["status"] != req.Status {
			continue
		}
		if req.Search != "" {
			username := user["username"].(string)
			email := user["email"].(string)
			if !contains(username, req.Search) && !contains(email, req.Search) {
				continue
			}
		}
		filteredUsers = append(filteredUsers, user)
	}

	return gin.H{
		"success": true,
		"message": "成功获取用户列表",
		"data": gin.H{
			"users":     filteredUsers,
			"total":     len(filteredUsers),
			"page":      req.Page,
			"page_size": req.PageSize,
			"filters":   req,
		},
	}, nil
}

// GetUserByID 根据ID获取用户
// frame:route(method="GET", path="/users/:id")
// frame:auth(roles={"ADMIN", "USER"})
// frame:validate(rules={"id": "required|int"})
// frame:response(200, gin.H, "成功获取用户信息")
// frame:response(404, gin.H, "用户不存在")
func (c *SimpleAutoBindingController) GetUserByID(ctx *gin.Context, id uint) (gin.H, error) {
	// gRain框架自动绑定URI参数id
	// 自动验证参数
	// 自动权限检查

	// 模拟用户数据
	user := gin.H{
		"id":         id,
		"username":   "user" + string(rune(id)),
		"email":      "user" + string(rune(id)) + "@example.com",
		"role":       "USER",
		"status":     "ACTIVE",
		"created_at": "2024-01-01T00:00:00Z",
	}

	return gin.H{
		"success": true,
		"message": "成功获取用户信息",
		"data":    user,
	}, nil
}

// DeleteUser 删除用户
// frame:route(method="DELETE", path="/users/:id")
// frame:auth(roles={"ADMIN"})
// frame:validate(rules={"id": "required|int"})
// frame:response(200, gin.H, "用户删除成功")
// frame:response(403, gin.H, "权限不足")
// frame:response(404, gin.H, "用户不存在")
func (c *SimpleAutoBindingController) DeleteUser(ctx *gin.Context, id uint) (gin.H, error) {
	// gRain框架自动绑定URI参数id
	// 自动验证参数
	// 自动权限检查

	return gin.H{
		"success": true,
		"message": "用户删除成功",
		"data": gin.H{
			"deleted_id": id,
		},
	}, nil
}

// 产品相关方法

// CreateProduct 创建产品
// frame:route(method="POST", path="/products")
// frame:auth(roles={"ADMIN"})
// frame:validate(rules={"name": "required|min:1|max:100", "price": "required|min:0", "stock": "required|min:0"})
// frame:response(201, gin.H, "产品创建成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(403, gin.H, "权限不足")
func (c *SimpleAutoBindingController) CreateProduct(ctx *gin.Context, req ProductCreateRequest) (gin.H, error) {
	// gRain框架自动绑定请求数据到req结构体
	// 自动验证参数
	// 自动权限检查

	// 模拟创建产品
	product := gin.H{
		"id":          456,
		"name":        req.Name,
		"description": req.Description,
		"price":       req.Price,
		"stock":       req.Stock,
		"category":    req.Category,
		"brand":       req.Brand,
		"sku":         req.SKU,
		"status":      "ACTIVE",
	}

	return gin.H{
		"success": true,
		"message": "产品创建成功",
		"data":    product,
	}, nil
}

// UpdateProduct 更新产品
// frame:route(method="PUT", path="/products/:id")
// frame:auth(roles={"ADMIN"})
// frame:validate(rules={"id": "required|int"})
// frame:response(200, gin.H, "产品更新成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(404, gin.H, "产品不存在")
func (c *SimpleAutoBindingController) UpdateProduct(ctx *gin.Context, id uint, req ProductCreateRequest) (gin.H, error) {
	// gRain框架自动绑定URI参数id和请求体到req结构体
	// 自动验证参数
	// 自动权限检查

	// 模拟更新产品
	product := gin.H{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"price":       req.Price,
		"stock":       req.Stock,
		"category":    req.Category,
		"brand":       req.Brand,
		"sku":         req.SKU,
		"status":      "ACTIVE",
	}

	return gin.H{
		"success": true,
		"message": "产品更新成功",
		"data":    product,
	}, nil
}

// GetProducts 获取产品列表
// frame:route(method="GET", path="/products")
// frame:response(200, gin.H, "成功获取产品列表")
// frame:response(500, gin.H, "服务器内部错误")
func (c *SimpleAutoBindingController) GetProducts(ctx *gin.Context, req ProductQueryRequest) (gin.H, error) {
	// gRain框架自动绑定查询参数到req结构体
	// 自动验证参数

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 模拟产品列表
	products := []gin.H{
		{"id": 1, "name": "iPhone 15", "price": 999.99, "stock": 50, "category": "Electronics", "brand": "Apple", "status": "ACTIVE"},
		{"id": 2, "name": "MacBook Pro", "price": 1999.99, "stock": 25, "category": "Electronics", "brand": "Apple", "status": "ACTIVE"},
		{"id": 3, "name": "AirPods Pro", "price": 249.99, "stock": 100, "category": "Electronics", "brand": "Apple", "status": "ACTIVE"},
	}

	// 应用查询条件
	filteredProducts := make([]gin.H, 0)
	for _, product := range products {
		if req.Category != "" && product["category"] != req.Category {
			continue
		}
		if req.Brand != "" && product["brand"] != req.Brand {
			continue
		}
		if req.MinPrice > 0 && product["price"].(float64) < req.MinPrice {
			continue
		}
		if req.MaxPrice > 0 && product["price"].(float64) > req.MaxPrice {
			continue
		}
		if req.InStock && product["stock"].(int) <= 0 {
			continue
		}
		if req.Search != "" {
			name := product["name"].(string)
			if !contains(name, req.Search) {
				continue
			}
		}
		filteredProducts = append(filteredProducts, product)
	}

	return gin.H{
		"success": true,
		"message": "成功获取产品列表",
		"data": gin.H{
			"products":  filteredProducts,
			"total":     len(filteredProducts),
			"page":      req.Page,
			"page_size": req.PageSize,
			"filters":   req,
		},
	}, nil
}

// GetProductByID 根据ID获取产品
// frame:route(method="GET", path="/products/:id")
// frame:response(200, gin.H, "成功获取产品信息")
// frame:response(404, gin.H, "产品不存在")
func (c *SimpleAutoBindingController) GetProductByID(ctx *gin.Context, id uint) (gin.H, error) {
	// gRain框架自动绑定URI参数id
	// 自动验证参数

	// 模拟产品数据
	product := gin.H{
		"id":          id,
		"name":        "Product " + string(rune(id)),
		"description": "This is product " + string(rune(id)),
		"price":       99.99,
		"stock":       100,
		"category":    "General",
		"brand":       "Brand",
		"sku":         "SKU" + string(rune(id)),
		"status":      "ACTIVE",
	}

	return gin.H{
		"success": true,
		"message": "成功获取产品信息",
		"data":    product,
	}, nil
}

// DeleteProduct 删除产品
// frame:route(method="DELETE", path="/products/:id")
// frame:auth(roles={"ADMIN"})
// frame:validate(rules={"id": "required|int"})
// frame:response(200, gin.H, "产品删除成功")
// frame:response(403, gin.H, "权限不足")
// frame:response(404, gin.H, "产品不存在")
func (c *SimpleAutoBindingController) DeleteProduct(ctx *gin.Context, id uint) (gin.H, error) {
	// gRain框架自动绑定URI参数id
	// 自动验证参数
	// 自动权限检查

	return gin.H{
		"success": true,
		"message": "产品删除成功",
		"data": gin.H{
			"deleted_id": id,
		},
	}, nil
}

// 统计相关方法

// GetUserStats 获取用户统计信息
// frame:route(method="GET", path="/stats/users")
// frame:auth(roles={"ADMIN"})
// frame:response(200, gin.H, "成功获取用户统计信息")
// frame:response(403, gin.H, "权限不足")
func (c *SimpleAutoBindingController) GetUserStats(ctx *gin.Context) (gin.H, error) {
	// gRain框架自动权限检查

	return gin.H{
		"success": true,
		"message": "成功获取用户统计信息",
		"data": gin.H{
			"total_users":    100,
			"active_users":   85,
			"inactive_users": 15,
			"admin_users":    5,
			"regular_users":  95,
		},
	}, nil
}

// GetProductStats 获取产品统计信息
// frame:route(method="GET", path="/stats/products")
// frame:auth(roles={"ADMIN"})
// frame:response(200, gin.H, "成功获取产品统计信息")
// frame:response(403, gin.H, "权限不足")
func (c *SimpleAutoBindingController) GetProductStats(ctx *gin.Context) (gin.H, error) {
	// gRain框架自动权限检查

	return gin.H{
		"success": true,
		"message": "成功获取产品统计信息",
		"data": gin.H{
			"total_products":        500,
			"active_products":       450,
			"low_stock_products":    25,
			"out_of_stock_products": 5,
			"total_value":           125000.00,
		},
	}, nil
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
