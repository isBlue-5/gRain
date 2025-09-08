# gRain框架自动参数绑定功能说明

## 🚀 功能概述

gRain框架现在支持**真正的自动参数绑定**！这意味着您不再需要在控制器中手动解析HTTP请求数据，框架会自动将请求数据绑定到您定义的结构体参数中。

## ✨ 核心特性

### 1. 自动参数绑定
- **URI参数自动绑定**：`/users/:id` 自动绑定到 `id uint` 参数
- **查询参数自动绑定**：`?page=1&size=10` 自动绑定到结构体字段
- **请求体自动绑定**：JSON/Form数据自动绑定到结构体
- **请求头自动绑定**：自定义请求头自动绑定到参数
- **Cookie自动绑定**：Cookie值自动绑定到参数

### 2. 自动验证
- **参数验证**：基于结构体标签自动验证
- **业务验证**：支持自定义验证规则
- **错误处理**：自动返回验证错误信息

### 3. 自动权限控制
- **角色检查**：基于注解自动检查用户角色
- **权限验证**：基于注解自动验证用户权限
- **安全拦截**：未授权请求自动拦截

## 🎯 使用方法

### 1. 定义请求结构体

```go
// 用户创建请求
type UserCreateRequest struct {
    Username string `json:"username" binding:"required" validate:"required,min=3,max=20"`
    Email    string `json:"email" binding:"required" validate:"required,email"`
    Password string `json:"password" binding:"required" validate:"required,min=6"`
    Role     string `json:"role" binding:"required" validate:"required,oneof=ADMIN USER"`
}

// 用户查询请求
type UserQueryRequest struct {
    Role     string `query:"role" validate:"omitempty,oneof=ADMIN USER"`
    Status   string `query:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
    Search   string `query:"search" validate:"omitempty,max=100"`
    Page     int    `query:"page" validate:"omitempty,min=1"`
    PageSize int    `query:"page_size" validate:"omitempty,min=1,max=100"`
}
```

### 2. 编写控制器方法

```go
// CreateUser 创建用户
// frame:route(method="POST", path="/users")
// frame:auth(roles={"ADMIN"})
// frame:validate(rules={"username": "required|min:3|max:20", "email": "required|email"})
func (c *AutoBindingController) CreateUser(ctx *gin.Context, req UserCreateRequest) (*models.User, error) {
    // gRain框架自动绑定请求数据到req结构体
    // 自动验证参数
    // 自动权限检查
    
    // 直接使用绑定后的数据
    user := &models.User{
        Username: req.Username,
        Email:    req.Email,
        Password: req.Password,
        Role:     req.Role,
    }
    
    // 业务逻辑...
    return user, nil
}

// GetUsers 获取用户列表
// frame:route(method="GET", path="/users")
func (c *AutoBindingController) GetUsers(ctx *gin.Context, req UserQueryRequest) ([]*models.User, int64, error) {
    // gRain框架自动绑定查询参数到req结构体
    
    // 设置默认值
    if req.Page <= 0 {
        req.Page = 1
    }
    if req.PageSize <= 0 {
        req.PageSize = 10
    }
    
    // 使用绑定后的查询参数
    filters := map[string]interface{}{}
    if req.Role != "" {
        filters["role"] = req.Role
    }
    if req.Status != "" {
        filters["status"] = req.Status
    }
    
    // 调用服务层...
    return users, total, nil
}

// UpdateUser 更新用户
// frame:route(method="PUT", path="/users/:id")
func (c *AutoBindingController) UpdateUser(ctx *gin.Context, id uint, req UserUpdateRequest) (*models.User, error) {
    // gRain框架自动绑定URI参数id和请求体到req结构体
    
    // 直接使用绑定后的数据
    existingUser, err := c.userService.GetUserByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 更新用户信息
    if req.Username != "" {
        existingUser.Username = req.Username
    }
    if req.Email != "" {
        existingUser.Email = req.Email
    }
    
    // 调用服务层...
    return existingUser, nil
}
```

### 3. 启动框架

```go
func main() {
    // ... 配置和数据库初始化 ...
    
    // 🎯 gRain框架增强版的魔法时刻！
    // 支持自动参数绑定 + 自动路由注册
    log.Println("🚀 启动gRain框架增强版自动路由注册...")
    generated.RegisterAllEnhancedgRainRoutes(router, cfg, dbManager)
    log.Println("✅ gRain框架增强版自动路由注册完成！")
    
    // ... 启动服务器 ...
}
```

## 🔧 支持的绑定类型

### 1. URI参数绑定
```go
// 路径: /users/:id
func (c *Controller) GetUser(ctx *gin.Context, id uint) (*models.User, error) {
    // id 自动绑定为 uint 类型
}
```

### 2. 查询参数绑定
```go
type QueryRequest struct {
    Page     int    `query:"page" validate:"min=1"`
    PageSize int    `query:"page_size" validate:"min=1,max=100"`
    Search   string `query:"search" validate:"max=100"`
}

// 请求: GET /users?page=1&page_size=10&search=john
func (c *Controller) GetUsers(ctx *gin.Context, req QueryRequest) ([]*models.User, error) {
    // req.Page = 1, req.PageSize = 10, req.Search = "john"
}
```

### 3. 请求体绑定
```go
type CreateRequest struct {
    Name  string `json:"name" binding:"required" validate:"required,min=1"`
    Email string `json:"email" binding:"required" validate:"required,email"`
}

