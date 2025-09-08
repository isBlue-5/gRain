# gRain 认证系统

gRain 框架提供强大而灵活的认证系统，支持多种认证方式（JWT、会话、基本认证等）和身份管理功能。

## 功能特点

- **多种认证方式**: 支持 JWT、会话、基本认证等多种认证方式
- **中间件集成**: 与 Gin 框架无缝集成，提供认证中间件
- **角色权限**: 支持基于角色的访问控制
- **链式 API**: 提供简洁易用的链式 API 设计
- **事件钩子**: 支持认证成功/失败事件钩子
- **可扩展**: 可轻松扩展自定义认证提供者

## 快速开始

### 基本使用

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/your-module/gRain/pkg/auth"
)

// 创建 JWT 认证提供者
jwtProvider := auth.NewJWTProvider(
    "your-secret-key",
    auth.WithExpiration(24*time.Hour),
)

// 创建认证管理器
authManager := auth.NewAuthManager()
authManager.Register(jwtProvider)

// 创建认证中间件
authMiddleware := auth.AuthMiddleware(
    authManager,
    auth.WithSkipPaths("/login", "/public"),
)

// 应用中间件
r := gin.Default()
r.Use(authMiddleware)

// 保护的路由
r.GET("/profile", func(c *gin.Context) {
    identity, _ := auth.GetIdentityGin(c)
    c.JSON(200, gin.H{
        "id": identity.GetID(),
        "username": identity.GetUsername(),
        "roles": identity.GetRoles(),
    })
})
```

### 生成 JWT 令牌

```go
// 创建令牌
token, err := jwtProvider.GenerateToken(map[string]interface{}{
    "sub": "user_123",
    "name": "John Doe",
    "roles": []string{"admin", "user"},
})
if err != nil {
    // 处理错误
}
```

### 会话认证

```go
// 创建会话存储
store := sessions.NewCookieStore([]byte("secret-key"))

// 创建会话认证提供者
sessionProvider := auth.NewSessionProvider("session-name", store)

// 注册到认证管理器
authManager.Register(sessionProvider)

// 创建会话
identity := auth.NewIdentity("123", "john", []string{"user"}, nil)
sessionProvider.CreateSession(c, identity)
```

### 要求角色

```go
// 仅允许管理员访问
r.GET("/admin", auth.RequireRolesMiddleware("admin"), func(c *gin.Context) {
    // 处理请求...
})
```

## 核心组件

### Identity（身份）

`Identity` 接口代表已认证的用户身份，提供以下方法：

- `GetID()`: 返回唯一标识符
- `GetUsername()`: 返回用户名
- `GetRoles()`: 返回用户角色列表
- `HasRole(role)`: 检查是否拥有特定角色
- `HasAnyRole(roles)`: 检查是否拥有任一指定角色
- `GetClaims()`: 获取附加信息

### AuthProvider（认证提供者）

`AuthProvider` 接口定义了如何从请求中提取和验证身份信息：

- `Authenticate(ctx)`: 从请求中解析身份信息
- `Name()`: 返回提供者名称

已实现的提供者包括：
- `JWTProvider`: JWT 令牌认证
- `SessionProvider`: 会话认证
- `BasicAuthProvider`: HTTP 基本认证

### AuthManager（认证管理器）

`AuthManager` 接口协调多个认证提供者：

- `Register(provider)`: 注册认证提供者
- `Authenticate(ctx)`: 尝试所有已注册提供者进行认证
- `GetProviders()`: 获取所有已注册提供者

### 中间件

框架提供以下中间件：

- `AuthMiddleware`: 基本认证中间件
- `RequireAuthMiddleware`: 要求认证的中间件
- `RequireRolesMiddleware`: 要求特定角色的中间件

## 高级用法

### 自定义认证提供者

实现 `AuthProvider` 接口即可创建自定义认证提供者：

```go
type CustomProvider struct {
    // 自定义字段
}

func (p *CustomProvider) Name() string {
    return "custom"
}

func (p *CustomProvider) Authenticate(ctx *gin.Context) (auth.Identity, error) {
    // 自定义认证逻辑
    // ...
    
    return identity, nil
}
```

### 认证事件处理

```go
authMiddleware := auth.AuthMiddleware(
    authManager,
    auth.WithSuccessHandler(func(c *gin.Context, identity auth.Identity) {
        // 认证成功处理
        log.Printf("User %s authenticated", identity.GetUsername())
    }),
    auth.WithFailureHandler(func(c *gin.Context, err error) {
        // 认证失败处理
        log.Printf("Authentication failed: %v", err)
        c.JSON(401, gin.H{"error": "Unauthorized access"})
    }),
)
```

### 匿名访问

```go
authMiddleware := auth.AuthMiddleware(
    authManager,
    auth.WithAnonymousAccess(),  // 允许匿名访问
)
```

## 示例

完整示例请参见 `examples/auth/main.go`。 