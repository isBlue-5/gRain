package authz

import (
	"fmt"
	"strings"
)

// Permission 权限定义
type Permission struct {
	ID          string `json:"id" yaml:"id"`
	Resource    string `json:"resource" yaml:"resource"`
	Action      string `json:"action" yaml:"action"`
	Description string `json:"description" yaml:"description"`
}

// NewPermission 创建新权限
func NewPermission(resource, action string) *Permission {
	return &Permission{
		ID:       fmt.Sprintf("%s:%s", resource, action),
		Resource: resource,
		Action:   action,
	}
}

// String 返回权限字符串表示
func (p *Permission) String() string {
	return p.ID
}

// Matches 检查权限是否匹配模式
func (p *Permission) Matches(pattern string) bool {
	if pattern == "*" || pattern == "*:*" {
		return true
	}

	parts := strings.Split(pattern, ":")
	if len(parts) != 2 {
		return false
	}

	resourcePattern := parts[0]
	actionPattern := parts[1]

	// 检查资源匹配
	if resourcePattern != "*" && resourcePattern != p.Resource {
		return false
	}

	// 检查操作匹配
	if actionPattern != "*" && actionPattern != p.Action {
		return false
	}

	return true
}

// Role 角色定义
type Role struct {
	ID          string        `json:"id" yaml:"id"`
	Name        string        `json:"name" yaml:"name"`
	Description string        `json:"description" yaml:"description"`
	Permissions []*Permission `json:"permissions" yaml:"permissions"`
}

// NewRole 创建新角色
func NewRole(id, name string) *Role {
	return &Role{
		ID:          id,
		Name:        name,
		Permissions: make([]*Permission, 0),
	}
}

// AddPermission 为角色添加权限
func (r *Role) AddPermission(permission *Permission) {
	r.Permissions = append(r.Permissions, permission)
}

// HasPermission 检查角色是否拥有指定权限
func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p.ID == permission || p.Matches(permission) {
			return true
		}
	}
	return false
}

// GetPermissionIDs 获取角色所有权限ID
func (r *Role) GetPermissionIDs() []string {
	ids := make([]string, len(r.Permissions))
	for i, p := range r.Permissions {
		ids[i] = p.ID
	}
	return ids
}
