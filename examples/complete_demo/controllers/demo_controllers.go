package controllers

import (
	"complete_demo/models"
	"complete_demo/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DemoController demo控制器
// frame:controller(path="/api/v1/demo")
type DemoController struct {
	userService *services.UserService
}

// NewDemoController 创建新的demo控制器
func NewDemoController(userService *services.UserService) *DemoController {
	return &DemoController{
		userService: userService,
	}
}

// GetUserById 获取用户
// frame:route(method="GET", path="/users/:id")
// frame:auth(roles={"ADMIN", "USER"})
// frame:validate(rules={"id": "required|int"})
// frame:response(200, models.User, "OK")
// frame:response(404, models.Error, "Not Found")
func (c *DemoController) GetUserById(ctx *gin.Context, id uint) (models.User, error) {
	byID, err := c.userService.GetUserByID(ctx, id)
	return *byID, err
}

type PageInfo struct {
	Page     int `form:"page" json:"page" binding:"required"`
	PageSize int `form:"page_size" json:"page_size" binding:"required"`
}

type UserQuery struct {
	Role   string `form:"role" json:"role" binding:"required"`
	Status string `form:"status" json:"status" binding:"required"`
	Search string `form:"search" json:"search" binding:"required"`
}

// GetUsers 获取用户列表
// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:response(200, []models.User, "成功获取用户列表")
// frame:response(500, gin.H, "服务器内部错误")
func (c *UserController) GetUserList(ctx *gin.Context, pageInfo PageInfo, userQuery UserQuery) (users []*models.User, total int64, err error) {

	// 构建查询条件
	filters := map[string]interface{}{}
	if userQuery.Role != "" {
		filters["role"] = userQuery.Role
	}
	if userQuery.Status != "" {
		filters["status"] = userQuery.Status
	}

	users, total, err = c.userService.GetUsers(ctx, filters, userQuery.Search, pageInfo.Page, pageInfo.PageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取用户列表失败",
			"error":   err.Error(),
		})
		return
	}

	return users, total, nil
}
