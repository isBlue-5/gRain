package authz

import (
	"fmt"
	"strings"

	"github.com/isBlue-5/grain/pkg/auth"
)

// NewAuthorizer 创建授权决策器
func NewAuthorizer(config AuthorizationConfig) Authorizer {
	// 基于配置创建默认授权器
	authorizer := NewMemoryAuthorizer()

	// 加载预定义角色和权限
	if len(config.Roles) > 0 {
		authorizer.LoadRolesFromConfig(config.Roles)
	}

	return authorizer
}

// NewDefaultAuthorizer 创建默认授权器
func NewDefaultAuthorizer() Authorizer {
	// 创建默认配置
	config := AuthorizationConfig{
		Enabled:     true,
		DefaultMode: "permissive",
		Roles: []RoleConfig{
			{
				ID:          "admin",
				Name:        "Administrator",
				Description: "System administrator",
				Permissions: []string{"user:*", "system:*", "admin:*"},
			},
			{
				ID:          "user",
				Name:        "Regular user",
				Description: "Regular system user",
				Permissions: []string{"user:view", "user:edit", "profile:*"},
			},
			{
				ID:          "guest",
				Name:        "Guest user",
				Description: "Guest user with limited access",
				Permissions: []string{"public:*"},
			},
		},
		Rules: []AuthorizationRule{
			{
				Path:        "/admin/*",
				Method:      "",
				Roles:       []string{"admin"},
				Permissions: []string{},
				Expression:  "",
			},
			{
				Path:        "/api/users/*",
				Method:      "",
				Roles:       []string{"admin", "user"},
				Permissions: []string{"user:*"},
				Expression:  "",
			},
			{
				Path:        "/api/public/*",
				Method:      "",
				Roles:       []string{},
				Permissions: []string{},
				Expression:  "",
			},
		},
	}

	return NewAuthorizer(config)
}

// CreateRole 创建角色
func CreateRole(id, name, description string, permissions ...string) *Role {
	role := NewRole(id, name)
	role.Description = description

	for _, permStr := range permissions {
		parts := strings.Split(permStr, ":")
		if len(parts) == 2 {
			permission := NewPermission(parts[0], parts[1])
			role.AddPermission(permission)
		}
	}

	return role
}

// CreatePermission 创建权限
func CreatePermission(resource, action, description string) *Permission {
	permission := NewPermission(resource, action)
	permission.Description = description
	return permission
}

// ValidatePermission 验证权限格式
func ValidatePermission(permission string) error {
	parts := strings.Split(permission, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid permission format: %s, expected format: resource:action", permission)
	}

	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid permission format: %s, resource and action cannot be empty", permission)
	}

	return nil
}

// ValidateRole 验证角色
func ValidateRole(role *Role) error {
	if role.ID == "" {
		return fmt.Errorf("role ID cannot be empty")
	}

	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	// 验证权限格式
	for _, permission := range role.Permissions {
		if err := ValidatePermission(permission.ID); err != nil {
			return fmt.Errorf("invalid permission in role %s: %w", role.ID, err)
		}
	}

	return nil
}

// MergePermissions 合并权限列表
func MergePermissions(permissions ...[]string) []string {
	permMap := make(map[string]bool)

	for _, permList := range permissions {
		for _, perm := range permList {
			permMap[perm] = true
		}
	}

	result := make([]string, 0, len(permMap))
	for perm := range permMap {
		result = append(result, perm)
	}

	return result
}

// FilterPermissions 过滤权限
func FilterPermissions(permissions []string, filter func(string) bool) []string {
	result := make([]string, 0)

	for _, perm := range permissions {
		if filter(perm) {
			result = append(result, perm)
		}
	}

	return result
}

// HasAnyPermission 检查是否拥有任意一个权限
func HasAnyPermission(authorizer Authorizer, identity auth.Identity, permissions []string) bool {
	for _, perm := range permissions {
		if authorizer.HasPermission(identity, perm) {
			return true
		}
	}
	return false
}

// HasAllPermissions 检查是否拥有所有权限
func HasAllPermissions(authorizer Authorizer, identity auth.Identity, permissions []string) bool {
	for _, perm := range permissions {
		if !authorizer.HasPermission(identity, perm) {
			return false
		}
	}
	return true
}

// HasAnyRole 检查是否拥有任意一个角色
func HasAnyRole(authorizer Authorizer, identity auth.Identity, roles []string) bool {
	for _, role := range roles {
		if authorizer.HasRole(identity, role) {
			return true
		}
	}
	return false
}

// HasAllRoles 检查是否拥有所有角色
func HasAllRoles(authorizer Authorizer, identity auth.Identity, roles []string) bool {
	for _, role := range roles {
		if !authorizer.HasRole(identity, role) {
			return false
		}
	}
	return true
}
