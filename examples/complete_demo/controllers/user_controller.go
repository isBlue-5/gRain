package controllers

import (
	"net/http"
	"strconv"

	"complete_demo/models"
	"complete_demo/services"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
// frame:controller(path="/api/users")
type UserController struct {
	userService *services.UserService `inject:""`
}

// NewUserController 创建用户控制器
func NewUserController(userService *services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// GetUsers 获取用户列表
// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:response(200, []models.User, "成功获取用户列表")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) GetUsers(ctx *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	// 获取查询参数
	role := ctx.Query("role")
	status := ctx.Query("status")
	search := ctx.Query("search")

	// 构建查询条件
	filters := map[string]interface{}{}
	if role != "" {
		filters["role"] = role
	}
	if status != "" {
		filters["status"] = status
	}

	users, total, err := c.userService.GetUsers(ctx, filters, search, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取用户列表失败",
			"error":   err.Error(),
		})
		return
	}

	// 返回分页响应
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":      users,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetUser 获取单个用户
// frame:route(method="GET", path="/:id")
// frame:summary(获取单个用户)
// frame:response(200, models.User, "成功获取用户信息")
// frame:response(404, gin.H, "用户不存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) GetUser(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	user, err := c.userService.GetUserByID(ctx, uint(id))
	if err != nil {
		if err.Error() == "用户不存在" {
			ctx.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "用户不存在",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取用户失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// CreateUser 创建用户
// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:bind(source="json", model="models.UserCreateRequest")
// frame:validate(groups="create")
// frame:response(201, models.User, "用户创建成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(409, gin.H, "用户名或邮箱已存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) CreateUser(ctx *gin.Context) {
	var req models.UserCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	createdUser, err := c.userService.CreateUser(ctx, &req)
	if err != nil {
		if err.Error() == "用户名已存在" || err.Error() == "邮箱已存在" {
			ctx.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": "创建失败",
				"error":   err.Error(),
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "创建用户失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdUser,
	})
}

// UpdateUser 更新用户
// frame:route(method="PUT", path="/:id")
// frame:summary(更新用户信息)
// frame:bind(source="json", model="models.UserUpdateRequest")
// frame:validate(groups="update")
// frame:response(200, models.User, "用户更新成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(404, gin.H, "用户不存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) UpdateUser(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	var req models.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	updatedUser, err := c.userService.UpdateUser(ctx, uint(id), &req)
	if err != nil {
		if err.Error() == "用户不存在" {
			ctx.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "用户不存在",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新用户失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedUser,
	})
}

// DeleteUser 删除用户
// frame:route(method="DELETE", path="/:id")
// frame:summary(删除用户)
// frame:response(200, gin.H, "用户删除成功")
// frame:response(400, gin.H, "无效的用户ID")
// frame:response(404, gin.H, "用户不存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) DeleteUser(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	err = c.userService.DeleteUser(ctx, uint(id))
	if err != nil {
		if err.Error() == "用户不存在" {
			ctx.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "用户不存在",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "删除用户失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "用户删除成功"},
	})
}

// GetUserStats 获取用户统计信息
// frame:route(method="GET", path="/stats")
// frame:summary(获取用户统计信息)
// frame:response(200, gin.H, "成功获取用户统计")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) GetUserStats(ctx *gin.Context) {
	stats, err := c.userService.GetUserStats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取用户统计失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
