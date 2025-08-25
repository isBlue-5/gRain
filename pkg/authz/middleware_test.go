package authz

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequirePermissionsMiddleware(t *testing.T) {
	// 创建权限授权器
	authorizer := NewMemoryAuthorizer()

	// 创建角色和权限
	adminRole := NewRole("admin", "Administrator")
	adminRole.AddPermission(NewPermission("user", "read"))
	authorizer.AddRole(adminRole)

	userRole := NewRole("user", "User")
	userRole.AddPermission(NewPermission("user", "read"))
	authorizer.AddRole(userRole)

	// 测试中间件创建
	middleware := RequirePermissionsMiddleware(authorizer, "user:read")
	if middleware == nil {
		t.Errorf("中间件创建失败")
	}

	// 测试中间件类型
	// 这里我们只测试中间件能够创建，实际的HTTP测试需要更复杂的设置
	t.Logf("中间件创建成功，类型: %T", middleware)
}

func TestRequireRolesMiddleware(t *testing.T) {
	// 创建权限授权器
	authorizer := NewMemoryAuthorizer()

	// 创建角色
	adminRole := NewRole("admin", "Administrator")
	authorizer.AddRole(adminRole)

	userRole := NewRole("user", "User")
	authorizer.AddRole(userRole)

	// 测试角色中间件创建
	middleware := RequireRolesMiddleware(authorizer, "admin")
	if middleware == nil {
		t.Errorf("角色中间件创建失败")
	}

	// 测试中间件类型
	t.Logf("角色中间件创建成功，类型: %T", middleware)
}

func TestMiddlewareFactory(t *testing.T) {
	// 创建权限授权器
	authorizer := NewMemoryAuthorizer()

	// 测试不同配置的中间件创建
	testCases := []struct {
		name        string
		permissions []string
		roles       []string
	}{
		{
			name:        "权限中间件",
			permissions: []string{"user:read"},
			roles:       []string{},
		},
		{
			name:        "角色中间件",
			permissions: []string{},
			roles:       []string{"admin"},
		},
		{
			name:        "混合中间件",
			permissions: []string{"user:read"},
			roles:       []string{"admin"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var middleware gin.HandlerFunc

			if len(tc.permissions) > 0 {
				middleware = RequirePermissionsMiddleware(authorizer, tc.permissions...)
			} else if len(tc.roles) > 0 {
				middleware = RequireRolesMiddleware(authorizer, tc.roles...)
			}

			if middleware == nil {
				t.Errorf("中间件创建失败")
			}

			t.Logf("%s 创建成功", tc.name)
		})
	}
}
