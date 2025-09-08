# gRain Framework Swagger 集成指南

## 概述

本指南详细介绍了 gRain 框架中 Swagger 文档生成系统的完整实现，包括架构设计、核心组件、使用方法和技术细节。

## 系统架构

### 整体架构图

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Controller    │    │  Annotation      │    │  Swagger        │
│   (Go Code)     │───▶│  Parser          │───▶│  Generator      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │  Registry        │    │  Type Resolver  │
                       │  (Storage)       │    │  (AST Parser)   │
                       └──────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │  OpenAPI Spec   │
                                               │  (JSON Schema)  │
                                               └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │  Swagger UI     │
                                               │  (Web Interface)│
                                               └─────────────────┘
```

### 核心组件

#### 1. SwaggerGenerator
- **职责**: 核心文档生成器，协调各个组件工作
- **功能**: 
  - 解析注解信息
  - 生成 OpenAPI 3.0 规范
  - 管理类型缓存
  - 构建 API 路径和操作

#### 2. TypeResolver
- **职责**: Go 类型到 OpenAPI Schema 的转换器
- **功能**:
  - 解析 Go AST
  - 提取结构体信息
  - 解析字段标签
  - 生成 JSON Schema

#### 3. SwaggerServer
- **职责**: HTTP 服务提供者
- **功能**:
  - 提供 Swagger UI 页面
  - 提供 swagger.json 接口
  - 集成到 Gin 路由系统

#### 4. Registry
- **职责**: 注解信息存储和查询
- **功能**:
  - 存储解析的注解
  - 提供查询接口
  - 支持按类型、目标等分类

## 注解系统详解

### 注解类型

#### 1. 控制器注解
```go
// frame:controller(path="/api/users")
type UserController struct{}
```

**属性**:
- `path`: 控制器基础路径前缀

#### 2. 路由注解
```go
// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:response(200, []UserResponse, "成功获取用户列表")
func (c *UserController) GetUsers(ctx *gin.Context) {
    // 实现逻辑
}
```

**属性**:
- `method`: HTTP 方法
- `path`: 路由路径
- `summary`: API 摘要
- `response`: 响应定义

#### 3. 绑定注解
```go
// frame:bind(source="json", model="CreateUserRequest")
func (c *UserController) CreateUser(ctx *gin.Context) {
    // 实现逻辑
}
```

**属性**:
- `source`: 数据来源 (json, form, query, path)
- `model`: 请求体模型类型

### 结构体标签支持

#### 1. JSON 标签
```go
type User struct {
    ID       uint   `json:"id" example:"1"`
    Username string `json:"username" binding:"required" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
}
```

#### 2. 验证标签
```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50"`
    Age      int    `json:"age" binding:"gte=0,lte=150"`
    Role     string `json:"role" binding:"oneof=USER ADMIN MODERATOR"`
}
```

#### 3. 示例标签
```go
type Product struct {
    ID    uint    `json:"id" example:"1"`
    Name  string  `json:"name" example:"iPhone 15"`
    Price float64 `json:"price" example:"999.99"`
}
```

## 类型解析机制

### 1. 基本类型映射

| Go 类型 | OpenAPI 类型 | 说明 |
|---------|--------------|------|
| `string` | `string` | 字符串类型 |
| `int`, `int64` | `integer` | 整数类型 |
| `float64` | `number` | 数字类型 |
| `bool` | `boolean` | 布尔类型 |
| `time.Time` | `string` | 时间类型，格式为 `date-time` |
| `[]byte` | `string` | 字节数组，格式为 `byte` |

### 2. 复合类型处理

#### 数组类型
```go
type UserList struct {
    Users []User `json:"users"`
}
```

生成:
```json
{
  "type": "object",
  "properties": {
    "users": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/User"
      }
    }
  }
}
```

#### 指针类型
```go
type UserResponse struct {
    User *User `json:"user,omitempty"`
}
```

生成:
```json
{
  "type": "object",
  "properties": {
    "user": {
      "$ref": "#/components/schemas/User"
    }
  }
}
```

### 3. 验证规则映射

| Go 标签 | OpenAPI 特性 | 示例 |
|----------|--------------|------|
| `binding:"required"` | `required: true` | 必填字段 |
| `binding:"min=3,max=50"` | `minLength: 3, maxLength: 50` | 长度限制 |
| `binding:"oneof=USER ADMIN"` | `enum: ["USER", "ADMIN"]` | 枚举值 |
| `binding:"email"` | `format: "email"` | 邮箱格式 |
| `binding:"gte=0"` | `minimum: 0` | 最小值 |
| `binding:"lte=100"` | `maximum: 100` | 最大值 |

## 使用方法

### 1. 基本设置

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/isBlue-5/grain/pkg/annotation/processor"
    "github.com/isBlue-5/grain/pkg/annotation/registry"
)

func main() {
    // 创建 Gin 引擎
    r := gin.Default()
    
    // 创建注解注册表
    registry := registry.NewRegistry()
    
    // 创建 Swagger 生成器
    swaggerGenerator := processor.NewSwaggerGenerator(registry, ".")
    
    // 创建 Swagger 服务器
    swaggerServer := processor.NewSwaggerServer(swaggerGenerator)
    
    // 注册 Swagger 路由
    swaggerServer.RegisterRoutes(r)
    
    // 启动服务器
    r.Run(":8080")
}
```

