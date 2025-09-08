# 权限控制系统 (Authorization)

gRain框架的权限控制系统提供了基于角色的访问控制（RBAC）功能，支持灵活的权限管理和细粒度的访问控制。

## 功能特性

- **基于角色的访问控制（RBAC）**：支持角色和权限的层次化管理
- **多种授权策略**：支持基于角色、基于权限和基于表达式的授权
- **中间件集成**：与Gin框架无缝集成，支持路由级和方法级权限控制
- **灵活配置**：支持内存和数据库两种权限存储方式
- **表达式支持**：支持复杂的权限表达式评估
- **上下文集成**：与认证系统无缝集成，复用身份信息

## 快速开始

### 1. 基本使用

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/isBlue-5/gRain/pkg/auth"
    "github.com/isBlue-5/gRain/pkg/authz"
)

func main() {
    // 创建认证管理器
    authManager := auth.NewAuthManager()
    
    // 创建授权器
    authorizer := authz.NewDefaultAuthorizer()
    
    r := gin.Default()
    
    // 应用认证中间件
    r.Use(auth.AuthMiddleware(authManager))
    
    // 需要管理员角色的接口
    r.GET("/admin", authz.RequireRolesMiddleware(authorizer, "admin"), func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Admin area"})
    })
    
    // 需要特定权限的接口
    r.GET("/users", authz.RequirePermissionsMiddleware(authorizer, "user:read"), func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Users list"})
    })
    
    r.Run(":8080")
}
```

### 2. 自定义授权配置

```go
// 创建自定义授权配置
config := authz.AuthorizationConfig{
    Enabled: true,
    Roles: []authz.RoleConfig{
        {
            ID:          "manager",
            Name:        "Manager",
            Description: "Department manager",
            Permissions: []string{"user:read", "user:update", "report:view"},
        },
    },
    Rules: []authz.AuthorizationRule{
        {
            Path:        "/api/managers/*",
            Method:      "",
            Roles:       []string{"manager"},
            Permissions: []string{},
        },
    },
}

authorizer := authz.NewAuthorizer(config)
```

## 核心组件

### 1. 权限模型

#### Permission（权限）
```go
permission := authz.NewPermission("user", "create")
permission.Description = "Create new users"
```

#### Role（角色）
```go
role := authz.NewRole("admin", "Administrator")
role.AddPermission(authz.NewPermission("user", "*"))
role.AddPermission(authz.NewPermission("system", "*"))
```

### 2. 授权决策器

#### Authorizer接口
```go
type Authorizer interface {
    HasPermission(identity auth.Identity, permission string) bool
    HasRole(identity auth.Identity, role string) bool
    Evaluate(identity auth.Identity, expression string) (bool, error)
    AddRole(role *Role)
    GetRole(roleID string) (*Role, bool)
    GetAllRoles() []*Role
}
```

#### MemoryAuthorizer（内存授权器）
```go
authorizer := authz.NewMemoryAuthorizer()
authorizer.AddRole(role)
```

### 3. 中间件

#### 角色要求中间件
```go
r.GET("/admin", authz.RequireRolesMiddleware(authorizer, "admin"), handler)
```

#### 权限要求中间件
```go
r.GET("/users", authz.RequirePermissionsMiddleware(authorizer, "user:read"), handler)
```

#### 自定义授权中间件
```go
r.GET("/users", authz.RequireAuthMiddleware(authorizer, authz.AuthConfig{
    Roles:       []string{"admin", "manager"},
    Permissions: []string{"user:read"},
    Expression:  "hasRole('admin') || hasPermission('user:*')",
}), handler)
```

## 高级用法

### 1. 复杂权限检查

```go
func (c *UserController) GetUser(ctx *gin.Context) {
    identity, _ := auth.GetIdentityGin(ctx)
    userID := ctx.Param("id")
    
    // 检查是否有读取用户权限
    if !authorizer.HasPermission(identity, "user:read") {
        ctx.JSON(403, gin.H{"error": "Insufficient permission"})
        return
    }
    
    // 如果不是管理员，只能查看自己的信息
    if !authorizer.HasRole(identity, "admin") && identity.GetID() != userID {
        ctx.JSON(403, gin.H{"error": "Can only view own profile"})
        return
    }
    
    // 处理请求...
}
```

### 2. 权限表达式

```go
// 支持复杂的权限表达式
expression := "hasRole('admin') || (hasPermission('user:read') && resourceBelongsTo(#userId))"
result, err := authorizer.Evaluate(identity, expression)
```

### 3. 批量权限检查

```go
// 检查是否拥有任意一个权限
hasAny := authz.HasAnyPermission(authorizer, identity, []string{"user:read", "user:write"})

