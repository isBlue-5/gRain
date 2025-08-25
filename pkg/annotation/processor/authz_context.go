package processor

import (
	"context"
	"strings"
	"sync"

	"github.com/grain-framework/grain/pkg/authz"
)

// AuthzContextKey 授权器上下文键
type AuthzContextKey string

const (
	AuthorizerKey AuthzContextKey = "grain_authorizer"
	UserRolesKey  AuthzContextKey = "grain_user_roles"
	UserPermsKey  AuthzContextKey = "grain_user_permissions"
)

// AuthzContextManager 授权器上下文管理器
type AuthzContextManager struct {
	defaultAuthorizer authz.Authorizer
	roleStore         map[string][]string // userID -> roles
	permissionStore   map[string][]string // userID -> permissions
	mutex             sync.RWMutex
}

// NewAuthzContextManager 创建授权器上下文管理器
func NewAuthzContextManager() *AuthzContextManager {
	manager := &AuthzContextManager{
		roleStore:       make(map[string][]string),
		permissionStore: make(map[string][]string),
	}

	// 设置默认测试数据
	manager.setupTestData()

	return manager
}

// setupTestData 设置测试数据
func (m *AuthzContextManager) setupTestData() {
	// 设置测试用户角色
	m.roleStore["admin"] = []string{"admin", "user"}
	m.roleStore["user123"] = []string{"user"}
	m.roleStore["demo_user"] = []string{"user"}
	m.roleStore["guest"] = []string{"guest"}

	// 设置测试用户权限
	m.permissionStore["admin"] = []string{"admin:read", "admin:write", "admin:delete", "user:read", "user:write", "user:delete"}
	m.permissionStore["user123"] = []string{"user:read", "user:write"}
	m.permissionStore["demo_user"] = []string{"user:read", "user:write"}
	m.permissionStore["guest"] = []string{"user:read"}
}

// GetUserRoles 获取用户角色
func (m *AuthzContextManager) GetUserRoles(userID string) ([]string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	roles, exists := m.roleStore[userID]
	return roles, exists
}

// GetUserPermissions 获取用户权限
func (m *AuthzContextManager) GetUserPermissions(userID string) ([]string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	permissions, exists := m.permissionStore[userID]
	return permissions, exists
}

// SetUserRoles 设置用户角色
func (m *AuthzContextManager) SetUserRoles(userID string, roles []string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.roleStore[userID] = roles
}

// SetUserPermissions 设置用户权限
func (m *AuthzContextManager) SetUserPermissions(userID string, permissions []string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.permissionStore[userID] = permissions
}

// HasRequiredRoles 检查用户是否具有所需角色
func (m *AuthzContextManager) HasRequiredRoles(userID string, requiredRoles []string) bool {
	if userID == "" || len(requiredRoles) == 0 {
		return false
	}

	userRoles, exists := m.GetUserRoles(userID)
	if !exists {
		return false
	}

	// 检查是否至少拥有一个所需角色
	for _, requiredRole := range requiredRoles {
		for _, userRole := range userRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}

	return false
}

// HasRequiredPermissions 检查用户是否具有所需权限
func (m *AuthzContextManager) HasRequiredPermissions(userID string, requiredPermissions []string) bool {
	if userID == "" || len(requiredPermissions) == 0 {
		return false
	}

	userPermissions, exists := m.GetUserPermissions(userID)
	if !exists {
		return false
	}

	// 检查是否拥有所有所需权限
	for _, requiredPerm := range requiredPermissions {
		hasPermission := false
		for _, userPerm := range userPermissions {
			if userPerm == requiredPerm {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return false
		}
	}

	return true
}

// EvaluateExpression 评估权限表达式
func (m *AuthzContextManager) EvaluateExpression(userID string, expression string) bool {
	if userID == "" || expression == "" {
		return false
	}

	// 简单的表达式评估器实现
	// 支持基本的表达式：user.role == 'admin'、user.role == 'user'等

	// 解析表达式
	if strings.Contains(expression, "user.role == 'admin'") {
		return m.HasRequiredRoles(userID, []string{"admin"})
	}

	if strings.Contains(expression, "user.role == 'user'") {
		return m.HasRequiredRoles(userID, []string{"user"})
	}

	if strings.Contains(expression, "user.department == 'IT'") {
		// 模拟部门检查，对于admin用户返回true
		return m.HasRequiredRoles(userID, []string{"admin"})
	}

	// 更复杂的表达式解析
	if strings.Contains(expression, "&&") {
		parts := strings.Split(expression, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if !m.EvaluateExpression(userID, part) {
				return false
			}
		}
		return true
	}

	if strings.Contains(expression, "||") {
		parts := strings.Split(expression, "||")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if m.EvaluateExpression(userID, part) {
				return true
			}
		}
		return false
	}

	// 默认返回false，表示表达式无法解析
	return false
}

// WithAuthzContext 将授权器管理器添加到上下文
func WithAuthzContext(ctx context.Context, manager *AuthzContextManager) context.Context {
	return context.WithValue(ctx, AuthorizerKey, manager)
}

// GetAuthzContext 从上下文获取授权器管理器
func GetAuthzContext(ctx context.Context) (*AuthzContextManager, bool) {
	manager, ok := ctx.Value(AuthorizerKey).(*AuthzContextManager)
	return manager, ok
}

// 全局授权器管理器实例（用于注解生成的代码）
var globalAuthzManager *AuthzContextManager
var authzOnce sync.Once

// GetGlobalAuthzManager 获取全局授权器管理器
func GetGlobalAuthzManager() *AuthzContextManager {
	authzOnce.Do(func() {
		globalAuthzManager = NewAuthzContextManager()
	})
	return globalAuthzManager
}

// 以下是导出的辅助函数，用于注解生成的代码

// GetUserID 获取用户ID的辅助函数
func GetUserID(c interface{}) string {
	// 尝试转换为gin.Context
	if ginCtx, ok := c.(interface {
		Get(key string) (value interface{}, exists bool)
		GetHeader(key string) string
	}); ok {
		// 优先从用户上下文获取
		if user, exists := ginCtx.Get("user"); exists {
			if userID, ok := user.(string); ok {
				return userID
			}
		}

		// 尝试从user_id字段获取
		if userID, exists := ginCtx.Get("user_id"); exists {
			if id, ok := userID.(string); ok {
				return id
			}
		}

		// 从请求头获取
		if userID := ginCtx.GetHeader("X-User-ID"); userID != "" {
			return userID
		}
	}

	return ""
}

// HasRequiredRoles 检查用户是否具有所需角色
func HasRequiredRoles(userID string, requiredRoles []string) bool {
	manager := GetGlobalAuthzManager()
	return manager.HasRequiredRoles(userID, requiredRoles)
}

// HasRequiredPermissions 检查用户是否具有所需权限
func HasRequiredPermissions(userID string, requiredPermissions []string) bool {
	manager := GetGlobalAuthzManager()
	return manager.HasRequiredPermissions(userID, requiredPermissions)
}

// EvaluateExpression 评估权限表达式
func EvaluateExpression(userID string, expression string) bool {
	manager := GetGlobalAuthzManager()
	return manager.EvaluateExpression(userID, expression)
}