### 2. 定义控制器

```go
// frame:controller(path="/api/users")
type UserController struct{}

// GetUsers 获取用户列表
// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:response(200, []UserResponse, "成功获取用户列表")
func (c *UserController) GetUsers(ctx *gin.Context) {
    // 实现逻辑
}
```

### 3. 定义模型

```go
type User struct {
    ID       uint   `json:"id" example:"1"`
    Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Role     string `json:"role" binding:"oneof=USER ADMIN" example:"USER"`
}
```

### 4. 注册路由

```go
// 用户 API
users := r.Group("/api/users")
{
    users.GET("/", userController.GetUsers)
    users.GET("/:id", userController.GetUser)
    users.POST("/", userController.CreateUser)
    users.PUT("/:id", userController.UpdateUser)
    users.DELETE("/:id", userController.DeleteUser)
}
```

## 高级特性

### 1. 自定义响应

```go
// frame:response(200, UserResponse, "成功获取用户信息")
// frame:response(404, ErrorResponse, "用户不存在")
// frame:response(500, ErrorResponse, "服务器内部错误")
func (c *UserController) GetUser(ctx *gin.Context) {
    // 实现逻辑
}
```

### 2. 请求体验证

```go
// frame:bind(source="json", model="CreateUserRequest")
func (c *UserController) CreateUser(ctx *gin.Context) {
    var req CreateUserRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    // 处理请求
}
```

### 3. 路径参数

```go
// frame:route(method="GET", path="/:id")
func (c *UserController) GetUser(ctx *gin.Context) {
    id := ctx.Param("id")
    // 处理路径参数
}
```

### 4. 查询参数

```go
// frame:route(method="GET", path="/")
func (c *UserController) GetUsers(ctx *gin.Context) {
    page := ctx.DefaultQuery("page", "1")
    size := ctx.DefaultQuery("size", "10")
    // 处理查询参数
}
```

## 配置选项

### 1. Swagger 信息配置

```go
spec := &OpenAPISpec{
    OpenAPI: "3.0.0",
    Info: &OpenAPIInfo{
        Title:       "My API",
        Description: "API 描述信息",
        Version:     "1.0.0",
    },
    Servers: []*OpenAPIServer{
        {
            URL:         "http://localhost:8080",
            Description: "开发环境",
        },
        {
            URL:         "https://api.example.com",
            Description: "生产环境",
        },
    },
}
```

### 2. 安全配置

```go
Components: &OpenAPIComponents{
    SecuritySchemes: map[string]*OpenAPISecurityScheme{
        "bearerAuth": {
            Type:        "http",
            Scheme:      "bearer",
            Description: "Bearer token认证",
        },
        "apiKey": {
            Type:        "apiKey",
            In:          "header",
            Name:        "X-API-Key",
            Description: "API密钥认证",
        },
    },
}
```

### 3. 标签配置

```go
// 在操作中添加标签
operation := &OpenAPIOperation{
    Tags:        []string{"用户管理", "认证授权"},
    Summary:     "获取用户信息",
    Description: "根据用户ID获取用户详细信息",
}
```

## 性能优化

### 1. 类型缓存

系统实现了类型缓存机制，避免重复解析相同的类型：

```go
type TypeResolver struct {
    typeCache map[string]*OpenAPISchema
    packages  map[string]*ast.Package
}
```

### 2. 包缓存

AST 包信息也会被缓存，减少文件系统访问：

```go
func (tr *TypeResolver) findPackage(packagePath string) (*ast.Package, error) {
    if pkg, exists := tr.packages[packagePath]; exists {
        return pkg, nil
    }
    // 解析并缓存包
}
```

### 3. 延迟解析

只有在实际需要时才解析类型，避免不必要的计算：