// 检查是否拥有所有权限
hasAll := authz.HasAllPermissions(authorizer, identity, []string{"user:read", "user:write"})

// 检查是否拥有任意一个角色
hasAnyRole := authz.HasAnyRole(authorizer, identity, []string{"admin", "manager"})
```

### 4. 权限验证

```go
// 验证权限格式
err := authz.ValidatePermission("user:create")
if err != nil {
    // 处理错误
}

// 验证角色
err = authz.ValidateRole(role)
if err != nil {
    // 处理错误
}
```

## 配置选项

### AuthorizationConfig

```go
type AuthorizationConfig struct {
    Enabled     bool   `json:"enabled" yaml:"enabled" env:"AUTHZ_ENABLED" default:"true"`
    DefaultMode string `json:"defaultMode" yaml:"defaultMode" env:"AUTHZ_DEFAULT_MODE" default:"permissive"`
    
    // 预定义角色
    Roles []RoleConfig `json:"roles" yaml:"roles"`
    
    // 授权规则
    Rules []AuthorizationRule `json:"rules" yaml:"rules"`
}
```

### AuthorizationRule

```go
type AuthorizationRule struct {
    Path        string   `json:"path" yaml:"path"`
    Method      string   `json:"method" yaml:"method"`
    Roles       []string `json:"roles" yaml:"roles"`
    Permissions []string `json:"permissions" yaml:"permissions"`
    Expression  string   `json:"expression" yaml:"expression"`
}
```

## 最佳实践

### 1. 权限命名规范

- 使用 `resource:action` 格式
- 资源名使用小写，用冒号分隔层级
- 操作名使用小写，常见操作：create, read, update, delete, list

```
user:create
user:read
user:update
user:delete
system:config
admin:*
```

### 2. 角色设计

- 角色应该反映业务职责
- 避免角色过多，保持简洁
- 使用继承关系减少重复权限

```
admin: 系统管理员
manager: 部门经理
user: 普通用户
guest: 访客
```

### 3. 权限检查时机

- 在路由层面进行粗粒度权限检查
- 在业务逻辑中进行细粒度权限检查
- 在数据访问层进行数据权限检查

### 4. 错误处理

```go
// 统一的权限错误处理
func handlePermissionError(c *gin.Context, err error) {
    c.JSON(403, gin.H{
        "error": "Permission denied",
        "message": err.Error(),
    })
}
```

## 示例

完整的使用示例请参考 `examples/authz/main.go`，该示例演示了：

- 用户认证和授权集成
- 不同角色的权限控制
- 路由级和方法级权限检查
- 复杂权限逻辑实现
- 权限检查工具接口

## 扩展开发

### 1. 自定义授权器

```go
type CustomAuthorizer struct {
    // 自定义实现
}

func (a *CustomAuthorizer) HasPermission(identity auth.Identity, permission string) bool {
    // 自定义权限检查逻辑
    return true
}

// 实现其他接口方法...
```

### 2. 数据库集成

```go
type DBAuthorizer struct {
    db *gorm.DB
}

func (a *DBAuthorizer) HasPermission(identity auth.Identity, permission string) bool {
    // 从数据库查询权限
    var count int64
    a.db.Model(&UserPermission{}).
        Where("user_id = ? AND permission = ?", identity.GetID(), permission).
        Count(&count)
    return count > 0
}
```

## 注意事项

1. **性能考虑**：权限检查应该在请求早期进行，避免不必要的计算
2. **缓存策略**：对于频繁的权限检查，考虑使用缓存
3. **安全原则**：遵循最小权限原则，默认拒绝访问
4. **测试覆盖**：确保权限逻辑有充分的测试覆盖
5. **日志记录**：记录权限检查失败的情况，便于安全审计 