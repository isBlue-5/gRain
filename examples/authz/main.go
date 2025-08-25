package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/auth"
	"github.com/grain-framework/grain/pkg/authz"
)

// 模拟用户数据
var users = map[string]*User{
	"admin": {
		ID:       "admin",
		Username: "admin",
		Password: "admin123",
		Roles:    []string{"admin"},
		Permissions: []string{
			"user:create", "user:read", "user:update", "user:delete",
			"system:config", "system:monitor",
		},
	},
	"user1": {
		ID:       "user1",
		Username: "user1",
		Password: "user123",
		Roles:    []string{"user"},
		Permissions: []string{
			"user:read", "user:update",
			"profile:read", "profile:update",
		},
	},
	"guest": {
		ID:       "guest",
		Username: "guest",
		Password: "guest123",
		Roles:    []string{"guest"},
		Permissions: []string{
			"public:read",
		},
	},
}

// User 用户模型
type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Password    string   `json:"-"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func main() {
	// 创建认证管理器
	authManager := auth.NewAuthManager()

	// 添加JWT提供者
	jwtProvider := auth.NewJWTProvider("your-secret-key",
		auth.WithExpiration(24*time.Hour),
	)
	authManager.Register(jwtProvider)

	// 创建授权器
	authorizer := authz.NewDefaultAuthorizer()

	// 创建路由
	r := gin.Default()

	// 应用认证中间件
	r.Use(auth.AuthMiddleware(authManager,
		auth.WithSkipPaths("/login", "/public"),
		auth.WithFailureHandler(func(c *gin.Context, err error) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		}),
	))

	// 应用授权中间件
	authzConfig := authz.AuthorizationConfig{
		Enabled:     true,
		DefaultMode: "permissive",
		Rules: []authz.AuthorizationRule{
			{
				Path:        "/admin/*",
				Method:      "",
				Roles:       []string{"admin"},
				Permissions: []string{},
			},
			{
				Path:        "/api/users/*",
				Method:      "",
				Roles:       []string{"admin", "user"},
				Permissions: []string{"user:*"},
			},
			{
				Path:        "/api/system/*",
				Method:      "",
				Roles:       []string{"admin"},
				Permissions: []string{"system:*"},
			},
		},
	}
	r.Use(authz.AuthorizationMiddleware(authorizer, authzConfig))

	// 登录接口
	r.POST("/login", func(c *gin.Context) {
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		user, exists := users[loginReq.Username]
		if !exists || user.Password != loginReq.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// 生成JWT令牌
		token, err := jwtProvider.GenerateToken(map[string]interface{}{
			"user_id":     user.ID,
			"username":    user.Username,
			"roles":       user.Roles,
			"permissions": user.Permissions,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"id":          user.ID,
				"username":    user.Username,
				"roles":       user.Roles,
				"permissions": user.Permissions,
			},
		})
	})

	// 公开接口
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Public endpoint - no authentication required"})
	})

	// 需要认证的接口
	r.GET("/profile", func(c *gin.Context) {
		identity, _ := auth.GetIdentityGin(c)
		c.JSON(http.StatusOK, gin.H{
			"message": "Profile endpoint",
			"user": gin.H{
				"id":       identity.GetID(),
				"username": identity.GetUsername(),
				"roles":    identity.GetRoles(),
			},
		})
	})

	// 需要管理员角色的接口
	r.GET("/admin", authz.RequireRolesMiddleware(authorizer, "admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Admin area - admin role required"})
	})

	// 需要特定权限的接口
	r.GET("/api/users", authz.RequirePermissionsMiddleware(authorizer, "user:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Users list - user:read permission required",
			"users":   []string{"user1", "user2", "user3"},
		})
	})

	// 需要多个权限的接口
	r.POST("/api/users", func(c *gin.Context) {
		identity, _ := auth.GetIdentityGin(c)

		// 检查是否拥有创建用户的权限
		if !authorizer.HasPermission(identity, "user:create") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permission: user:create required"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
	})

	// 系统配置接口（需要系统权限）
	r.GET("/api/system/config", authz.RequirePermissionsMiddleware(authorizer, "system:config"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "System configuration - system:config permission required",
			"config": gin.H{
				"version": "1.0.0",
				"debug":   false,
			},
		})
	})

	// 复杂权限检查示例
	r.GET("/api/users/:id", func(c *gin.Context) {
		identity, _ := auth.GetIdentityGin(c)
		userID := c.Param("id")

		// 检查是否有读取用户权限
		if !authorizer.HasPermission(identity, "user:read") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permission: user:read required"})
			return
		}

		// 如果不是管理员，只能查看自己的信息
		if !authorizer.HasRole(identity, "admin") && identity.GetID() != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Can only view own profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "User details retrieved",
			"userID":  userID,
		})
	})

	// 权限检查工具接口
	r.GET("/check-permissions", func(c *gin.Context) {
		identity, _ := auth.GetIdentityGin(c)

		permissions := []string{"user:read", "user:create", "system:config", "admin:*"}
		roles := []string{"admin", "user", "guest"}

		permResults := make(map[string]bool)
		for _, perm := range permissions {
			permResults[perm] = authorizer.HasPermission(identity, perm)
		}

		roleResults := make(map[string]bool)
		for _, role := range roles {
			roleResults[role] = authorizer.HasRole(identity, role)
		}

		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":       identity.GetID(),
				"username": identity.GetUsername(),
				"roles":    identity.GetRoles(),
			},
			"permissions": permResults,
			"roles":       roleResults,
		})
	})

	// 角色管理接口（仅管理员）
	r.GET("/api/roles", authz.RequireRolesMiddleware(authorizer, "admin"), func(c *gin.Context) {
		allRoles := authorizer.GetAllRoles()
		roleList := make([]gin.H, len(allRoles))

		for i, role := range allRoles {
			roleList[i] = gin.H{
				"id":          role.ID,
				"name":        role.Name,
				"description": role.Description,
				"permissions": role.GetPermissionIDs(),
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Roles list - admin role required",
			"roles":   roleList,
		})
	})

	log.Println("Authorization server starting on :8080")
	log.Println("Available users:")
	log.Println("  admin/admin123 - Administrator")
	log.Println("  user1/user123 - Regular user")
	log.Println("  guest/guest123 - Guest user")

	r.Run(":8080")
}
