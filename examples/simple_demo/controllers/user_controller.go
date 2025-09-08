package controllers

import (
	"github.com/gin-gonic/gin"
)

// frame:controller(path="/api/users", tags={"用户管理"})
type UserController struct {
	// 框架会自动注入这些服务
	// frame:inject
	UserService interface{} `inject:""`
	AuthService interface{} `inject:""`
}

// GetUsers 获取用户列表
// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:description(分页获取用户列表，支持搜索和过滤)
// frame:tags({"用户管理"})
// frame:response(200, []UserResponse, "成功获取用户列表")
// frame:response(400, ErrorResponse, "请求参数错误")
func (c *UserController) GetUsers(ctx *gin.Context) {
	// 业务逻辑在这里
	// 框架会自动处理路由注册、依赖注入、参数验证等
	ctx.JSON(200, gin.H{
		"message": "用户列表获取成功",
		"data":    []string{"用户1", "用户2", "用户3"},
	})
}

// GetUser 获取单个用户
// frame:route(method="GET", path="/:id")
// frame:summary(获取用户信息)
// frame:description(根据用户ID获取用户详细信息)
// frame:tags({"用户管理"})
// frame:response(200, UserResponse, "成功获取用户信息")
// frame:response(404, ErrorResponse, "用户不存在")
func (c *UserController) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{
		"message": "用户信息获取成功",
		"data": gin.H{
			"id":   id,
			"name": "示例用户",
		},
	})
}

// CreateUser 创建用户
// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:description(创建一个新的用户账户")
// frame:tags({"用户管理"})
// frame:bind(source="json", model="CreateUserRequest")
// frame:response(201, UserResponse, "用户创建成功")
// frame:response(400, ValidationError, "请求参数验证失败")
// frame:response(409, ConflictError, "用户已存在")
func (c *UserController) CreateUser(ctx *gin.Context) {
	// 框架会自动绑定和验证请求参数
	ctx.JSON(201, gin.H{
		"message": "用户创建成功",
		"data": gin.H{
			"id":   "new-user-id",
			"name": "新用户",
		},
	})
}

// UpdateUser 更新用户
// frame:route(method="PUT", path="/:id")
// frame:summary(更新用户信息)
// frame:description(更新指定用户的信息")
// frame:tags({"用户管理"})
// frame:bind(source="json", model="UpdateUserRequest")
// frame:response(200, UserResponse, "用户信息更新成功")
// frame:response(400, ValidationError, "请求参数验证失败")
// frame:response(404, ErrorResponse, "用户不存在")
func (c *UserController) UpdateUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{
		"message": "用户信息更新成功",
		"data": gin.H{
			"id":   id,
			"name": "更新后的用户",
		},
	})
}

// DeleteUser 删除用户
// frame:route(method="DELETE", path="/:id")
// frame:summary(删除用户)
// frame:description(删除指定的用户账户")
// frame:tags({"用户管理"})
// frame:response(200, SuccessResponse, "用户删除成功")
// frame:response(404, ErrorResponse, "用户不存在")
func (c *UserController) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{
		"message": "用户删除成功",
		"data": gin.H{
			"id": id,
		},
	})
}
