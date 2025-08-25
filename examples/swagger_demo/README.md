# gRain Framework Swagger 集成示例

这个示例展示了如何在 gRain 框架中集成 Swagger 文档生成功能，实现自动化的 API 文档生成。

## 功能特性

- **自动注解解析**: 解析 `frame:` 注解，自动生成 OpenAPI 3.0 规范
- **智能类型解析**: 自动解析 Go 结构体为 OpenAPI Schema
- **实时文档**: 运行时动态生成 Swagger 文档
- **美观的 UI**: 提供现代化的 Swagger UI 界面
- **完整示例**: 包含用户管理和产品管理的完整 CRUD 操作

## 注解系统

### 控制器注解

```go
// frame:controller(path="/api/users")
type UserController struct{}
```

- `path`: 控制器的基础路径前缀

### 路由注解

```go
// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:response(200, []UserResponse, "成功获取用户列表")
func (c *UserController) GetUsers(ctx *gin.Context) {
    // 实现逻辑
}
```

- `method`: HTTP 方法 (GET, POST, PUT, DELETE, PATCH)
- `path`: 路由路径
- `summary`: API 功能摘要
- `response`: 响应定义，格式为 `(状态码, 响应类型, 描述)`

### 绑定注解

```go
// frame:bind(source="json", model="CreateUserRequest")
func (c *UserController) CreateUser(ctx *gin.Context) {
    // 实现逻辑
}
```

- `source`: 数据来源 (json, form, query, path)
- `model`: 请求体模型类型

### 结构体标签

```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Password string `json:"password" binding:"required,min=6" example:"password123"`
    Role     string `json:"role" binding:"oneof=USER ADMIN" example:"USER"`
}
```

支持的标签：
- `json`: JSON 序列化标签
- `binding`: 验证规则标签
- `example`: 示例值标签
- `validate`: 验证规则标签

## 快速开始

### 1. 安装依赖

```bash
go mod init swagger_demo
go get github.com/gin-gonic/gin
go get github.com/isBlue-5/grain
```

### 2. 运行示例

```bash
go run main.go
```

### 3. 访问文档

- **Swagger UI**: http://localhost:8080/swagger
- **Swagger JSON**: http://localhost:8080/swagger/swagger.json

## API 端点

### 用户管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/users/` | 获取用户列表 |
| GET | `/api/users/:id` | 获取单个用户 |
| POST | `/api/users/` | 创建新用户 |
| PUT | `/api/users/:id` | 更新用户信息 |
| DELETE | `/api/users/:id` | 删除用户 |

### 产品管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/products/` | 获取产品列表 |
| GET | `/api/products/:id` | 获取单个产品 |
| POST | `/api/products/` | 创建新产品 |

## 自定义和扩展

### 添加新的控制器

```go
// frame:controller(path="/api/orders")
type OrderController struct{}

// frame:route(method="GET", path="/")
// frame:summary(获取订单列表)
func (c *OrderController) GetOrders(ctx *gin.Context) {
    // 实现逻辑
}
```

### 添加新的模型

```go
type Order struct {
    ID     uint      `json:"id" example:"1"`
    UserID uint      `json:"user_id" binding:"required" example:"1"`
    Total  float64   `json:"total" binding:"required,gt=0" example:"99.99"`
    Status string    `json:"status" binding:"oneof=PENDING PAID CANCELLED" example:"PENDING"`
}
```

### 自定义响应

```go
// frame:response(200, Order, "订单获取成功")
// frame:response(404, gin.H, "订单不存在")
// frame:response(500, gin.H, "服务器内部错误")
```

## 高级特性

### 1. 类型自动解析

系统会自动解析以下 Go 类型：

- **基本类型**: `string`, `int`, `float64`, `bool`
- **时间类型**: `time.Time`
- **数组类型**: `[]Type`
- **指针类型**: `*Type`
- **自定义类型**: 自动解析为引用

### 2. 验证规则映射

| Go 标签 | OpenAPI 特性 |
|----------|--------------|
| `binding:"required"` | `required: true` |
| `binding:"min=3,max=50"` | `minLength: 3, maxLength: 50` |
| `binding:"oneof=USER ADMIN"` | `enum: ["USER", "ADMIN"]` |
| `binding:"email"` | `format: "email"` |

### 3. 示例值支持

```go
type User struct {
    ID       uint   `json:"id" example:"1"`
    Username string `json:"username" example:"johndoe"`
    Email    string `json:"email" example:"john@example.com"`
}
```

## 配置选项

### Swagger 信息配置

```go
spec := &OpenAPISpec{
    OpenAPI: "3.0.0",
    Info: &OpenAPIInfo{
        Title:       "My API",
        Description: "API 描述",
        Version:     "1.0.0",
    },
    Servers: []*OpenAPIServer{
        {
            URL:         "http://localhost:8080",
            Description: "开发环境",
        },
    },
}
```

### 安全配置

```go
Components: &OpenAPIComponents{
    SecuritySchemes: map[string]*OpenAPISecurityScheme{
        "bearerAuth": {
            Type:        "http",
            Description: "Bearer token认证",
        },
    },
}
```

## 故障排除

### 常见问题

1. **注解不生效**
   - 确保注解格式正确
   - 检查包导入路径

2. **类型解析失败**
   - 确保结构体定义完整
   - 检查字段标签格式

3. **Swagger UI 无法访问**
   - 检查路由注册
   - 确认端口配置

### 调试技巧

1. **查看生成的 JSON**
   ```
   curl http://localhost:8080/swagger/swagger.json
   ```

2. **检查控制台日志**
   - 查看类型解析警告
   - 检查路由注册状态

3. **验证注解语法**
   - 确保注解格式正确
   - 检查属性值格式

## 最佳实践

### 1. 注解组织

```go
// 将相关注解分组
// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:bind(source="json", model="CreateUserRequest")
// frame:response(201, UserResponse, "用户创建成功")
// frame:response(400, gin.H, "请求参数错误")
func (c *UserController) CreateUser(ctx *gin.Context) {
    // 实现逻辑
}
```

### 2. 模型设计

```go
// 使用清晰的字段命名
// 添加适当的验证规则
// 提供有意义的示例值
type UserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Age      int    `json:"age" binding:"gte=0,lte=150" example:"25"`
}
```

### 3. 错误处理

```go
// 定义清晰的错误响应
// frame:response(400, gin.H, "请求参数错误")
// frame:response(404, gin.H, "资源不存在")
// frame:response(500, gin.H, "服务器内部错误")
```

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目！

## 许可证

本项目采用 MIT 许可证。 