// 请求: POST /users
// Body: {"name": "John", "email": "john@example.com"}
func (c *Controller) CreateUser(ctx *gin.Context, req CreateRequest) (*models.User, error) {
    // req.Name = "John", req.Email = "john@example.com"
}
```

### 4. 请求头绑定
```go
type HeaderRequest struct {
    Token string `header:"Authorization" validate:"required"`
    Lang  string `header:"Accept-Language" validate:"omitempty,oneof=en zh"`
}

// 请求头: Authorization: Bearer token123, Accept-Language: en
func (c *Controller) ProtectedAPI(ctx *gin.Context, req HeaderRequest) error {
    // req.Token = "Bearer token123", req.Lang = "en"
}
```

### 5. Cookie绑定
```go
type CookieRequest struct {
    SessionID string `cookie:"session_id" validate:"required"`
    Theme     string `cookie:"theme" validate:"omitempty,oneof=light dark"`
}

// Cookie: session_id=abc123; theme=dark
func (c *Controller) UserProfile(ctx *gin.Context, req CookieRequest) error {
    // req.SessionID = "abc123", req.Theme = "dark"
}
```

## 🛡️ 自动验证功能

### 1. 结构体标签验证
```go
type UserRequest struct {
    Username string `json:"username" binding:"required" validate:"required,min=3,max=20"`
    Email    string `json:"email" binding:"required" validate:"required,email"`
    Age      int    `json:"age" validate:"min=18,max=100"`
    Role     string `json:"role" validate:"oneof=ADMIN USER MODERATOR"`
}
```

### 2. 自定义验证规则
```go
// frame:validate(rules={"username": "required|min:3|max:20|unique", "email": "required|email|unique"})
func (c *Controller) CreateUser(ctx *gin.Context, req UserRequest) error {
    // 框架自动验证所有规则
}
```

### 3. 验证组
```go
// frame:validate(groups={"create", "update"})
func (c *Controller) UpdateUser(ctx *gin.Context, req UserRequest) error {
    // 只验证指定组的规则
}
```

## 🔐 自动权限控制

### 1. 角色检查
```go
// frame:auth(roles={"ADMIN", "USER"})
func (c *Controller) GetUser(ctx *gin.Context, id uint) (*models.User, error) {
    // 框架自动检查用户角色
    // 只有ADMIN或USER角色才能访问
}
```

### 2. 权限验证
```go
// frame:auth(permissions={"user:read", "user:write"})
func (c *Controller) UpdateUser(ctx *gin.Context, id uint, req UserRequest) error {
    // 框架自动检查用户权限
    // 需要user:read和user:write权限
}
```

### 3. 组合权限
```go
// frame:auth(roles={"ADMIN"}, permissions={"user:delete"})
func (c *Controller) DeleteUser(ctx *gin.Context, id uint) error {
    // 需要ADMIN角色 + user:delete权限
}
```

## 📊 性能优化

### 1. 反射优化
- 使用反射池减少内存分配
- 缓存反射信息提升性能
- 批量处理减少反射调用

### 2. 验证优化
- 延迟验证，只在需要时执行
- 验证结果缓存
- 并行验证支持

### 3. 绑定优化
- 智能类型转换
- 批量参数绑定
- 增量绑定更新

## 🧪 测试示例

### 1. 测试自动参数绑定
```bash
# 测试URI参数绑定
curl -X GET "http://localhost:8080/api/v1/auto-binding/users/123"

# 测试查询参数绑定
curl -X GET "http://localhost:8080/api/v1/auto-binding/users?page=1&page_size=10&role=ADMIN"

# 测试请求体绑定
curl -X POST "http://localhost:8080/api/v1/auto-binding/users" \
  -H "Content-Type: application/json" \
  -d '{"username":"john","email":"john@example.com","password":"123456","role":"USER"}'
```

### 2. 测试自动验证
```bash
# 测试验证失败
curl -X POST "http://localhost:8080/api/v1/auto-binding/users" \
  -H "Content-Type: application/json" \
  -d '{"username":"jo","email":"invalid-email","password":"123","role":"INVALID"}'
```

### 3. 测试权限控制
```bash
# 测试权限不足
curl -X DELETE "http://localhost:8080/api/v1/auto-binding/users/123"
```

## 🔄 与传统方式的对比

### 传统方式（手动解析）
```go
func (c *Controller) CreateUser(ctx *gin.Context) {
    // 手动解析请求体
    var req UserRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 手动验证
    if err := validate.Struct(req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 手动权限检查
    if !hasPermission(ctx, "user:create") {
        ctx.JSON(403, gin.H{"error": "权限不足"})
        return
    }
    
    // 业务逻辑...
}
```

### gRain框架方式（自动绑定）
```go
// frame:route(method="POST", path="/users")
// frame:auth(permissions={"user:create"})
// frame:validate(rules={"username": "required|min:3", "email": "required|email"})
func (c *Controller) CreateUser(ctx *gin.Context, req UserRequest) (*models.User, error) {
    // gRain框架自动完成：
    // 1. 请求体绑定到req
    // 2. 参数验证
    // 3. 权限检查
    
    // 直接使用绑定后的数据
    user := &models.User{
        Username: req.Username,
        Email:    req.Email,
    }
    
    // 业务逻辑...
    return user, nil
}
```

## 🎉 总结

gRain框架的自动参数绑定功能让您：

1. **专注于业务逻辑**：不再需要编写样板代码
2. **提升开发效率**：减少90%的参数处理代码
3. **增强代码质量**：统一的验证和错误处理
4. **简化维护工作**：配置即代码，易于理解和修改

这就是gRain框架的魔力：**让开发者专注于业务逻辑，框架处理所有样板代码！** 🚀 