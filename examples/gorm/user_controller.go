// Package gorm 演示GORM适配器的使用示例
package gorm

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
// frame:controller(prefix="/api/users")
type UserController struct {
	// 注入用户服务
	service UserService `inject:""`
}

// GetUserByID 根据ID获取用户
// frame:route(method="GET", path="/:id")
func (c *UserController) GetUserByID(ctx *gin.Context) {
	// 解析用户ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 调用服务获取用户
	user, err := c.service.GetUser(ctx, uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回用户信息
	ctx.JSON(http.StatusOK, user)
}

// GetAllUsers 获取所有用户
// frame:route(method="GET", path="/")
func (c *UserController) GetAllUsers(ctx *gin.Context) {
	// 调用服务获取所有用户
	users, err := c.service.GetAllUsers(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回用户列表
	ctx.JSON(http.StatusOK, users)
}

// CreateUser 创建用户
// frame:route(method="POST", path="/")
// frame:bind(source="json", model="User")
func (c *UserController) CreateUser(ctx *gin.Context) {
	// 绑定请求体到用户对象
	var user User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务创建用户
	if err := c.service.CreateUser(ctx, &user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回成功信息
	ctx.JSON(http.StatusCreated, gin.H{"message": "用户创建成功"})
}

// UpdateUser 更新用户
// frame:route(method="PUT", path="/:id")
// frame:bind(source="json", model="User")
func (c *UserController) UpdateUser(ctx *gin.Context) {
	// 解析用户ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 绑定请求体到用户对象
	var user User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置用户ID
	user.ID = uint(id)

	// 调用服务更新用户
	if err := c.service.UpdateUser(ctx, &user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回成功信息
	ctx.JSON(http.StatusOK, gin.H{"message": "用户更新成功"})
}

// DeleteUser 删除用户
// frame:route(method="DELETE", path="/:id")
func (c *UserController) DeleteUser(ctx *gin.Context) {
	// 解析用户ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 调用服务删除用户
	if err := c.service.DeleteUser(ctx, uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回成功信息
	ctx.JSON(http.StatusOK, gin.H{"message": "用户删除成功"})
}

// NewUserController 创建用户控制器
func NewUserController(service UserService) *UserController {
	return &UserController{service: service}
}