```go
func (sg *SwaggerGenerator) parseSchemas(annotations []types.Annotation) (map[string]*OpenAPISchema, error) {
    // 只解析被引用的类型
    for typeName := range referencedTypes {
        schema, err := sg.typeResolver.ResolveType(typeName)
        // ...
    }
}
```

## 扩展性

### 1. 自定义类型支持

可以通过扩展 `TypeResolver` 来支持自定义类型：

```go
func (tr *TypeResolver) parseCustomType(typeName string) *OpenAPISchema {
    switch typeName {
    case "custom.UUID":
        return &OpenAPISchema{
            Type:   "string",
            Format: "uuid",
        }
    case "custom.Decimal":
        return &OpenAPISchema{
            Type:   "number",
            Format: "decimal",
        }
    default:
        return nil
    }
}
```

### 2. 自定义验证规则

可以扩展验证标签的解析逻辑：

```go
func (tr *TypeResolver) parseCustomValidation(schema *OpenAPISchema, tag string) *OpenAPISchema {
    if strings.Contains(tag, "uuid") {
        schema.Format = "uuid"
    }
    if strings.Contains(tag, "phone") {
        schema.Pattern = "^1[3-9]\\d{9}$"
    }
    return schema
}
```

### 3. 插件系统

可以设计插件系统来支持第三方扩展：

```go
type SwaggerPlugin interface {
    Name() string
    Process(spec *OpenAPISpec) error
}

type SwaggerGenerator struct {
    plugins []SwaggerPlugin
}

func (sg *SwaggerGenerator) AddPlugin(plugin SwaggerPlugin) {
    sg.plugins = append(sg.plugins, plugin)
}
```

## 最佳实践

### 1. 注解组织

```go
// 将相关注解分组，提高可读性
// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:bind(source="json", model="CreateUserRequest")
// frame:response(201, UserResponse, "用户创建成功")
// frame:response(400, ErrorResponse, "请求参数错误")
// frame:response(409, ErrorResponse, "用户已存在")
func (c *UserController) CreateUser(ctx *gin.Context) {
    // 实现逻辑
}
```

### 2. 模型设计

```go
// 使用清晰的字段命名
// 添加适当的验证规则
// 提供有意义的示例值
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Age      int    `json:"age" binding:"gte=0,lte=150" example:"25"`
    Role     string `json:"role" binding:"oneof=USER ADMIN MODERATOR" example:"USER"`
}
```

### 3. 错误处理

```go
// 定义清晰的错误响应
// frame:response(400, ValidationError, "请求参数验证失败")
// frame:response(401, AuthError, "认证失败")
// frame:response(403, PermissionError, "权限不足")
// frame:response(404, NotFoundError, "资源不存在")
// frame:response(500, ServerError, "服务器内部错误")
```

### 4. 文档维护

```go
// 保持注解与代码同步
// 定期更新示例值
// 及时添加新的响应类型
// 维护清晰的API分组
```

## 故障排除

### 常见问题

#### 1. 注解不生效
**症状**: Swagger 文档中没有显示预期的 API
**原因**: 注解格式错误或解析失败
**解决**: 检查注解语法，确保格式正确

#### 2. 类型解析失败
**症状**: 控制台出现类型解析警告
**原因**: 结构体定义不完整或路径错误
**解决**: 检查结构体定义，确保包路径正确

#### 3. Swagger UI 无法访问
**症状**: 浏览器无法访问 Swagger 页面
**原因**: 路由注册失败或端口冲突
**解决**: 检查路由注册，确认端口配置

### 调试技巧

#### 1. 查看生成的 JSON
```bash
curl http://localhost:8080/swagger/swagger.json
```

#### 2. 检查控制台日志
- 查看类型解析警告
- 检查路由注册状态
- 监控性能指标

#### 3. 验证注解语法
- 确保注解格式正确
- 检查属性值格式
- 验证类型引用

## 总结

gRain 框架的 Swagger 集成系统提供了一个完整的、自动化的 API 文档生成解决方案。通过注解驱动的方式，开发者可以专注于业务逻辑的实现，而文档生成则完全自动化。

系统的核心优势包括：

1. **完全自动化**: 无需手动编写 OpenAPI 规范
2. **类型安全**: 基于 Go 类型系统，确保文档准确性
3. **实时更新**: 运行时动态生成，与代码保持同步
4. **高度可扩展**: 支持自定义类型和验证规则
5. **性能优化**: 实现了缓存和延迟解析机制

通过遵循本指南的最佳实践，开发者可以构建出高质量、易维护的 API 文档系统。 