package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserService 用户服务接口
type UserService interface {
	GetByID(id string) (User, error)
	Create(user User) (string, error)
	List() ([]User, error)
}

// User 用户模型
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// frame:controller(path="/api/users")
type UserController struct {
	// 通过inject注解标记依赖注入
	UserService UserService `inject:""`
}

// GetUser 获取单个用户
// frame:route(method="GET", path="/:id")
// frame:auth(roles={"USER", "ADMIN"})
// frame:log(level="INFO")
func (c *UserController) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")

	user, err := c.UserService.GetByID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
// frame:route(method="POST", path="/")
// frame:auth(roles={"ADMIN"})
// frame:validation(rules={"name": "required,min=2", "age": "required,min=18"})
func (c *UserController) CreateUser(ctx *gin.Context) {
	var user User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := c.UserService.Create(user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"id": id})
}

// ListUsers 获取用户列表
// frame:route(method="GET", path="/")
// frame:auth(roles={"USER", "ADMIN"})
// frame:cache(ttl="5m", key="users")
func (c *UserController) ListUsers(ctx *gin.Context) {
	users, err := c.UserService.List()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, users)
}
