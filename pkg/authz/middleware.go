package authz

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/auth"
)

// AuthorizationConfig 授权配置
type AuthorizationConfig struct {
	Enabled     bool   `json:"enabled" yaml:"enabled" env:"AUTHZ_ENABLED" default:"true"`
	DefaultMode string `json:"defaultMode" yaml:"defaultMode" env:"AUTHZ_DEFAULT_MODE" default:"permissive"`

	// 预定义角色
	Roles []RoleConfig `json:"roles" yaml:"roles"`

	// 授权规则
	Rules []AuthorizationRule `json:"rules" yaml:"rules"`
}

// AuthorizationRule 授权规则定义
type AuthorizationRule struct {
	Path        string   `json:"path" yaml:"path"`
	Method      string   `json:"method" yaml:"method"`
	Roles       []string `json:"roles" yaml:"roles"`
	Permissions []string `json:"permissions" yaml:"permissions"`
	Expression  string   `json:"expression" yaml:"expression"`
}

// AuthConfig 方法级授权配置
type AuthConfig struct {
	Roles       []string
	Permissions []string
	Expression  string
}

// AuthorizationOption 授权中间件选项
type AuthorizationOption func(*AuthorizationConfig)

// WithRoles 设置角色要求
func WithRoles(roles ...string) AuthorizationOption {
	return func(config *AuthorizationConfig) {
		// 这里可以设置默认角色要求
	}
}

// WithPermissions 设置权限要求
func WithPermissions(permissions ...string) AuthorizationOption {
	return func(config *AuthorizationConfig) {
		// 这里可以设置默认权限要求
	}
}

// WithRules 设置授权规则
func WithRules(rules ...AuthorizationRule) AuthorizationOption {
	return func(config *AuthorizationConfig) {
		config.Rules = append(config.Rules, rules...)
	}
}

// AuthorizationMiddleware 创建授权中间件
func AuthorizationMiddleware(authorizer Authorizer, config AuthorizationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取身份信息
		identity, exists := auth.GetIdentityGin(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}

		// 查找匹配的路由规则
		path := c.Request.URL.Path
		method := c.Request.Method

		for _, rule := range config.Rules {
			if matchesPath(path, rule.Path) && (rule.Method == "" || rule.Method == method) {
				// 应用授权规则
				if !applyAuthorizationRule(authorizer, identity, rule) {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": "Forbidden: access denied by policy",
					})
					return
				}
				break
			}
		}

		c.Next()
	}
}

// RequireAuthMiddleware 要求认证的中间件
func RequireAuthMiddleware(authorizer Authorizer, config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取身份信息
		identity, exists := auth.GetIdentityGin(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}

		// 检查角色
		if len(config.Roles) > 0 {
			hasRole := false
			for _, role := range config.Roles {
				if authorizer.HasRole(identity, role) {
					hasRole = true
					break
				}
			}

			if !hasRole {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Forbidden: insufficient role",
				})
				return
			}
		}

		// 检查权限
		if len(config.Permissions) > 0 {
			hasPermission := false
			for _, perm := range config.Permissions {
				if authorizer.HasPermission(identity, perm) {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Forbidden: insufficient permission",
				})
				return
			}
		}

		// 评估表达式
		if config.Expression != "" {
			result, err := authorizer.Evaluate(identity, config.Expression)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Error evaluating authorization expression",
				})
				return
			}

			if !result {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Forbidden: expression evaluated to false",
				})
				return
			}
		}

		c.Next()
	}
}

// RequireRolesMiddleware 要求特定角色的中间件
func RequireRolesMiddleware(authorizer Authorizer, roles ...string) gin.HandlerFunc {
	return RequireAuthMiddleware(authorizer, AuthConfig{Roles: roles})
}

// RequirePermissionsMiddleware 要求特定权限的中间件
func RequirePermissionsMiddleware(authorizer Authorizer, permissions ...string) gin.HandlerFunc {
	return RequireAuthMiddleware(authorizer, AuthConfig{Permissions: permissions})
}

// 辅助函数

// matchesPath 检查路径是否匹配
func matchesPath(requestPath, rulePath string) bool {
	// 简单的路径匹配，支持通配符
	if rulePath == "*" {
		return true
	}

	// 精确匹配
	if requestPath == rulePath {
		return true
	}

	// 前缀匹配
	if len(rulePath) > 0 && rulePath[len(rulePath)-1] == '*' {
		prefix := rulePath[:len(rulePath)-1]
		return len(requestPath) >= len(prefix) && requestPath[:len(prefix)] == prefix
	}

	return false
}

// applyAuthorizationRule 应用授权规则
func applyAuthorizationRule(authorizer Authorizer, identity auth.Identity, rule AuthorizationRule) bool {
	// 检查角色
	if len(rule.Roles) > 0 {
		hasRole := false
		for _, role := range rule.Roles {
			if authorizer.HasRole(identity, role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return false
		}
	}

	// 检查权限
	if len(rule.Permissions) > 0 {
		hasPermission := false
		for _, perm := range rule.Permissions {
			if authorizer.HasPermission(identity, perm) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return false
		}
	}

	// 评估表达式
	if rule.Expression != "" {
		result, err := authorizer.Evaluate(identity, rule.Expression)
		if err != nil || !result {
			return false
		}
	}

	return true
}
