package authz

import (
	"fmt"
	"testing"
)

// MockIdentity 模拟身份信息用于测试
type MockIdentity struct {
	ID       string
	Username string
	Roles    []string
	Claims   map[string]interface{}
}

func (m MockIdentity) GetID() string {
	return m.ID
}

func (m MockIdentity) GetUsername() string {
	return m.Username
}

func (m MockIdentity) GetRoles() []string {
	return m.Roles
}

func (m MockIdentity) HasRole(role string) bool {
	for _, r := range m.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (m MockIdentity) HasAnyRole(roles []string) bool {
	for _, role := range roles {
		if m.HasRole(role) {
			return true
		}
	}
	return false
}

func (m MockIdentity) GetClaims() map[string]interface{} {
	return m.Claims
}

func TestMemoryAuthorizer(t *testing.T) {
	// 创建权限授权器
	authorizer := NewMemoryAuthorizer()

	// 创建角色和权限
	adminRole := NewRole("admin", "Administrator")
	adminRole.AddPermission(NewPermission("user", "read"))
	adminRole.AddPermission(NewPermission("user", "write"))
	authorizer.AddRole(adminRole)

	managerRole := NewRole("manager", "Manager")
	managerRole.AddPermission(NewPermission("user", "read"))
	authorizer.AddRole(managerRole)

	userRole := NewRole("user", "User")
	userRole.AddPermission(NewPermission("user", "read"))
	authorizer.AddRole(userRole)

	// 测试管理员权限
	adminIdentity := &MockIdentity{
		ID:       "admin1",
		Username: "admin",
		Roles:    []string{"admin"},
		Claims: map[string]interface{}{
			"department": "IT",
		},
	}

	// 测试管理员可以读取用户
	canRead := authorizer.HasPermission(adminIdentity, "user:read")
	if !canRead {
		t.Errorf("管理员应该能够读取用户")
	}

	// 测试管理员可以写入用户
	canWrite := authorizer.HasPermission(adminIdentity, "user:write")
	if !canWrite {
		t.Errorf("管理员应该能够写入用户")
	}

	// 测试管理员角色
	hasRole := authorizer.HasRole(adminIdentity, "admin")
	if !hasRole {
		t.Errorf("管理员应该具有admin角色")
	}

	// 测试经理权限
	managerIdentity := &MockIdentity{
		ID:       "manager1",
		Username: "manager",
		Roles:    []string{"manager"},
		Claims: map[string]interface{}{
			"department": "IT",
		},
	}

	// 测试经理可以读取用户
	canRead = authorizer.HasPermission(managerIdentity, "user:read")
	if !canRead {
		t.Errorf("经理应该能够读取用户")
	}

	// 测试经理不能写入用户
	canWrite = authorizer.HasPermission(managerIdentity, "user:write")
	if canWrite {
		t.Errorf("经理不应该能够写入用户")
	}

	// 测试普通用户权限
	userIdentity := &MockIdentity{
		ID:       "user1",
		Username: "user",
		Roles:    []string{"user"},
		Claims: map[string]interface{}{
			"department": "Sales",
		},
	}

	// 测试普通用户可以读取用户
	canRead = authorizer.HasPermission(userIdentity, "user:read")
	if !canRead {
		t.Errorf("普通用户应该能够读取用户")
	}

	// 测试普通用户不能写入用户
	canWrite = authorizer.HasPermission(userIdentity, "user:write")
	if canWrite {
		t.Errorf("普通用户不应该能够写入用户")
	}
}

func TestMemoryAuthorizerEdgeCases(t *testing.T) {
	authorizer := NewMemoryAuthorizer()

	// 测试空权限
	identity := &MockIdentity{
		ID:       "user1",
		Username: "user",
		Roles:    []string{"user"},
		Claims:   map[string]interface{}{},
	}

	// 测试不存在的权限
	canDo := authorizer.HasPermission(identity, "nonexistent:permission")
	if canDo {
		t.Errorf("不存在的权限应该返回false")
	}

	// 测试nil身份
	canDo = authorizer.HasPermission(nil, "user:read")
	if canDo {
		t.Errorf("nil身份应该返回false")
	}

	// 测试空角色
	emptyIdentity := &MockIdentity{
		ID:       "empty1",
		Username: "empty",
		Roles:    []string{},
		Claims:   map[string]interface{}{},
	}

	canDo = authorizer.HasPermission(emptyIdentity, "user:read")
	if canDo {
		t.Errorf("空角色身份应该返回false")
	}
}

func TestPermission(t *testing.T) {
	// 测试权限创建
	perm := NewPermission("user", "read")
	if perm.Resource != "user" {
		t.Errorf("期望资源为user，实际得到%s", perm.Resource)
	}
	if perm.Action != "read" {
		t.Errorf("期望动作为read，实际得到%s", perm.Action)
	}
	if perm.ID != "user:read" {
		t.Errorf("期望ID为user:read，实际得到%s", perm.ID)
	}

	// 测试权限匹配
	if !perm.Matches("user:read") {
		t.Errorf("权限应该匹配user:read")
	}

	if perm.Matches("user:write") {
		t.Errorf("权限不应该匹配user:write")
	}

	if !perm.Matches("*:*") {
		t.Errorf("权限应该匹配通配符*:*")
	}
}

func TestRole(t *testing.T) {
	// 测试角色创建
	role := NewRole("admin", "Administrator")
	if role.ID != "admin" {
		t.Errorf("期望角色ID为admin，实际得到%s", role.ID)
	}
	if role.Name != "Administrator" {
		t.Errorf("期望角色名称为Administrator，实际得到%s", role.Name)
	}

	// 测试添加权限
	perm := NewPermission("user", "read")
	role.AddPermission(perm)
	if len(role.Permissions) != 1 {
		t.Errorf("期望权限数量为1，实际得到%d", len(role.Permissions))
	}

	// 测试权限检查
	if !role.HasPermission("user:read") {
		t.Errorf("角色应该具有user:read权限")
	}

	if role.HasPermission("user:write") {
		t.Errorf("角色不应该具有user:write权限")
	}

	// 测试获取权限ID
	permIDs := role.GetPermissionIDs()
	if len(permIDs) != 1 || permIDs[0] != "user:read" {
		t.Errorf("期望权限ID为[user:read]，实际得到%v", permIDs)
	}
}

func TestMemoryAuthorizerPerformance(t *testing.T) {
	authorizer := NewMemoryAuthorizer()

	// 添加大量角色和权限
	for i := 0; i < 100; i++ {
		roleID := fmt.Sprintf("role%d", i)
		role := NewRole(roleID, fmt.Sprintf("Role %d", i))
		role.AddPermission(NewPermission("resource", fmt.Sprintf("action%d", i)))
		authorizer.AddRole(role)
	}

	identity := &MockIdentity{
		ID:       "user1",
		Username: "user",
		Roles:    []string{"role50"},
		Claims:   map[string]interface{}{},
	}

	// 运行多次权限检查以测试性能
	for i := 0; i < 1000; i++ {
		canDo := authorizer.HasPermission(identity, "resource:action50")
		if !canDo {
			t.Fatalf("期望结果为true，实际得到false")
		}
	}
}

// BenchmarkMemoryAuthorizer 性能基准测试
func BenchmarkMemoryAuthorizer(b *testing.B) {
	authorizer := NewMemoryAuthorizer()
	role := NewRole("admin", "Administrator")
	role.AddPermission(NewPermission("user", "read"))
	authorizer.AddRole(role)

	identity := &MockIdentity{
		ID:       "admin1",
		Username: "admin",
		Roles:    []string{"admin"},
		Claims:   map[string]interface{}{},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = authorizer.HasPermission(identity, "user:read")
	}
}
