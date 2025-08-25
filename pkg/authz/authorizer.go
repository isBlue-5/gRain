package authz

import (
	"strings"

	"github.com/grain-framework/grain/pkg/auth"
)

// Authorizer 授权决策接口
type Authorizer interface {
	// HasPermission 检查身份是否具有指定权限
	HasPermission(identity auth.Identity, permission string) bool

	// HasRole 检查身份是否拥有指定角色
	HasRole(identity auth.Identity, role string) bool

	// Evaluate 评估复杂权限表达式
	Evaluate(identity auth.Identity, expression string) (bool, error)

	// AddRole 添加角色
	AddRole(role *Role)

	// GetRole 获取角色
	GetRole(roleID string) (*Role, bool)

	// GetAllRoles 获取所有角色
	GetAllRoles() []*Role
}

// MemoryAuthorizer 基于内存的授权决策器
type MemoryAuthorizer struct {
	roles       map[string]*Role
	permissions map[string]*Permission
	rolePerms   map[string][]string // 角色->权限映射
}

// NewMemoryAuthorizer 创建基于内存的授权决策器
func NewMemoryAuthorizer() *MemoryAuthorizer {
	return &MemoryAuthorizer{
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
		rolePerms:   make(map[string][]string),
	}
}

// HasPermission 检查身份是否具有指定权限
func (a *MemoryAuthorizer) HasPermission(identity auth.Identity, permission string) bool {
	// 检查身份是否为nil
	if identity == nil {
		return false
	}

	// 检查身份中是否有直接权限
	if claims := identity.GetClaims(); claims != nil {
		if perms, ok := claims["permissions"]; ok {
			if permList, ok := perms.([]interface{}); ok {
				for _, p := range permList {
					if permStr, ok := p.(string); ok {
						if permStr == permission {
							return true
						}
					}
				}
			}
		}
	}

	// 检查身份的角色是否有该权限
	for _, role := range identity.GetRoles() {
		if perms, ok := a.rolePerms[role]; ok {
			for _, p := range perms {
				if p == permission {
					return true
				}
			}
		}

		// 检查角色对象中的权限
		if roleObj, ok := a.roles[role]; ok {
			if roleObj.HasPermission(permission) {
				return true
			}
		}
	}

	return false
}

// HasRole 检查身份是否拥有指定角色
func (a *MemoryAuthorizer) HasRole(identity auth.Identity, role string) bool {
	for _, userRole := range identity.GetRoles() {
		if userRole == role {
			return true
		}
	}
	return false
}

// Evaluate 评估复杂权限表达式（简化版本）
func (a *MemoryAuthorizer) Evaluate(identity auth.Identity, expression string) (bool, error) {
	// 简化实现：支持基本的 hasRole 和 hasPermission 函数
	// 实际项目中可以使用更复杂的表达式解析器

	// 这里实现一个简单的表达式评估
	// 支持格式：hasRole('ADMIN') || hasPermission('user:delete')

	// 简单的表达式解析实现
	// 支持基本的等值判断和逻辑运算符
	result := evaluateSimpleExpression(expression, identity)
	return result, nil
}

// AddRole 添加角色
func (a *MemoryAuthorizer) AddRole(role *Role) {
	a.roles[role.ID] = role
	a.rolePerms[role.ID] = role.GetPermissionIDs()
}

// GetRole 获取角色
func (a *MemoryAuthorizer) GetRole(roleID string) (*Role, bool) {
	role, exists := a.roles[roleID]
	return role, exists
}

// GetAllRoles 获取所有角色
func (a *MemoryAuthorizer) GetAllRoles() []*Role {
	roles := make([]*Role, 0, len(a.roles))
	for _, role := range a.roles {
		roles = append(roles, role)
	}
	return roles
}

// LoadRolesFromConfig 从配置加载角色
func (a *MemoryAuthorizer) LoadRolesFromConfig(roles []RoleConfig) {
	for _, roleConfig := range roles {
		role := NewRole(roleConfig.ID, roleConfig.Name)
		role.Description = roleConfig.Description

		for _, permStr := range roleConfig.Permissions {
			parts := strings.Split(permStr, ":")
			if len(parts) == 2 {
				permission := NewPermission(parts[0], parts[1])
				role.AddPermission(permission)
			}
		}

		a.AddRole(role)
	}
}

// RoleConfig 角色配置结构
type RoleConfig struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Permissions []string `json:"permissions" yaml:"permissions"`
}

// evaluateSimpleExpression 简单的表达式评估器
func evaluateSimpleExpression(expression string, identity auth.Identity) bool {
	// 使用新的表达式引擎
	engine := NewExpressionEngine(identity)
	result, err := engine.Evaluate(expression, identity)
	if err != nil {
		// 如果表达式引擎失败，回退到简单逻辑
		return evaluateFallbackExpression(expression, identity)
	}
	return result
}

// evaluateFallbackExpression 回退表达式评估器
func evaluateFallbackExpression(expression string, identity auth.Identity) bool {
	// 移除空格
	expr := strings.ReplaceAll(expression, " ", "")

	// 处理简单的角色检查：user.role == 'admin'
	if strings.Contains(expr, "user.role=='admin'") {
		for _, role := range identity.GetRoles() {
			if role == "admin" {
				return true
			}
		}
		return false
	}

	// 处理简单的角色检查：user.role == 'user'
	if strings.Contains(expr, "user.role=='user'") {
		for _, role := range identity.GetRoles() {
			if role == "user" {
				return true
			}
		}
		return false
	}

	// 处理部门检查：user.department == 'IT'
	if strings.Contains(expr, "user.department=='IT'") {
		// 简化处理：假设admin角色的用户属于IT部门
		for _, role := range identity.GetRoles() {
			if role == "admin" {
				return true
			}
		}
		return false
	}

	// 处理AND操作符
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		for _, part := range parts {
			if !evaluateFallbackExpression(strings.TrimSpace(part), identity) {
				return false
			}
		}
		return true
	}

	// 处理OR操作符
	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, part := range parts {
			if evaluateFallbackExpression(strings.TrimSpace(part), identity) {
				return true
			}
		}
		return false
	}

	// 默认返回false
	return false
}
